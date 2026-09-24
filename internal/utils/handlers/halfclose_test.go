package handlers

import (
	"context"
	"github.com/backpack/backpack/internal/metrics"
	"github.com/backpack/backpack/internal/web"
	"io"
	"net"
	"testing"
	"time"
)

func TestHalfClosePreservesResponse(t *testing.T) {
	old := ZeroCopy()
	defer SetZeroCopy(old)
	for _, splice := range []bool{false, true} {
		SetZeroCopy(splice)
		client, from := tcpPair(t)
		to, backend := tcpPair(t)
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			TCPConnectionHandler(ctx, false, metrics.CountedConn(from), to, quietLogger(), &web.Usage{}, 80, false)
		}()
		backendDone := make(chan error, 1)
		go func() {
			_, err := io.ReadAll(backend)
			if err == nil {
				_, err = backend.Write([]byte("response after EOF"))
			}
			backend.Close()
			backendDone <- err
		}()
		client.SetDeadline(time.Now().Add(2 * time.Second))
		if _, err := client.Write([]byte("request")); err != nil {
			cancel()
			t.Fatal(err)
		}
		if err := client.(*net.TCPConn).CloseWrite(); err != nil {
			cancel()
			t.Fatal(err)
		}
		got, err := io.ReadAll(client)
		cancel()
		<-done
		if err != nil || string(got) != "response after EOF" {
			t.Fatalf("splice=%v response=%q err=%v", splice, got, err)
		}
		if err := <-backendDone; err != nil {
			t.Fatal(err)
		}
	}
}
