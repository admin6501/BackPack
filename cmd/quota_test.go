package cmd

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/metrics"
)

func TestQuotaPersistsAcrossRestartAndStopsTraffic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "iran.toml")
	var bytes atomic.Uint64
	bytes.Store(1<<30 - 20)
	seed := metrics.NewCollector(dir, "iran", "tcp", "server", bytes.Load, nil)
	if err := seed.Write(); err != nil {
		t.Fatal(err)
	}
	bytes.Store(0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	restarted := metrics.NewCollector(dir, "iran", "tcp", "server", bytes.Load, nil)
	go (&quotaGuard{limit: 1 << 30, cancel: cancel}).watch(ctx, restarted)
	bytes.Store(30)
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("quota did not stop the tunnel")
	}
	paused, err := quotaExhausted(path, &config.Config{TrafficLimitGB: 1})
	if err != nil || !paused {
		t.Fatalf("quota after restart = %v, %v; want paused", paused, err)
	}
	paused, err = quotaExhausted(path, &config.Config{TrafficLimitGB: 2})
	if err != nil || paused {
		t.Fatalf("raised quota = %v, %v; want running", paused, err)
	}
}

func TestQuotaRefusesInvalidAndCorruptedHistory(t *testing.T) {
	if _, err := trafficQuotaBytes(-1); err == nil {
		t.Fatal("negative quota accepted")
	}
	if _, err := trafficQuotaBytes(1 << 35); err == nil {
		t.Fatal("overflowing quota accepted")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "iran.toml")
	if paused, err := quotaExhausted(path, &config.Config{TrafficLimitGB: 1}); err != nil || paused {
		t.Fatalf("brand-new tunnel = %v, %v", paused, err)
	}
}

// The quota guard must wake the same generation the reload loop watches.
// After the cap is raised, that loop must leave the paused state on its own.
func TestQuotaPauseAndResumeOnConfigChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "iran.toml")
	if err := os.WriteFile(path, []byte("traffic_limit_gb = 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	applyDefaults(cfg)
	var bytes atomic.Uint64
	bytes.Store(1 << 30)
	gen, cancel := context.WithCancel(context.Background())
	defer cancel()
	guard := &quotaGuard{limit: 1 << 30, cancel: cancel}
	collector := metrics.NewCollector(dir, "iran", "tcp", "server", bytes.Load, nil)
	go guard.watch(gen, collector)
	root, stop := context.WithTimeout(context.Background(), 4*time.Second)
	defer stop()
	if _, why := awaitConfigChange(root, gen, path, cfg); why != wakeRestart {
		t.Fatalf("quota did not wake running generation: %v", why)
	}
	if paused, err := quotaExhausted(path, cfg); err != nil || !paused {
		t.Fatalf("generation was not paused: %v, %v", paused, err)
	}
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = os.WriteFile(path, []byte("traffic_limit_gb = 2\n"), 0600)
	}()
	next, why := awaitConfigChange(root, root, path, cfg)
	if why != wakeConfig || next.TrafficLimitGB != 2 {
		t.Fatalf("raising quota did not resume watcher: %v, %v", why, next)
	}
	if paused, err := quotaExhausted(path, next); err != nil || paused {
		t.Fatalf("raised quota still paused: %v, %v", paused, err)
	}
}

func TestMetricsFinalWriteCompletesBeforeGenerationEnds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "iran.toml")
	var bytes atomic.Uint64
	ctx, cancel := context.WithCancel(context.Background())
	finish := startMetricsWithTraffic(ctx, path, "tcp", "server", bytes.Load, nil)
	bytes.Store(27)
	cancel()
	// Bytes can finish during the engine's shutdown, after its context is
	// cancelled and after the periodic writer made its final snapshot.
	bytes.Store(29)
	finish()
	snap, err := metrics.Read(dir, "iran")
	if err != nil || snap.BytesIn != 29 {
		t.Fatalf("final metrics = %v, %v; want 29 bytes", snap.BytesIn, err)
	}
}
