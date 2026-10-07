package network

import (
	"bytes"
	"golang.org/x/net/bpf"
	"golang.org/x/net/icmp"
	"net"
	"os"
	"testing"
	"time"
)

func TestICMPv6RawRoundTrip(t *testing.T) {
	// Real sockets verify Linux checksums, BPF offsets, and both batching paths.
	// No firewall commands: all traffic stays on loopback.
	for _, batch := range []bool{false, true} {
		t.Run(map[bool]string{false: "single", true: "batch"}[batch], func(t *testing.T) {
			sp, err := icmp.ListenPacket("ip6:ipv6-icmp", "::1")
			if err != nil {
				if os.Getenv("BACKPACK_REQUIRE_RAW_TEST") == "1" {
					t.Fatal(err)
				}
				t.Skipf("raw IPv6 socket unavailable: %v", err)
			}
			s := newICMPServerConnWith(sp, "ipv6-regression").(*icmpConn)
			s.proto = 58
			defer s.Close()
			cp, err := icmp.ListenPacket("ip6:ipv6-icmp", "::1")
			if err != nil {
				t.Fatal(err)
			}
			c := newICMPClientConnWith(cp, "ipv6-regression").(*icmpConn)
			c.proto = 58
			defer c.Close()
			attachICMPFilter(sp, 128, -1)
			attachICMPFilter(cp, 129, int(c.id))
			s.SetDeadline(time.Now().Add(2 * time.Second))
			c.SetDeadline(time.Now().Add(2 * time.Second))
			write := func(x *icmpConn, p []byte, dst net.Addr) {
				t.Helper()
				var e error
				if batch {
					_, e = x.WriteBatch([][]byte{p}, dst)
				} else {
					_, e = x.WriteTo(p, dst)
				}
				if e != nil {
					t.Fatal(e)
				}
			}
			read := func(x *icmpConn, want []byte) net.Addr {
				t.Helper()
				buf := make([]byte, 2048)
				var n int
				var a net.Addr
				var e error
				if batch {
					sizes := make([]int, 1)
					from := make([]net.Addr, 1)
					_, e = x.ReadBatch([][]byte{buf}, sizes, from)
					n, a = sizes[0], from[0]
				} else {
					n, a, e = x.ReadFrom(buf)
				}
				if e != nil {
					t.Fatal(e)
				}
				if !bytes.Equal(buf[:n], want) {
					t.Fatalf("got %q want %q", buf[:n], want)
				}
				return a
			}
			up, down := []byte("IPv6 upload"), []byte("IPv6 download")
			write(c, up, &net.IPAddr{IP: net.IPv6loopback})
			from := read(s, up)
			write(s, down, from)
			read(c, down)
		})
	}
}

func TestICMPv6FilterAndZone(t *testing.T) {
	vm, err := bpf.NewVM(icmp6Filter(129, 42))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		typ  byte
		id   uint16
		want bool
	}{{129, 42, true}, {129, 43, false}, {128, 42, false}, {135, 42, false}} {
		packet := appendEcho(nil, tc.typ, tc.id, 1)
		n, e := vm.Run(packet)
		if e != nil || (n > 0) != tc.want {
			t.Fatalf("filter %+v: %d %v", tc, n, e)
		}
	}
	c := &icmpConn{server: true, proto: 58}
	a := c.peerAddr(&net.IPAddr{IP: net.ParseIP("fe80::1"), Zone: "eth0"}, 42).(*net.UDPAddr)
	if a.Zone != "eth0" || a.Port != 42 {
		t.Fatalf("lost zone/session: %v", a)
	}
}

func TestICMPv6Factory(t *testing.T) {
	s, _, err := NewXdiPacketConn(true, "ipv6-factory-regression", "[::1]:9000")
	if err != nil {
		if os.Getenv("BACKPACK_REQUIRE_RAW_TEST") == "1" {
			t.Fatal(err)
		}
		t.Skipf("raw socket unavailable: %v", err)
	}
	defer s.Close()
	c, peer, err := NewXdiPacketConn(false, "ipv6-factory-regression", "[::1]:9000")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if s.(*icmpConn).proto != 58 || c.(*icmpConn).proto != 58 {
		t.Fatal("IPv6 opened an IPv4 socket")
	}
	s.SetDeadline(time.Now().Add(2 * time.Second))
	c.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err = c.WriteTo([]byte("factory"), peer); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 128)
	n, from, err := s.ReadFrom(buf)
	if err != nil || string(buf[:n]) != "factory" {
		t.Fatalf("request %q: %v", buf[:n], err)
	}
	if _, err = s.WriteTo([]byte("reply"), from); err != nil {
		t.Fatal(err)
	}
	n, _, err = c.ReadFrom(buf)
	if err != nil || string(buf[:n]) != "reply" {
		t.Fatalf("response %q: %v", buf[:n], err)
	}
}
