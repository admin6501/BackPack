package l3

import (
	"net"
	"testing"
	"time"
)

func TestAuthenticatedMultipathWithAndWithoutFEC(t *testing.T) {
	for _, fec := range []bool{false, true} {
		name := "udp"
		if fec {
			name = "fec"
		}
		t.Run(name, func(t *testing.T) {
			var listeners, dialers []DatagramCarrier
			for i := 0; i < 2; i++ {
				l, err := listenUDP("127.0.0.1:0", 0)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { l.Close() })
				d, peer, err := dialUDP(l.LocalAddr().String(), 0)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { d.Close() })
				listeners = append(listeners, &pinnedCarrier{DatagramCarrier: l})
				dialers = append(dialers, &pinnedCarrier{DatagramCarrier: d, peer: peer})
			}
			listen := newMultipathCarrier(listeners, nil)
			dial := newMultipathCarrier(dialers, listeners[0].LocalAddr())
			if fec {
				var err error
				listen, err = newFECCarrier(listen, FECConfig{Data: 4, Parity: 1})
				if err != nil {
					t.Fatal(err)
				}
				dial, err = newFECCarrier(dial, FECConfig{Data: 4, Parity: 1})
				if err != nil {
					t.Fatal(err)
				}
			}
			defer listen.Close()
			defer dial.Close()
			read := func(c DatagramCarrier) ([]byte, net.Addr) {
				t.Helper()
				c.SetReadDeadline(time.Now().Add(2 * time.Second))
				buf := make([]byte, 2048)
				n, a, err := c.ReadFrom(buf)
				if err != nil {
					t.Fatal(err)
				}
				return buf[:n], a
			}
			attempt, err := beginHandshake("multipath-token", 0, "gre")
			if err != nil {
				t.Fatal(err)
			}
			encap, err := NewEncap("ipip", 0)
			if err != nil {
				t.Fatal(err)
			}
			tunnel := &Tunnel{cfg: Config{Mode: ModeListen, Token: "multipath-token"}, carrier: listen, encap: encap, log: quietLogger(), tun: newFakeDevice(1400)}
			if _, err = dial.WriteTo(attempt.datagram(), nil); err != nil {
				t.Fatal(err)
			}
			packet, from := read(listen)
			tunnel.route(nil, make([][]byte, 1), packet, from)
			reply, _ := read(dial)
			_, body, err := parseHeader(reply)
			if err != nil {
				t.Fatal(err)
			}
			session, err := attempt.complete(body)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 4; i++ {
				sealed, err := session.seal(nil, []byte("inner"))
				if err != nil {
					t.Fatal(err)
				}
				if _, err = dial.WriteTo(sealed, nil); err != nil {
					t.Fatal(err)
				}
				packet, from = read(listen)
				tunnel.route(nil, make([][]byte, 1), packet, from)
			}
			for i, p := range listeners {
				pinned := p.(*pinnedCarrier)
				pinned.mu.Lock()
				peer := pinned.peer
				pinned.mu.Unlock()
				if peer == nil {
					t.Fatalf("path %d was never authenticated", i)
				}
			}
			for i := 0; i < 4; i++ {
				if _, err = listen.WriteTo([]byte("reply"), tunnel.peerAddr()); err != nil {
					t.Fatal(err)
				}
				got, _ := read(dial)
				if string(got) != "reply" {
					t.Fatalf("multipath response=%q", got)
				}
			}
		})
	}
}
