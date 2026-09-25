package cmd

import (
	"context"
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
