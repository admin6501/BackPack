package network

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestResolveIPv6Backends(t *testing.T) {
	for _, addr := range []string{"[::1]:443", "[2001:db8::1]:8443", "[fe80::1%eth0]:80"} {
		_, got, err := ResolveRemoteAddr(addr)
		if err != nil || got != addr {
			t.Fatalf("%s: got %s err %v", addr, got, err)
		}
	}
	for _, addr := range []string{"[::1]", "[::1]:65536", "::1:443", "-1", "0"} {
		if _, _, err := ResolveRemoteAddr(addr); err == nil {
			t.Fatalf("accepted %q", addr)
		}
	}
}
func TestQUICCloseInterruptsReader(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ln, err := QUICListen("127.0.0.1:0", QUICSettings{})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	client, err := QUICDial(ctx, ln.Addr().String(), QUICSettings{})
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseWithError(0, "")
	server, err := ln.Accept(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer server.CloseWithError(0, "")
	cs, err := client.OpenStreamSync(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = cs.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	ss, err := server.AcceptStream(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.ReadFull(ss, make([]byte, 1)); err != nil {
		t.Fatal(err)
	}
	wrapped := NewQUICStreamConn(ss, server)
	done := make(chan error, 1)
	go func() { _, err := wrapped.Read(make([]byte, 1)); done <- err }()
	if err = wrapped.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("read succeeded after close")
		}
	case <-ctx.Done():
		t.Fatal("Close did not interrupt Read")
	}
}

func TestQUICCloseAfterFINPreservesBufferedResponse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ln, err := QUICListen("127.0.0.1:0", QUICSettings{})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	client, err := QUICDial(ctx, ln.Addr().String(), QUICSettings{})
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseWithError(0, "")
	server, err := ln.Accept(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer server.CloseWithError(0, "")
	cs, err := client.OpenStreamSync(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = cs.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	ss, err := server.AcceptStream(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.ReadFull(ss, make([]byte, 1)); err != nil {
		t.Fatal(err)
	}
	wrapped := NewQUICStreamConn(ss, server).(*QUICStreamConn)
	payload := bytes.Repeat([]byte("response"), 32768)
	done := make(chan error, 1)
	go func() {
		_, err := wrapped.Write(payload)
		if err == nil {
			err = wrapped.CloseWrite()
		}
		wrapped.Close()
		done <- err
	}()
	cs.SetReadDeadline(time.Now().Add(3 * time.Second))
	got, err := io.ReadAll(cs)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("got %d of %d bytes, err=%v", len(got), len(payload), err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
