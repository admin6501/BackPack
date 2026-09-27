package metrics

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestResetTrafficStartsNewPeriodAndPersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	var carried atomic.Uint64
	carried.Store(900)
	c := NewCollector(dir, "tunnel", "tcp", "server", carried.Load, nil)
	if err := c.Write(); err != nil {
		t.Fatal(err)
	}
	// The service has stopped, so its final snapshot is on disk. The next
	// process has fresh counters, just as it does after a systemd restart.
	if err := ResetTraffic(dir, "tunnel"); err != nil {
		t.Fatal(err)
	}
	var newBytes atomic.Uint64
	next := NewCollector(dir, "tunnel", "tcp", "server", newBytes.Load, nil)
	if got := next.Snapshot().BytesIn; got != 0 {
		t.Fatalf("after reset: %d bytes, want zero", got)
	}
	newBytes.Store(17)
	if err := next.Write(); err != nil {
		t.Fatal(err)
	}
	last := NewCollector(dir, "tunnel", "tcp", "server", nil, nil)
	if got := last.Snapshot().BytesIn; got != 17 {
		t.Fatalf("new usage after restart: %d, want 17", got)
	}
}

func TestResetTrafficDoesNotEraseDamagedHistory(t *testing.T) {
	dir := t.TempDir()
	path := Path(dir, "tunnel")
	if err := os.WriteFile(path, []byte("broken"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ResetTraffic(dir, "tunnel"); err == nil {
		t.Fatal("damaged history was silently reset")
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "broken" {
		t.Fatalf("damaged history changed: %q, %v", got, err)
	}
	if err := ResetTraffic(dir, "new"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "new.metrics.json")); err != nil {
		t.Fatal(err)
	}
}
