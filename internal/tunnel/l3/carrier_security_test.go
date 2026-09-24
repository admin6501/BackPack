package l3

import (
	"net"
	"testing"
	"time"
)

func TestUDPPathChangesOnlyAfterAuthentication(t *testing.T) {
	carrier, _, err := openUDPPaths(Config{Mode: ModeListen, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	defer carrier.Close()
	good, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer good.Close()
	stranger, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer stranger.Close()
	init, resp := handshakePair(t, "test-key")
	encap, err := NewEncap("ipip", 0)
	if err != nil {
		t.Fatal(err)
	}
	tun := &Tunnel{current: resp, carrier: carrier, encap: encap, log: quietLogger(), tun: newFakeDevice(1400)}
	receive := func(sender net.PacketConn, payload []byte) {
		t.Helper()
		if _, err := sender.WriteTo(payload, carrier.LocalAddr()); err != nil {
			t.Fatal(err)
		}
		carrier.SetReadDeadline(time.Now().Add(time.Second))
		buf := make([]byte, 2048)
		n, from, err := carrier.ReadFrom(buf)
		if err != nil {
			t.Fatal(err)
		}
		tun.route(nil, make([][]byte, 1), buf[:n], from)
	}
	// An authenticated packet pins the address, even if its inner IP is malformed.
	sealed, err := init.seal(nil, []byte("invalid inner IP"))
	if err != nil {
		t.Fatal(err)
	}
	receive(good, sealed)
	receive(stranger, []byte("untrusted garbage"))
	if _, err := carrier.WriteTo([]byte("reply"), tun.peerAddr()); err != nil {
		t.Fatal(err)
	}
	good.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 32)
	n, _, err := good.ReadFrom(buf)
	if err != nil || string(buf[:n]) != "reply" {
		t.Fatalf("genuine peer lost reply: %q %v", buf[:n], err)
	}
	// A new address with a fresh authenticated packet may still roam.
	sealed, err = init.seal(nil, []byte("another inner IP"))
	if err != nil {
		t.Fatal(err)
	}
	receive(stranger, sealed)
	if _, err := carrier.WriteTo([]byte("roamed"), tun.peerAddr()); err != nil {
		t.Fatal(err)
	}
	stranger.SetReadDeadline(time.Now().Add(time.Second))
	n, _, err = stranger.ReadFrom(buf)
	if err != nil || string(buf[:n]) != "roamed" {
		t.Fatalf("authenticated roaming failed: %q %v", buf[:n], err)
	}
}

func TestQUICCloseBeforeAnyPeer(t *testing.T) {
	c, _, err := listenQuic(Config{Mode: ModeListen, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	q := c.(*quicCarrier)
	defer q.Close()
	readDone := make(chan struct{})
	go func() { defer close(readDone); q.ReadFrom(make([]byte, 32)) }()
	// Wait until session has entered its accept section.
	deadline := time.Now().Add(time.Second)
	for q.acceptMu.TryLock() {
		q.acceptMu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("reader never started")
		}
		time.Sleep(time.Millisecond)
	}
	done := make(chan struct{})
	go func() { q.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		q.ln.Close()
		t.Fatal("Close deadlocked behind Accept")
	}
	select {
	case <-readDone:
	case <-time.After(time.Second):
		t.Fatal("ReadFrom survived Close")
	}
	if _, err := q.session(); err == nil {
		t.Fatal("closed carrier reopened")
	}
}

func TestReconstructedPacketCannotMoveUDPPath(t *testing.T) {
	good := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1000}
	forged := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 2000}
	path := &pinnedCarrier{peer: good}
	recovered := reconstructedAddress(&receivedPathAddr{Addr: forged, path: path, source: forged})
	tunnel := &Tunnel{peer: good, log: quietLogger()}
	tunnel.notePeer(recovered)
	if !sameAddr(path.peer, good) || !sameAddr(tunnel.peerAddr(), good) {
		t.Fatal("reconstructed packet moved the peer")
	}
}
