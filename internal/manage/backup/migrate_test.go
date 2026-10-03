package backup

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/metrics"
)

func TestServerIPMigrationLeavesClientsSpoofAndAccountingIntact(t *testing.T) {
	live, stage := t.TempDir(), t.TempDir()
	server := "traffic_limit_gb = 90\ntraffic_limit_mode = \"download\"\n[server]\nbind_addr = \"198.51.100.8:9000\"\ntransport = \"tcp\"\ntoken = \"private-token\"\nports = [\"198.51.100.8:443=127.0.0.1:2096\", \"198.51.100.8:10000-10009=198.51.100.9:8080\", \"127.0.0.1:444\", \"53835\"]\nfuture_setting = \"keep\"\n"
	client := "traffic_limit_gb = 30\n[client]\nremote_addr = \"198.51.100.8:9000\"\nlocal_addr = \"198.51.100.8\"\ntoken = \"client-token\"\n"
	dial := "[l3]\nmode = \"dial\"\naddr = \"198.51.100.8:9001\"\ncarrier = \"spoof\"\nspoof_src_ip = \"81.28.60.1\"\nspoof_peer_src_ip = \"81.28.60.2\"\n"
	listen := "[l3]\nmode = \"listen\"\naddr = \"198.51.100.8:9002\"\ncarrier = \"spoof\"\nlocal_ip = \"10.20.0.2/30\"\npeer_ip = \"10.20.0.1\"\nspoof_src_ip = \"81.28.60.1\"\nspoof_src_pool = [\"81.28.60.1\",\"81.28.60.3\"]\nspoof_peer_src_ip = \"81.28.60.2\"\nspoof_peer_ip = \"198.51.100.19\"\n"
	accounting := map[string]string{
		"edge.metrics.json": `{"name":"edge","bytes_in":123456789,"bytes_out":987654321}`,
		"history.json":      `{"tunnels":{"edge":{"recent":[{"t":1700000000,"in":123456789,"out":987654321,"up":true}]}}}`,
		"webui.json":        `{"password_hash":"saved","customer_links":{"private-id":"edge"},"traffic":"keep"}`,
		"meta.json":         `{"edge":{"country":"TR"}}`,
	}
	entries := []entry{{name: "edge.toml", body: server}, {name: "client.toml", body: client}, {name: "dial.toml", body: dial}, {name: "listen.toml", body: listen},
		{name: "wildcard.toml", body: "[server]\nbind_addr = \"0.0.0.0:9003\"\n"},
		{name: "direct.toml", body: "[direct]\nrole = \"kharej\"\naddr = \"198.51.100.8:9004\"\n"},
		{name: "direct-client.toml", body: "[direct]\nrole = \"iran\"\naddr = \"198.51.100.8:9004\"\n"}}
	for name, body := range accounting {
		entries = append(entries, entry{name: name, body: body})
	}
	if err := os.WriteFile(filepath.Join(live, "unarchived.toml"), []byte(server), 0600); err != nil {
		t.Fatal(err)
	}
	contents, err := stageRestore(bytes.NewReader(archiveOf(t, entries...)), live, stage)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := migrateServerIPs(stage, contents.TunnelFiles, "203.0.113.20")
	if err != nil || len(changed) != 3 {
		t.Fatalf("migration: %v %v", changed, err)
	}
	for name, body := range map[string]string{"client.toml": client, "dial.toml": dial, "unarchived.toml": server} {
		assertStaged(t, stage, name, body)
	}
	for name, body := range accounting {
		assertStaged(t, stage, name, body)
	}
	var s, l config.Config
	if _, err := toml.DecodeFile(filepath.Join(stage, "edge.toml"), &s); err != nil {
		t.Fatal(err)
	}
	if s.Server.BindAddr != "203.0.113.20:9000" || s.TrafficLimitGB != 90 || s.TrafficLimitMode != "download" || s.Server.Token != "private-token" {
		t.Fatalf("server settings altered: %+v", s)
	}
	wantPorts := []string{"203.0.113.20:443=127.0.0.1:2096", "203.0.113.20:10000-10009=198.51.100.9:8080", "127.0.0.1:444", "53835"}
	if !reflect.DeepEqual(s.Server.Ports, wantPorts) {
		t.Fatalf("ports: %v", s.Server.Ports)
	}
	if _, err := toml.DecodeFile(filepath.Join(stage, "listen.toml"), &l); err != nil {
		t.Fatal(err)
	}
	if l.L3.Addr != "203.0.113.20:9002" || l.L3.SpoofSrcIP != "81.28.60.1" || l.L3.SpoofPeerSrcIP != "81.28.60.2" || l.L3.SpoofPeerIP != "198.51.100.19" || l.L3.LocalIP != "10.20.0.2/30" || len(l.L3.SpoofSrcPool) != 2 {
		t.Fatalf("spoof/private addressing corrupted: %+v", l.L3)
	}
	// The fresh process uses exactly the backup's cumulative traffic baseline.
	snap := metrics.NewCollector(stage, "edge", "tcp", "server", nil, nil).Snapshot()
	if snap.BytesIn != 123456789 || snap.BytesOut != 987654321 {
		t.Fatalf("traffic reset: %d/%d", snap.BytesIn, snap.BytesOut)
	}
	// A destination writer's final flush must not overwrite archived usage;
	// a tunnel absent from the backup keeps its latest destination totals.
	if err := os.WriteFile(filepath.Join(live, "edge.metrics.json"), []byte("destination old counters"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(live, "untouched.metrics.json"), []byte("latest destination counters"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := seedStageExcept(live, stage, contents.ArchiveFiles); err != nil {
		t.Fatal(err)
	}
	assertStaged(t, stage, "edge.metrics.json", accounting["edge.metrics.json"])
	assertStaged(t, stage, "untouched.metrics.json", "latest destination counters")
}

func TestMigrateBindKeepsWildcardsLoopbackPortsAndHandlesIPv6(t *testing.T) {
	for _, tc := range []struct{ old, want string }{
		{"[2001:db8::8]:443", "[2001:db8::20]:443"}, {"0.0.0.0:443", "0.0.0.0:443"}, {"[::]:443", "[::]:443"}, {"127.0.0.1:443", "127.0.0.1:443"}, {"443-450", "443-450"}} {
		got, err := migrateBindAddress(tc.old, "2001:db8::20")
		if err != nil || got != tc.want {
			t.Fatalf("%s -> %s: %v", tc.old, got, err)
		}
	}
	if _, err := migrateBindAddress("bad:address:443", "203.0.113.20"); err == nil {
		t.Fatal("malformed listener accepted")
	}
	if _, err := migrateServerIPs(t.TempDir(), nil, "not-an-ip"); err == nil {
		t.Fatal("invalid destination accepted")
	}
}

func TestRestorePausesAllWritersAndResumesOnFailure(t *testing.T) {
	for _, failStop := range []bool{false, true} {
		var calls []string
		run := func(args ...string) (string, error) {
			calls = append(calls, strings.Join(args, " "))
			if args[0] == "is-active" {
				return "active", nil
			}
			if failStop && args[0] == "stop" && args[1] == "panel" {
				return "", errors.New("stop failed")
			}
			return "", nil
		}
		resume, err := pauseRestoreWriters([]string{"monitor", "panel", "tunnel"}, run)
		if (err != nil) != failStop {
			t.Fatalf("pause: %v", err)
		}
		if failed := resume(); len(failed) != 0 {
			t.Fatal(failed)
		}
		if calls[len(calls)-1] != "start monitor" {
			t.Fatal("monitor resumed before tunnel writers")
		}
		if failStop && strings.Contains(strings.Join(calls, ";"), "stop tunnel") {
			t.Fatal("continued after stop failure")
		}
	}
	resume, err := pauseRestoreWriters([]string{"inactive"}, func(args ...string) (string, error) {
		if args[0] == "is-active" {
			return "inactive", errors.New("exit 3")
		}
		return "", fmt.Errorf("must not start")
	})
	if err != nil || len(resume()) != 0 {
		t.Fatal("started a previously inactive service")
	}
	resume, err = pauseRestoreWriters([]string{"panel"}, func(args ...string) (string, error) {
		if args[0] == "is-active" {
			return "active", nil
		}
		if args[0] == "start" {
			return "", errors.New("start failed")
		}
		return "", nil
	})
	if err != nil || len(resume()) != 1 {
		t.Fatal("resume failure was hidden")
	}
}
