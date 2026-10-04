package manage

import (
	"github.com/BurntSushi/toml"
	"github.com/backpack/backpack/config"
	"testing"
)

func TestSpoofPeerSourceReachesRenderedPeerFromEitherEnd(t *testing.T) {
	for _, from := range []string{"kharej", "iran"} {
		link := ShareLink{Kind: "direct", From: from, Name: "paired", Tok: "a-token-0123456789abcdefghijklmno", Tr: "spoof", Encap: "gre", Port: "9000", Host: "203.0.113.11", LocalIP: "10.10.0.1/24", PeerIP: "10.10.0.2", SrcIPs: "81.28.60.1", PeerSrcIP: "8.8.4.4", Profile: "icmp", Ports: "443"}
		encoded, err := link.Encode()
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeShareLink(encoded)
		if err != nil {
			t.Fatal(err)
		}
		form := MirrorForPeer(decoded)
		spec, err := form.ToNewDirectTunnel().spec()
		if err != nil {
			t.Fatal(err)
		}
		var cfg config.Config
		if _, err := toml.Decode(spec.render(), &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.L3.SpoofSrcIP != "8.8.4.4" || cfg.L3.SpoofPeerSrcIP != "81.28.60.1" || cfg.L3.SpoofPeerIP != "203.0.113.11" {
			t.Fatalf("%s: incorrect real/forged peer fields: %+v", from, cfg.L3.SpoofConfig)
		}
	}
}

func TestSpoofPoolDoesNotPinOnePeerSource(t *testing.T) {
	f := MirrorForPeer(ShareLink{Kind: "direct", From: "kharej", Tr: "spoof", SrcIPs: "81.28.60.1, 81.28.60.2", PeerSrcIP: "8.8.4.4"})
	if f.Spoof.PeerSrcIP != "" || f.Spoof.SrcIPs != "8.8.4.4" {
		t.Fatal("rotating source incorrectly pinned", f.Spoof)
	}
}

func TestPeerTunnelAddressKeepsProducersSubnetPrefix(t *testing.T) {
	for _, prefix := range []string{"24", "30"} {
		f := MirrorForPeer(ShareLink{Kind: "direct", From: "kharej", LocalIP: "10.10.0.2/" + prefix, PeerIP: "10.10.0.1"})
		if f.LocalIP != "10.10.0.1/"+prefix || f.PeerIP != "10.10.0.2" {
			t.Fatal("local prefix lost", f.LocalIP, f.PeerIP)
		}
	}
}

// A normal wizard sets only this machine's source, not an expected peer source.
// Exercise that real producer config, rather than a hand-built link with ps set.
func TestWizardSpoofSourceIsInheritedByRenderedPeer(t *testing.T) {
	for _, from := range []string{"iran", "kharej"} {
		for _, tc := range []struct {
			name, source, expected string
			pool                   []string
		}{
			{name: "single", source: "81.28.60.1"},
			{name: "pool", source: "81.28.60.1", pool: []string{"81.28.60.1", "81.28.60.2"}},
			{name: "explicit-peer", source: "81.28.60.1", expected: "81.28.60.3"},
			{name: "unforged"},
		} {
			t.Run(from+"/"+tc.name, func(t *testing.T) {
				mode, local, peer := "dial", "10.10.0.1/30", "10.10.0.2"
				if from == "kharej" {
					mode, local, peer = "listen", "10.10.0.2/30", "10.10.0.1"
				}
				producer := config.Config{L3: config.L3Config{Mode: mode, Carrier: "spoof", Encap: "gre", Addr: "203.0.113.10:2547", Token: "a-token-0123456789abcdefghijklmno", Iface: "bp0", LocalIP: local, PeerIP: peer, MTU: 1400, SpoofConfig: config.SpoofConfig{SpoofProfile: "icmp", SpoofPeerIP: "203.0.113.11", SpoofSrcIP: tc.source, SpoofSrcPool: tc.pool, SpoofPeerSrcIP: tc.expected}}}
				encoded, err := shareLinkOf("source", "203.0.113.10", producer)
				if err != nil {
					t.Fatal(err)
				}
				link, err := DecodeShareLink(encoded)
				if err != nil {
					t.Fatal(err)
				}
				form := MirrorForPeer(link)
				if form.Side == "iran" {
					form.Ports = "2082, 2095"
				}
				spec, err := form.ToNewDirectTunnel().spec()
				if err != nil {
					t.Fatal(err)
				}
				var got config.Config
				if _, err := toml.Decode(spec.render(), &got); err != nil {
					t.Fatal(err)
				}
				want := tc.source
				if tc.expected != "" {
					want = tc.expected
				}
				if got.L3.SpoofSrcIP != want {
					t.Fatalf("peer own source=%q, want %q", got.L3.SpoofSrcIP, want)
				}
				if len(tc.pool) > 0 && tc.expected == "" {
					if len(got.L3.SpoofSrcPool) != 2 {
						t.Fatalf("source pool lost: %v", got.L3.SpoofSrcPool)
					}
					if got.L3.SpoofPeerSrcIP != "" {
						t.Fatal("pool pinned to one source")
					}
				}
				if got.L3.SpoofPeerIP != "203.0.113.10" {
					t.Fatal("real IP confused with forged source")
				}
			})
		}
	}
}
