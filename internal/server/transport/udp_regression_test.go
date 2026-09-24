package transport

import (
	"context"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"testing"
	"time"
)

func TestLimitedUDPFlowStillSkipsProxyProtocol(t *testing.T) {
	limits := newLimiter(Limits{BandwidthMbps: 10})
	conn := limits.wrap(context.Background(), &udpFlow{})
	if !isUDPFlow(conn) {
		t.Fatal("bandwidth wrapping hides UDP flow")
	}
}
func TestUDPPairingTimeoutDropsFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &UdpTransport{logger: logrus.New(), limits: newLimiter(Limits{MaxConnections: 1})}
	if !s.limits.acquire() {
		t.Fatal("slot unavailable")
	}
	flow := &LocalUDPConn{timeCreated: time.Now().Add(-pairingTimeout + 100*time.Millisecond).UnixMilli(), payload: make(chan []byte), addr: &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234}}
	active := map[string]*LocalUDPConn{flow.addr.String(): flow}
	var mu sync.Mutex
	queue := make(chan *LocalUDPConn, 1)
	queue <- flow
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.handleLoop(&udpGen{ctx: ctx, tunnelChannel: make(chan *TunnelUDPConn)}, queue, &active, &mu)
	}()
	select {
	case _, ok := <-flow.payload:
		if ok {
			t.Fatal("flow was not dropped")
		}
	case <-time.After(time.Second):
		t.Fatal("pairing deadline did not release flow")
	}
	cancel()
	<-done
	if s.limits.active.Load() != 0 {
		t.Fatal("slot leaked")
	}
	mu.Lock()
	remaining := len(active)
	mu.Unlock()
	if remaining != 0 {
		t.Fatal("stale flow remains")
	}
}
