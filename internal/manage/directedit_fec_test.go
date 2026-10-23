package manage

import (
	"reflect"
	"strings"
	"testing"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tui"
)

func TestDirectEditorOffersFECAtTheSameNumberAcrossCarriers(t *testing.T) {
	for _, carrier := range []string{"udp", "quic", "spoof", "pck", "sni", "xdi", "gre-fou"} {
		for _, side := range []string{"iran", "kharej"} {
			for _, mode := range []string{"dial", "listen"} {
				l := config.L3Config{Side: side, Mode: mode, Carrier: carrier, FECData: 20, FECParity: 4}
				options := l3EditOptions(l)
				index := 5 // displayed choice 6 on Iran
				if side == "kharej" {
					index = 3 // displayed choice 4 on kharej
				}
				if options[index].Title != "Error correction (FEC)" {
					t.Fatalf("%s/%s/%s: choice %d is %q", carrier, side, mode, index+1, options[index].Title)
				}
				if !strings.Contains(options[index].Desc, "4 spare packets per 20 data") {
					t.Fatalf("the menu hid the current scheme: %q", options[index].Desc)
				}
				if action, ok := l3EditAction(index, side == "iran"); !ok || action != 5 {
					t.Fatalf("%s: FEC choice maps to action %d, ok=%v", side, action, ok)
				}
				if carrier == "spoof" && options[index+1].Title != "IP Spoofing" {
					t.Fatal("adding FEC displaced the Spoof editor")
				}
			}
		}
	}
}

func TestDirectFECEditCanEnableDisableAndKeepACustomScheme(t *testing.T) {
	plan := defaultL3FEC()
	for _, side := range []string{"iran", "kharej"} {
		for _, tc := range []struct {
			name              string
			data, parity      int
			input             string
			wantData, wantPar int
		}{
			{"enable", 0, 0, "y\n", plan.Data, plan.Parity},
			{"disable", 10, 3, "n\n", 0, 0},
			{"leave off", 0, 0, "\n", 0, 0},
			{"keep custom default", 20, 4, "\n", 20, 4},
			{"keep custom yes", 20, 4, "y\n", 20, 4},
			{"disable custom", 20, 4, "n\n", 0, 0},
		} {
			t.Run(side+"/"+tc.name, func(t *testing.T) {
				before := config.L3Config{
					Side: side, Mode: "dial", Carrier: "spoof", Encap: "gre", GREKey: 7,
					Addr: "203.0.113.9:9000", Token: "keep-this-token", Iface: "bp0",
					LocalIP: "10.20.0.1/30", PeerIP: "10.20.0.2", MTU: 1300,
					Ports: []string{"443"}, AcceptUDP: true, SockBuf: 8 << 20,
					FECData: tc.data, FECParity: tc.parity,
					SpoofConfig: config.SpoofConfig{SpoofProfile: "tcp", SpoofPeerIP: "198.51.100.4", SpoofSrcIP: "192.0.2.1"},
				}
				applySpoofStealth(&before.SpoofConfig)
				got := before
				restore := tui.SetInput(strings.NewReader(tc.input))
				defer restore()
				out := capture(t, func() { askL3FECEdit(&got) })
				want := before
				want.FECData, want.FECParity = tc.wantData, tc.wantPar
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("FEC edit changed the wrong settings: got %+v, want %+v", got, want)
				}
				there := "kharej"
				if side == "kharej" {
					there = "Iran"
				}
				if !strings.Contains(out, "Set the "+there+" end to the SAME fec_data/fec_parity pair") {
					t.Fatal("the editor did not explain how to match the other end")
				}
				body := l3SpecOf(Tunnel{Name: "fec-edit"}, got).render()
				round := decode(t, body).L3
				if round.FECData != tc.wantData || round.FECParity != tc.wantPar {
					t.Fatalf("saved scheme = %d/%d, want %d/%d", round.FECData, round.FECParity, tc.wantData, tc.wantPar)
				}
				if tc.wantData == 0 && strings.Contains(body, "fec_") {
					t.Fatal("disabling FEC left scheme keys in the saved config")
				}
				if round.Token != before.Token || round.MTU != before.MTU || !reflect.DeepEqual(round.SpoofConfig, before.SpoofConfig) {
					t.Fatal("saving the FEC edit changed the token, MTU or Spoof settings")
				}
			})
		}
	}
}
