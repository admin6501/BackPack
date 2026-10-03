package backup

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRestorePrunesOnlyOrphanAccounting(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"live.toml":         "invalid config retained conservatively",
		"live.metrics.json": "123456789",
		"gone.metrics.json": "old",
		"history/live.json": "live config history",
		"history/gone.json": "old config history",
		"history.json":      `{"updated":"unchanged","tunnels":{"live":{"recent":[{"in":123456789}]},"gone":{"recent":[]}},"future":{"keep":true}}`,
		"webui.json":        "customer data",
	}
	for name, data := range files {
		path := filepath.Join(root, name)
		os.MkdirAll(filepath.Dir(path), 0755)
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := pruneOrphanAccounting(root); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"gone.metrics.json", "history/gone.json"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("orphan retained: %s", name)
		}
	}
	for _, name := range []string{"live.toml", "live.metrics.json", "history/live.json", "webui.json"} {
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(b) != files[name] {
			t.Fatalf("active data changed: %s", name)
		}
	}
	b, _ := os.ReadFile(filepath.Join(root, "history.json"))
	var doc map[string]json.RawMessage
	json.Unmarshal(b, &doc)
	var hist map[string]json.RawMessage
	json.Unmarshal(doc["tunnels"], &hist)
	if _, ok := hist["gone"]; ok {
		t.Fatal("deleted tunnel traffic history retained")
	}
	var live, future bytes.Buffer
	json.Compact(&live, hist["live"])
	json.Compact(&future, doc["future"])
	if live.String() != `{"recent":[{"in":123456789}]}` || future.String() != `{"keep":true}` {
		t.Fatal("live usage or unknown fields changed")
	}
	if !orphanAccounting(root, "history/gone.json") || orphanAccounting(root, "history/live.json") || orphanAccounting(root, "webui.json") {
		t.Fatal("incorrect archive filtering")
	}
}

func TestBackupExcludesDeletedTunnelFilesAndRestoreKeepsTotals(t *testing.T) {
	root, stage := t.TempDir(), t.TempDir()
	usage := `{"name":"live","bytes_in":123456789,"bytes_out":987654321}`
	for name, data := range map[string]string{"live.toml": "[server]\nbind_addr = \"0.0.0.0:443\"\n", "live.metrics.json": usage, "gone.metrics.json": "orphan", "history/gone.json": "orphan"} {
		path := filepath.Join(root, name)
		os.MkdirAll(filepath.Dir(path), 0755)
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var archive bytes.Buffer
	if err := writeBackupTree(&archive, root); err != nil {
		t.Fatal(err)
	}
	empty := t.TempDir()
	if _, err := stageRestore(bytes.NewReader(archive.Bytes()), empty, stage); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"gone.metrics.json", "history/gone.json"} {
		if _, err := os.Stat(filepath.Join(stage, name)); !os.IsNotExist(err) {
			t.Fatal("orphan entered archive", name)
		}
	}
	b, _ := os.ReadFile(filepath.Join(stage, "live.metrics.json"))
	if string(b) != usage {
		t.Fatal("saved totals changed")
	}
}
