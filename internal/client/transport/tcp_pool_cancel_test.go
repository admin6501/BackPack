package transport

import (
	"context"
	"github.com/backpack/backpack/internal/utils/network"
	"github.com/sirupsen/logrus"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestIdleTCPPoolCancellation(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := NewTCPClient(ctx, &TcpConfig{RemoteAddr: ln.Addr().String(), Endpoints: network.NewEndpoints(ln.Addr().String()), DialTimeOut: time.Second}, logrus.New())
	done := make(chan struct{})
	go func() { defer close(done); c.tunnelDialer() }()
	ln.(*net.TCPListener).SetDeadline(time.Now().Add(3 * time.Second))
	peer, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	deadline := time.Now().Add(3 * time.Second)
	for atomic.LoadInt32(&c.poolConnections) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("connection never reached the idle pool")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("idle pool survived cancellation")
	}
}
