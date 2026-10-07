package l3

import (
	"net"
	"testing"
	"time"
)

func TestIPv6DatagramCarriers(t *testing.T) {
	for _, carrier := range []string{CarrierUDP, CarrierGREFOU, CarrierQuic} {
		t.Run(carrier, func(t *testing.T) {
			listener, _, err := openBareCarrier(Config{Mode: ModeListen, Addr: "[::1]:0", Carrier: carrier})
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			dialer, peer, err := openBareCarrier(Config{Mode: ModeDial, Addr: listener.LocalAddr().String(), Carrier: carrier})
			if err != nil {
				t.Fatal(err)
			}
			defer dialer.Close()
			listener.SetDeadline(time.Now().Add(3 * time.Second))
			dialer.SetDeadline(time.Now().Add(3 * time.Second))
			for _, c := range []DatagramCarrier{listener, dialer} {
				if c.Overhead() < 48 {
					t.Fatalf("IPv6 overhead undercounted: %d", c.Overhead())
				}
			}
			if _, err = dialer.WriteTo([]byte("up"), peer); err != nil {
				t.Fatal(err)
			}
			buf := make([]byte, 2048)
			n, from, err := listener.ReadFrom(buf)
			if err != nil || string(buf[:n]) != "up" {
				t.Fatalf("upload %q %v", buf[:n], err)
			}
			if _, err = listener.WriteTo([]byte("down"), from); err != nil {
				t.Fatal(err)
			}
			n, _, err = dialer.ReadFrom(buf)
			if err != nil || string(buf[:n]) != "down" {
				t.Fatalf("download %q %v", buf[:n], err)
			}
		})
	}
}

func TestIPv6CarrierValidation(t *testing.T) {
	for _, carrier := range []string{CarrierXdi, CarrierUDP, CarrierQuic, CarrierGREFOU, CarrierPck, CarrierSNI, CarrierSpoof} {
		err := CheckCarrierAddress(carrier, net.JoinHostPort("2001:db8::1", "9000"))
		if (err == nil) != SupportsIPv6(carrier) {
			t.Fatalf("%s: %v", carrier, err)
		}
		if err = CheckCarrierAddress(carrier, "192.0.2.1:9000"); err != nil {
			t.Fatal(err)
		}
	}
}
