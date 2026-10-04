package l3

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestGREFOUCarriesEncryptedIPBothWays(t *testing.T) {
	p := newPair(t, "gre", 17, "fou-shared-token", "fou-shared-token", CarrierGREFOU)
	awaitSession(t, p.dialer, 5*time.Second)
	across(t, p.dialDev, p.listenDev, ipv4Packet(1, 2, 3, 4))
	across(t, p.listenDev, p.dialDev, ipv4Packet(5, 6, 7, 8))
}

func TestGREFOURejectsWrongToken(t *testing.T) {
	p := newPair(t, "gre", 0, "wrong-token", "right-token", CarrierGREFOU)
	p.dialDev.inject <- ipv4Packet(1)
	select {
	case <-p.listenDev.emitted:
		t.Fatal("unauthenticated packet reached the interface")
	case <-time.After(300 * time.Millisecond):
	}
	if p.listener.sendSession() != nil {
		t.Fatal("wrong token established a session")
	}
}

func TestGREFOUWireAndMalformedPackets(t *testing.T) {
	c, _, err := openGREFOU(Config{Mode: ModeListen, Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	u, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer u.Close()
	c.SetDeadline(time.Now().Add(time.Second))
	u.SetDeadline(time.Now().Add(time.Second))
	if c.Overhead() != 32 {
		t.Fatalf("overhead: %d", c.Overhead())
	}
	if n, err := c.WriteTo([]byte("sealed"), u.LocalAddr()); err != nil || n != 6 {
		t.Fatalf("write: %d %v", n, err)
	}
	b := make([]byte, 100)
	n, _, err := u.ReadFrom(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b[:n], []byte{0, 0, 0x88, 0xb5, 's', 'e', 'a', 'l', 'e', 'd'}) {
		t.Fatalf("not GRE-in-UDP: %x", b[:n])
	}
	for _, bad := range [][]byte{{0}, {0, 1, 0x88, 0xb5, 1}, {0, 0, 8, 0, 1}, {0x20, 0, 0x88, 0xb5, 1}} {
		if _, err := u.WriteTo(bad, c.LocalAddr()); err != nil {
			t.Fatal(err)
		}
	}
	_, err = u.WriteTo([]byte{0, 0, 0x88, 0xb5, 9, 8}, c.LocalAddr())
	if err != nil {
		t.Fatal(err)
	}
	n, _, err = c.ReadFrom(b)
	if err != nil || !bytes.Equal(b[:n], []byte{9, 8}) {
		t.Fatalf("read: %x %v", b[:n], err)
	}
}
