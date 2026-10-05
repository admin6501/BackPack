package manage

import (
	"github.com/backpack/backpack/config"
	"net"
	"testing"
)

func TestDatagramSetupLinksCarryPeerDirectionAndAddresses(t *testing.T) {
	for _, carrier := range []string{"xdi", "pck", "udp", "quic", "gre-fou"} {
		for _, side := range []string{"iran", "kharej"} {
			for _, mode := range []string{"dial", "listen"} {
				t.Run(carrier+"/"+side+"/"+mode, func(t *testing.T) {
					cfg := config.Config{L3: config.L3Config{Side: side, Mode: mode, Carrier: carrier, Addr: "0.0.0.0:8888", Token: "regression-token-for-paired-links", LocalIP: "10.243.81.1/30", PeerIP: "10.243.81.2", MTU: 1400, Ports: []string{"23298", "53835"}}}
					encoded, err := shareLinkOf("link-test", "203.0.113.7", cfg)
					if err != nil {
						t.Fatal(err)
					}
					link, err := DecodeShareLink(encoded)
					if err != nil {
						t.Fatal(err)
					}
					f := MirrorForPeer(link)
					if f.Side == "iran" {
						f.Ports = "23298,53835"
					}
					if needsPeerServerAddress(f) {
						t.Fatal("complete link requires another address")
					}
					s, err := f.ToNewDirectTunnel().spec()
					if err != nil {
						t.Fatal(err)
					}
					wantMode := "listen"
					if mode == "listen" {
						wantMode = "dial"
					}
					if s.Mode != wantMode || s.Carrier != carrier || s.Token != cfg.L3.Token || s.LocalIP != "10.243.81.2/30" || s.PeerIP != "10.243.81.1" {
						t.Fatalf("peer settings lost: carrier=%s mode=%s local=%s peer=%s", s.Carrier, s.Mode, s.LocalIP, s.PeerIP)
					}
					host, _, _ := net.SplitHostPort(s.Addr)
					if wantMode == "dial" && host != "203.0.113.7" {
						t.Fatal("dialer has no listener address")
					}
					if wantMode == "listen" && host != "0.0.0.0" {
						t.Fatal("listener incorrectly requires remote host")
					}
				})
			}
		}
	}
}

func TestLegacyXDILinkOnlyAsksTheDialerForAnAddress(t *testing.T) {
	for _, mode := range []string{"dial", "listen"} {
		cfg := config.Config{L3: config.L3Config{Mode: mode, Carrier: "xdi", Addr: "0.0.0.0:8888", Token: "legacy-link-token", LocalIP: "10.243.82.1/30", PeerIP: "10.243.82.2", Ports: []string{"2082"}}}
		encoded, err := shareLinkOf("legacy-xdi", "", cfg)
		if err != nil {
			t.Fatal(err)
		}
		link, err := DecodeShareLink(encoded)
		if err != nil {
			t.Fatal(err)
		}
		f := MirrorForPeer(link)
		if mode == "dial" {
			if needsPeerServerAddress(f) {
				t.Fatal("the ICMP listener must not require Iran's public IP")
			}
		} else {
			if !needsPeerServerAddress(f) {
				t.Fatal("the ICMP dialer must require the listening peer's IP")
			}
			f, err = withPeerServerAddress(f, "203.0.113.12")
			if err != nil {
				t.Fatal(err)
			}
			f.Ports = "2082"
		}
		if _, err = f.ToNewDirectTunnel().spec(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPanelPortEditsRenderNewMappingsWithoutChangingXDIDirection(t *testing.T) {
	for _, mode := range []string{"dial", "listen"} {
		before := config.L3Config{Mode: mode, Side: "iran", Carrier: "xdi", Addr: "203.0.113.9:8888", Token: "keep-token", Iface: "bp4", LocalIP: "10.243.83.1/30", PeerIP: "10.243.83.2", Ports: []string{"2082", "2095"}, MTU: 1400, FECData: 10, FECParity: 3}
		for _, raw := range []string{"2095,53835", "53835"} {
			after, err := applyDirectEdit(before, DirectEdit{Ports: &raw})
			if err != nil {
				t.Fatal(err)
			}
			got := decode(t, directSpecFrom("port-edit", after).render()).L3
			if len(got.Ports) != len(parsePorts(raw)) {
				t.Fatal("old port list survived render")
			}
			for i, p := range parsePorts(raw) {
				if got.Ports[i] != p {
					t.Fatal("new mapping missing")
				}
			}
			if got.Mode != before.Mode || got.Side != before.Side || got.Token != before.Token || got.Iface != before.Iface || got.FECData != 10 || got.FECParity != 3 {
				t.Fatal("port edit changed unrelated tunnel settings")
			}
		}
	}
}
