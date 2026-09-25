package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/backpack/backpack/cmd"
	"github.com/backpack/backpack/internal/metrics"
)

// Exercise the installed engine's reverse path, not just its quota helper:
// real forwarded traffic must close the entry listener at the allowance, and
// editing the config must make it carry traffic again without erasing usage.
func TestReverseQuotaPausesAndResumesRealTraffic(t *testing.T) {
	backend := startEchoBackend(t)
	tunnelPort, entryPort := freePort(t), freePort(t)
	dir := scratchDir(t)
	serverPath := filepath.Join(dir, "server.toml")
	clientPath := filepath.Join(dir, "client.toml")
	const gib = uint64(1 << 30)
	seed := metrics.Snapshot{Name: "server", BytesIn: gib - 128}
	data, err := json.Marshal(seed)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metrics.Path(dir, "server"), data, 0600); err != nil {
		t.Fatal(err)
	}
	serverConfig := func(gb int) string {
		return fmt.Sprintf("traffic_limit_gb = %d\n", gb) + serverToml(tunnelPort, entryPort, backend.addr)
	}
	writeTunnelConfig(t, serverPath, serverConfig(1))
	writeTunnelConfig(t, clientPath, clientToml(tunnelPort))

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); cmd.Run(serverPath, ctx) }()
	time.Sleep(300 * time.Millisecond)
	go func() { defer wg.Done(); cmd.Run(clientPath, ctx) }()
	t.Cleanup(func() { cancel(); wg.Wait() })
	entry := fmt.Sprintf("127.0.0.1:%d", entryPort)
	if err := echoThrough(entry, tunnelReadyTimeout); err != nil {
		t.Fatalf("reverse tunnel did not start below its quota: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for !portFree(entryPort) && time.Now().Before(deadline) {
		_ = oneEcho(entry)
		time.Sleep(80 * time.Millisecond)
	}
	if !portFree(entryPort) {
		t.Fatal("the forwarded listener stayed open after the traffic allowance was exhausted")
	}
	paused, err := metrics.Read(dir, "server")
	if err != nil || paused.BytesIn+paused.BytesOut < gib {
		t.Fatalf("paused counters = %d in, %d out, %v; want at least 1 GiB total",
			paused.BytesIn, paused.BytesOut, err)
	}
	// An exhausted service must remain paused until the quota changes. A
	// systemd or watchdog restart must not inadvertently reset that decision.
	time.Sleep(3 * time.Second)
	if !portFree(entryPort) {
		t.Fatal("the listener reopened without increasing the quota")
	}

	writeTunnelConfig(t, serverPath, serverConfig(2))
	if err := echoThrough(entry, 25*time.Second); err != nil {
		t.Fatalf("raising the quota did not resume the reverse tunnel: %v", err)
	}
	resumed, err := metrics.Read(dir, "server")
	if err != nil || resumed.BytesIn+resumed.BytesOut < paused.BytesIn+paused.BytesOut {
		t.Fatalf("usage went backwards after resuming: paused %+v, resumed %+v, error %v",
			paused, resumed, err)
	}
}
