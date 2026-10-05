package manage

import (
	"strings"
	"testing"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tui"
)

func TestDirectPortsPromptRequiresAValidNonemptyList(t *testing.T) {
	restore := tui.SetInput(strings.NewReader("\n , , \nnot-a-port\n30814,42221\n"))
	defer restore()
	var ports []string
	var ok bool
	out := capture(t, func() { ports, ok = askRequiredDirectPorts("Forwarded Ports (Required): ") })
	if !ok || strings.Join(ports, ",") != "30814,42221" {
		t.Fatalf("ports=%v ok=%v", ports, ok)
	}
	if !strings.Contains(out, "at least one forwarded port is required") {
		t.Fatal("blank input was not explained")
	}
}

func TestDirectPortsPromptStopsAtEndOfInput(t *testing.T) {
	restore := tui.SetInput(strings.NewReader("\n"))
	defer restore()
	capture(t, func() {
		if _, ok := askRequiredDirectPorts("Ports: "); ok {
			t.Fatal("EOF accepted an empty list")
		}
	})
}

func TestAllDirectCarriersRequireIranPortsOnCreateAndEdit(t *testing.T) {
	for _, carrier := range []string{"xdi", "pck", "udp", "quic", "spoof", "sni", "gre-fou"} {
		for _, mode := range []string{"dial", "listen"} {
			for _, raw := range []string{"", "  ", ", ,"} {
				n := NewDirectTunnel{Side: "iran", Mode: mode, Carrier: carrier, Name: "required-ports", Token: "test-token", PeerAddr: "203.0.113.9", TunnelPort: "9000", Ports: raw}
				if _, err := n.spec(); err == nil || !strings.Contains(err.Error(), "forwarded port") {
					t.Fatalf("%s/%s create blank: %v", carrier, mode, err)
				}
				before := config.L3Config{Side: "iran", Mode: mode, Carrier: carrier, Ports: []string{"30814", "42221"}}
				after, err := applyDirectEdit(before, DirectEdit{Ports: &raw})
				if err == nil || strings.Join(after.Ports, ",") != "30814,42221" {
					t.Fatalf("%s/%s edit wiped old ports: %v %v", carrier, mode, after.Ports, err)
				}
			}
		}
	}
}
