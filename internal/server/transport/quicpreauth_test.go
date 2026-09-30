package transport

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
)

func TestQUICPreAuthLimiterBoundsConnectionsPerPeerAndGlobally(t *testing.T) {
	limiter := newQUICPreAuthLimiter(2, 1)
	first, ok := limiter.acquire(&net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 1001})
	if !ok {
		t.Fatal("first peer was refused")
	}
	if _, ok := limiter.acquire(&net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 1002}); ok {
		t.Fatal("second pending connection from the same peer was accepted")
	}
	second, ok := limiter.acquire(&net.UDPAddr{IP: net.ParseIP("192.0.2.2"), Port: 1003})
	if !ok {
		t.Fatal("different peer was refused before the global limit")
	}
	if _, ok := limiter.acquire(&net.UDPAddr{IP: net.ParseIP("192.0.2.3"), Port: 1004}); ok {
		t.Fatal("connection beyond the global limit was accepted")
	}

	gate := &quicGate{pending: make(chan struct{}, 1), lease: first}
	gate.settled(true, false) // A valid token proof releases the pending slot.
	first.release()            // Closing the connection later must not release it twice.
	third, ok := limiter.acquire(&net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 1005})
	if !ok {
		t.Fatal("peer slot was not released after authentication")
	}
	second.release()
	third.release()
}

func TestQUICPreAuthLimiterGroupsIPv6PrivacyAddresses(t *testing.T) {
	limiter := newQUICPreAuthLimiter(4, 1)
	lease, ok := limiter.acquire(&net.UDPAddr{IP: net.ParseIP("2001:db8:1:2::1"), Port: 1001})
	if !ok {
		t.Fatal("first IPv6 peer was refused")
	}
	defer lease.release()
	if _, ok := limiter.acquire(&net.UDPAddr{IP: net.ParseIP("2001:db8:1:2::abcd"), Port: 1002}); ok {
		t.Fatal("privacy address in the same /64 bypassed the peer limit")
	}
}

func TestQUICPreAuthLimiterIsSafeUnderConcurrentAdmissions(t *testing.T) {
	const limit = 8
	limiter := newQUICPreAuthLimiter(limit, limit)
	start := make(chan struct{})
	leases := make(chan *quicPreAuthLease, 64)
	var admitted atomic.Int32
	var workers sync.WaitGroup
	for i := 0; i < cap(leases); i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			if lease, ok := limiter.acquire(&net.UDPAddr{IP: net.ParseIP("198.51.100.1"), Port: 2000}); ok {
				admitted.Add(1)
				leases <- lease
			}
		}()
	}
	close(start)
	workers.Wait()
	close(leases)
	for lease := range leases {
		lease.release()
	}
	if got := admitted.Load(); got != limit {
		t.Fatalf("admitted %d unauthenticated connections, want %d", got, limit)
	}
}
