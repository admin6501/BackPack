package manage

import (
	"fmt"
	"strings"
	"testing"

	"github.com/backpack/backpack/internal/tui"
)

func TestManualDirectAddressPromptsBothSides(t *testing.T) {
	for _, side := range []directSide{sideIran, sideKharej} {
		for _, carrier := range []string{"xdi", "pck", "udp", "quic"} {
			t.Run(fmt.Sprintf("%d/%s", side, carrier), func(t *testing.T) {
				cfg := l3Spec{Side: side, Carrier: carrier, LocalIP: "10.243.81.1/30", PeerIP: "10.243.81.2"}
				restore := tui.SetInput(strings.NewReader("10.243.82.1/30\n10.243.82.2\n"))
				defer restore()
				out := capture(t, func() { askL3TunnelAddresses(&cfg, side) })
				if cfg.LocalIP != "10.243.82.1/30" || cfg.PeerIP != "10.243.82.2" {
					t.Fatal("manual address answers were not applied")
				}
				peer := "Kharej"
				if side == sideKharej {
					peer = "Iran"
				}
				if !strings.Contains(out, "This Server's Tunnel Address") || !strings.Contains(out, "The "+peer+" Server's Tunnel Address") {
					t.Fatal("missing local or correctly labelled peer prompt")
				}
			})
		}
	}
}

func TestManualDirectAddressesKeepDefaultsAndRetryInvalidPair(t *testing.T) {
	for _, input := range []string{"\n\n", "bad\n10.243.81.2\n10.243.82.1/30\n10.243.82.2\n", "10.243.81.1/30\n10.243.81.2/30\n10.243.82.1/30\n10.243.82.2\n", "10.243.81.2/30\n10.243.81.2\n10.243.82.1/30\n10.243.82.2\n"} {
		cfg := l3Spec{Side: sideIran, Carrier: "xdi", LocalIP: "10.243.81.1/30", PeerIP: "10.243.81.2"}
		restore := tui.SetInput(strings.NewReader(input))
		capture(t, func() { askL3TunnelAddresses(&cfg, sideIran) })
		restore()
		want := "10.243.81.1/30"
		if input != "\n\n" {
			want = "10.243.82.1/30"
		}
		if cfg.LocalIP != want {
			t.Fatalf("got local %q, want %q", cfg.LocalIP, want)
		}
	}
}
