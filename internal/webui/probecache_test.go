package webui

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPanelTabsShareAColdProbeAndUseStaleAnswerWhileRefreshing(t *testing.T) {
	var c probeCache
	var calls atomic.Int32
	firstStarted := make(chan struct{})
	firstDone := make(chan struct{})
	measure := func() int {
		calls.Add(1)
		close(firstStarted)
		<-firstDone
		return 42
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := c.get("tunnel-a", measure); got != 42 {
				t.Errorf("cold probe = %d, want 42", got)
			}
		}()
	}
	<-firstStarted
	close(firstDone)
	wg.Wait()
	if got := calls.Load(); got != 1 {
		t.Fatalf("simultaneous tabs ran %d probes, want one", got)
	}

	c.mu.Lock()
	c.entries["tunnel-a"].at = time.Now().Add(-time.Minute)
	c.mu.Unlock()
	refresh := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once
	refreshMeasure := func() int {
		once.Do(func() { close(started) })
		<-refresh
		return 75
	}
	if got := c.get("tunnel-a", refreshMeasure); got != 42 {
		t.Fatalf("refresh blocked the panel or discarded the last reading: %d", got)
	}
	<-started
	if got := c.get("tunnel-a", refreshMeasure); got != 42 {
		t.Fatalf("second poll blocked during refresh: %d", got)
	}
	close(refresh)
	deadline := time.Now().Add(time.Second)
	for {
		c.mu.Lock()
		got := c.entries["tunnel-a"].value
		c.mu.Unlock()
		if got == 75 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("background refresh never published its result")
		}
		time.Sleep(time.Millisecond)
	}
}
