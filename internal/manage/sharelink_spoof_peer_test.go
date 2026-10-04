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
