package node

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestSSHHandshakeCancellation(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		c, _, err := dialSSH(ctx, "silent", SSHTarget{Host: "127.0.0.1", Port: ln.Addr().(*net.TCPAddr).Port, User: "test"})
		if c != nil {
			c.Close()
		}
		done <- err
	}()
	ln.(*net.TCPListener).SetDeadline(time.Now().Add(3 * time.Second))
	peer, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled handshake succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("handshake ignored cancellation")
	}
}
