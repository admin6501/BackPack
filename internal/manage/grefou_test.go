package manage

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/backpack/backpack/config"
)

func TestGREFOUSetupLinkAndEditBothDirections(t *testing.T) {
	for _, side := range []directSide{sideIran, sideKharej} {
		for _, mode := range []string{"dial", "listen"} {
			t.Run(side.String()+"-"+mode, func(t *testing.T) {
				s := l3Spec{Side: side, Mode: mode, Carrier: "gre-fou", Token: strings.Repeat("k", 64), Addr: "203.0.113.7:9000", LocalIP: "10.230.17.1/30", PeerIP: "10.230.17.2", MTU: 1400, GREKey: 17}
				if side == sideIran {
					s.Ports = []string{"2082", "2095"}
				}
				var cfg config.Config
				if _, err := toml.Decode(s.render(), &cfg); err != nil {
					t.Fatal(err)
				}
				if cfg.L3.SideName() != side.String() || cfg.L3.Mode != mode {
					t.Fatal("direction/geography lost")
				}
				var edited config.Config
				if _, err := toml.Decode(l3SpecOf(Tunnel{Name: "fou-edit"}, cfg.L3).render(), &edited); err != nil {
					t.Fatal(err)
				}
				if edited.L3.Mode != mode || edited.L3.SideName() != side.String() {
					t.Fatal("edit changed direction")
				}
				link, err := shareLinkOf("fou-test", "203.0.113.8", cfg)
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := DecodeShareLink(link)
				if err != nil {
					t.Fatal(err)
				}
				form := MirrorForPeer(decoded)
				wantMode := "listen"
				if mode == "listen" {
					wantMode = "dial"
				}
				if form.Mode != wantMode || form.Side == side.String() {
					t.Fatal("peer direction/geography wrong")
				}
				// The Iran receiver supplies its own forwarded ports when the link
				// comes from kharej, just like the existing setup flow.
				if form.Side == "iran" {
					form.Ports = "2082,2095"
				}
				peer, err := form.ToNewDirectTunnel().spec()
				if err != nil {
					t.Fatal(err)
				}
				var peerCfg config.Config
				if _, err := toml.Decode(peer.render(), &peerCfg); err != nil {
					t.Fatal(err)
				}
				if peerCfg.L3.Mode != wantMode || peerCfg.L3.Carrier != "gre-fou" || peerCfg.L3.LocalIP != "10.230.17.2/30" || peerCfg.L3.GREKey != 17 {
					t.Fatal("paired settings lost")
				}
				if wantMode == "dial" && peerCfg.L3.Addr != "203.0.113.8:9000" {
					t.Fatal("dialler has no real peer address")
				}
				if wantMode == "listen" && peerCfg.L3.Addr != "0.0.0.0:9000" {
					t.Fatal("listener not bound locally")
				}
				settings := directSettingsFrom("fou-test", cfg.L3)
				if settings.HoldsPorts != (side == sideIran) || settings.Side != side.String() {
					t.Fatal("forwarded ports moved to wrong side")
				}
			})
		}
	}
}

func TestGREFOUPortClashesWithUDP(t *testing.T) {
	existing := []l3Tunnel{{T: Tunnel{Name: "existing"}, L: config.L3Config{Mode: "listen", Carrier: "udp", Addr: "0.0.0.0:9000"}}}
	if l3ListenClash("new", "gre-fou", 9000, 1, existing) == "" {
		t.Fatal("FOU listener can collide with UDP")
	}
}

func TestGREFOUReviewRegressions(t *testing.T) {
	for _, side := range []directSide{sideIran, sideKharej} {
		mode := "listen"
		if side == sideKharej {
			mode = "dial"
		}
		c := config.L3Config{Side: side.String(), Mode: " " + strings.ToUpper(mode) + " ", Carrier: "udp", Addr: "203.0.113.7:9000", Token: "shared", LocalIP: "10.231.0.1/30", PeerIP: "10.231.0.2", MTU: 1400}
		if c.DirectionName() != "reverse" {
			t.Fatal("whitespace changed the initiation direction")
		}
		tun := Tunnel{Role: c.SideName(), Transport: "l3/udp", Direction: c.DirectionName()}
		if DialsOut(tun) != (side == sideKharej) || HoldsPorts(tun) != (side == sideIran) {
			t.Fatal("management confused connection direction and geography")
		}
		out := capture(t, func() { summariseL3Classic(l3Spec{Side: side, Mode: mode, Carrier: "gre-fou", Addr: c.Addr}) })
		wrong := "Dials       :"
		if side == sideKharej {
			wrong = "Listens on  :"
		}
		if strings.Contains(out, wrong) {
			t.Fatal("summary describes the wrong initiation mode")
		}
		link, err := shareLinkOf("explicit-udp", "203.0.113.8", config.Config{L3: c})
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeShareLink(link)
		if err != nil {
			t.Fatal(err)
		}
		f := MirrorForPeer(decoded)
		if f.Side == "iran" {
			f.Ports = "2082"
		}
		if _, err := f.ToNewDirectTunnel().spec(); err != nil {
			t.Fatalf("valid explicit UDP mode rejected: %v", err)
		}
	}
	for _, carrier := range []string{"gre-fou", " GRE-FOU "} {
		existing := []l3Tunnel{{T: Tunnel{Name: "upper"}, L: config.L3Config{Mode: "listen", Carrier: " GRE-FOU ", Addr: "0.0.0.0:9000"}}}
		if l3ListenClash("new", carrier, 9000, 1, existing) == "" {
			t.Fatal("capitalized carrier bypassed the port collision check")
		}
	}
}
