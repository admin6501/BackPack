package webui

import (
	"sync"
	"time"
)

const probeFresh = 10 * time.Second

type probeResult struct {
	value int
	at    time.Time
	busy  bool
	ready chan struct{}
}

type probeCache struct {
	mu      sync.Mutex
	entries map[string]*probeResult
}

var tunnelProbes probeCache

// A cold key is measured once; later polls use the last result while one
// background refresh checks the route. Simultaneous browser tabs share both.
func (c *probeCache) get(key string, measure func() int) int {
	c.mu.Lock()
	if c.entries == nil {
		c.entries = make(map[string]*probeResult)
	}
	e := c.entries[key]
	if e == nil {
		e = &probeResult{ready: make(chan struct{})}
		c.entries[key] = e
		c.mu.Unlock()
		v := measure()
		c.mu.Lock()
		e.value, e.at = v, time.Now()
		close(e.ready)
		e.ready = nil
		c.mu.Unlock()
		return v
	}
	if e.ready != nil {
		ready := e.ready
		c.mu.Unlock()
		<-ready
		c.mu.Lock()
		v := e.value
		c.mu.Unlock()
		return v
	}
	v := e.value
	if !e.busy && time.Since(e.at) >= probeFresh {
		e.busy = true
		go func() {
			next := measure()
			c.mu.Lock()
			e.value, e.at, e.busy = next, time.Now(), false
			c.mu.Unlock()
		}()
	}
	c.mu.Unlock()
	return v
}

func tcpPingCached(host, port string) int {
	return tunnelProbes.get("tcp\x00"+host+"\x00"+port, func() int { return tcpPing(host, port) })
}

func icmpPingCached(ip string) int {
	return tunnelProbes.get("icmp\x00"+ip, func() int { return icmpPing(ip) })
}
