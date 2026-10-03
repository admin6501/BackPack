package manage

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/backpack/backpack/config"
)

func TestSpoofSourcesReachCorrectFieldsWhenEitherSideStarts(t *testing.T) {
	for _, mode := range []string{"dial", "listen"} {
		t.Run(mode, func(t *testing.T) {
			cfg := config.Config{L3: config.L3Config{Mode: mode, Addr: "203.0.113.9:9000", Carrier: "spoof", Token: strings.Repeat("k", 64), LocalIP: "10.20.0.1/30", PeerIP: "10.20.0.2",
				SpoofConfig: config.SpoofConfig{SpoofProfile: "udp", SpoofSrcIP: "81.28.60.1", SpoofPeerSrcIP: "81.28.60.2", SpoofPeerIP: "203.0.113.10"}}}
			encoded, err := shareLinkOf("paired", "198.51.100.8", cfg)
			if err != nil {
				t.Fatal(err)
			}
			link, err := DecodeShareLink(encoded)
			if err != nil {
				t.Fatal(err)
			}
			form := MirrorForPeer(link)
			if form.Side == "iran" {
				form.Ports = "23298, 53835"
			}
			if form.Spoof == nil || form.Spoof.SrcIPs != "81.28.60.2" || form.Spoof.PeerSrcIP != "81.28.60.1" {
				t.Fatalf("forged addresses in wrong fields: %+v", form.Spoof)
			}
			if form.SpoofPeerIP != "198.51.100.8" {
				t.Fatal("real IP confused with forged source")
			}
			spec, err := form.ToNewDirectTunnel().spec()
			if err != nil {
				t.Fatal(err)
			}
			var peer config.Config
			if _, err := toml.Decode(spec.render(), &peer); err != nil {
				t.Fatal(err)
			}
			if peer.L3.SpoofSrcIP != "81.28.60.2" || peer.L3.SpoofPeerSrcIP != "81.28.60.1" || peer.L3.SpoofPeerIP != "198.51.100.8" {
				t.Fatalf("rendered peer IPs: %+v", peer.L3.SpoofConfig)
			}
			if mode == "listen" && (peer.L3.Mode != "dial" || len(peer.L3.Ports) != 2) {
				t.Fatal("Iran peer missing supplied ports")
			}
			if mode == "dial" && peer.L3.Mode != "listen" {
				t.Fatal("kharej peer does not listen")
			}
		})
	}
}
