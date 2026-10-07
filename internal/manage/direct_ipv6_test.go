package manage

import (
	"github.com/backpack/backpack/config"
	"net"
	"testing"
)

func TestIPv6SetupLinkAndSSHForms(t *testing.T) {
	for _, carrier := range []string{"xdi", "udp", "quic", "gre-fou"} {
		for _, side := range []string{"iran", "kharej"} {
			for _, mode := range []string{"dial", "listen"} {
				t.Run(carrier+"/"+side+"/"+mode, func(t *testing.T) {
					addr := "[2001:db8::2]:9000"
					if mode == "listen" {
						addr = "[::]:9000"
					}
					cfg := config.Config{L3: config.L3Config{Side: side, Mode: mode, Carrier: carrier, Addr: addr, Token: "ipv6-test", LocalIP: "10.243.81.1/30", PeerIP: "10.243.81.2", Ports: []string{"2082"}}}
					encoded, err := shareLinkOf("ipv6-test", "2001:db8::1", cfg)
					if err != nil {
						t.Fatal(err)
					}
					link, err := DecodeShareLink(encoded)
					if err != nil {
						t.Fatal(err)
					}
					form := MirrorForPeer(link)
					form.Ports = "2082"
					spec, err := form.ToNewDirectTunnel().spec()
					if err != nil {
						t.Fatal(err)
					}
					host, _, _ := net.SplitHostPort(spec.Addr)
					want := "::"
					if mode == "listen" {
						want = "2001:db8::1"
					}
					if host != want || spec.LocalIP != "10.243.81.2/30" || spec.PeerIP != "10.243.81.1" {
						t.Fatalf("wrong peer: %+v", spec)
					}
				})
			}
		}
	}
}

func TestIPv6DirectFormValidation(t *testing.T) {
	for _, carrier := range []string{"xdi", "udp", "quic", "gre-fou", "pck", "sni", "spoof"} {
		for _, mode := range []string{"dial", "listen"} {
			f := NewDirectTunnel{Side: "kharej", Mode: mode, Carrier: carrier, Name: "ipv6-test", Token: "test", TunnelPort: "9000", PeerAddr: "[2001:db8::1]", ListenHost: "::"}
			s, err := f.spec()
			supported := carrier != "pck" && carrier != "sni" && carrier != "spoof"
			if (err == nil) != supported {
				t.Fatalf("%s/%s: %v", carrier, mode, err)
			}
			if err == nil {
				if _, _, err = net.SplitHostPort(s.Addr); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
