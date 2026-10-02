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

func TestQuotaCountsSelectedIranUserDirectionOnEitherRole(t *testing.T) {
	write := func(dir, name, role string, in, out uint64) {
		t.Helper()
		c := metrics.NewCollector(dir, name, "tcp", role, func() uint64 { return in }, func() uint64 { return out })
		if err := c.Write(); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name, role, mode string
		in, out          uint64
		want             bool
	}{
		{"iran download uses incoming", "server", "download", 1 << 30, 0, true},
		{"iran upload uses outgoing", "server", "upload", 0, 1 << 30, true},
		{"kharej download uses outgoing", "client", "download", 0, 1 << 30, true},
		{"kharej upload uses incoming", "client", "upload", 1 << 30, 0, true},
		{"unselected side does not exhaust", "client", "download", 1 << 30, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "edge.toml")
			write(dir, "edge", tc.role, tc.in, tc.out)
			paused, err := quotaExhausted(path, &config.Config{TrafficLimitGB: 1, TrafficLimitMode: tc.mode})
			if err != nil || paused != tc.want {
				t.Fatalf("paused=%v err=%v want %v", paused, err, tc.want)
			}
		})
	}
}

func TestQuotaGuardWaitsForSelectedDirection(t *testing.T) {
	dir := t.TempDir()
	var in, out atomic.Uint64
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := metrics.NewCollector(dir, "kharej", "tcp", "client", in.Load, out.Load)
	go (&quotaGuard{limit: 1 << 30, mode: "download", cancel: cancel}).watch(ctx, c)
	in.Store(1 << 30) // client-side incoming is upload from the Iran user's view
	select {
	case <-ctx.Done():
		t.Fatal("quota stopped on the unselected direction")
	case <-time.After(350 * time.Millisecond):
	}
	out.Store(1 << 30) // client-side outgoing is the user's download
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("quota did not stop on the selected direction")
	}
}

func TestResetTrafficRenewsTheSameQuota(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "iran.toml")
	var bytes atomic.Uint64
	bytes.Store(1 << 30)
	collector := metrics.NewCollector(dir, "iran", "tcp", "server", bytes.Load, nil)
	if err := collector.Write(); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{TrafficLimitGB: 1}
	if paused, err := quotaExhausted(path, cfg); err != nil || !paused {
		t.Fatalf("full quota = %v, %v; want paused", paused, err)
	}
	// Systemd has stopped the original collector before this reset.
	if err := metrics.ResetTraffic(dir, "iran"); err != nil {
		t.Fatal(err)
	}
	if paused, err := quotaExhausted(path, cfg); err != nil || paused {
		t.Fatalf("reset quota = %v, %v; want available", paused, err)
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
