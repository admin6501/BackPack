package manage

import (
	"reflect"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/tui"
)

func TestKharejSpoofWizardAsksForRealIranAddress(t *testing.T) {
	// Defaults for the profile and stealth, then an invalid and valid peer IP.
	restore := tui.SetInput(strings.NewReader("\n\nnot-an-ip\n203.0.113.10\n\n\n"))
	defer restore()
	var sc config.SpoofConfig
	askSpoofCarrier(&sc, false)
	if sc.SpoofPeerIP != "203.0.113.10" {
		t.Fatalf("kharej peer IP = %q, want the real Iran address", sc.SpoofPeerIP)
	}
	if sc.SpoofProfile != "udp" || sc.SpoofSrcIP != "" {
		t.Fatalf("unexpected profile or forged source: %+v", sc)
	}
}

// The forged-source carrier belongs to the direct tunnel now, and its settings
// have to survive a render and come back — that round trip is what the edit
// flow stands on. An edit re-renders the whole file from the spec, so a key the
// render drops is a setting the operator loses the next time they change the
// MTU, silently.
//
// This replaces the same guarantee that used to be held for the reverse spoof
// transport, which no longer exists: a reverse tunnel over a forged source
// could never carry traffic, because its pooled sessions all arrive at one
// address and kcp-go keys sessions on that address. See cmd/defaults.go.

func TestSpoofCarrierRenderRoundTrip(t *testing.T) {
	spec := l3Spec{
		Name:    "iran-spoof",
		Side:    sideIran,
		Carrier: "spoof",
		Addr:    "203.0.113.9:9000",
		Token:   "spoof-token-0123456789abcdefghij",
		LocalIP: "10.10.0.1/30",
		PeerIP:  "10.10.0.2",
		Ports:   []string{"443"},
		Spoof: config.SpoofConfig{
			SpoofProfile:     "icmp",
			SpoofUplink:      "icmp",
			SpoofDownlink:    "udp",
			SpoofSrcIP:       "81.28.60.1",
			SpoofSrcPool:     []string{"81.28.60.1", "81.28.60.2"},
			SpoofPeerIP:      "38.87.117.94",
			SpoofPeerSrcIP:   "8.8.4.4",
			SpoofInterface:   "eth0",
			SpoofSockBuf:     8 << 20,
			SpoofMTU:         1400,
			SpoofICMPReply:   true,
			SpoofTTLJitter:   true,
			SpoofRandomDSCP:  true,
			SpoofShufflePort: true,
			SpoofPortMin:     49152,
			SpoofPortMax:     65535,
			SpoofPadding:     true,
			SpoofPaddingMax:  64,
		},
	}

	out := spec.render()
	if !strings.Contains(out, "[l3]") {
		t.Fatalf("the carrier was not rendered into an [l3] table:\n%s", out)
	}

	var got config.Config
	if _, err := toml.Decode(out, &got); err != nil {
		t.Fatalf("the rendered config does not parse: %v\n%s", err, out)
	}
	if !got.L3.Enabled() {
		t.Fatal("the rendered config does not describe a layer-3 tunnel")
	}
	if got.L3.Carrier != "spoof" {
		t.Errorf("carrier = %q, want spoof", got.L3.Carrier)
	}
	if !reflect.DeepEqual(got.L3.SpoofConfig, spec.Spoof) {
		t.Errorf("the carrier did not survive the round trip:\n got %+v\nwant %+v",
			got.L3.SpoofConfig, spec.Spoof)
	}

	// And nothing may leak into the reverse tables, which no longer have a
	// carrier of their own to read these keys with.
	if strings.Contains(out, "[server]") || strings.Contains(out, "[client]") {
		t.Errorf("a direct tunnel rendered a reverse table:\n%s", out)
	}
}

// A carrier with nothing set must render nothing, so a udp or pck tunnel does
// not carry a block of empty spoof keys it never reads.
func TestAnUnsetSpoofCarrierRendersNothing(t *testing.T) {
	spec := l3Spec{
		Name: "plain", Side: sideIran, Carrier: "udp",
		Addr: "203.0.113.9:9000", Token: "a-token-0123456789abcdefghijklmno",
		LocalIP: "10.10.0.1/30", PeerIP: "10.10.0.2",
	}
	if out := spec.render(); strings.Contains(out, "spoof_") {
		t.Errorf("a tunnel with no forged source rendered spoof keys:\n%s", out)
	}
}

func TestSetupAgainCanTurnExistingStealthOff(t *testing.T) {
	sc := config.SpoofConfig{SpoofPeerIP: "203.0.113.10"}
	applySpoofStealth(&sc)
	input := "\n\n\n\n"
	if len(routableInterfaces()) > 1 {
		input += "\n"
	}
	input += "n\n"
	restore := tui.SetInput(strings.NewReader(input))
	defer restore()
	askSpoofCarrier(&sc, true)
	if spoofStealthOn(sc) || sc.SpoofPaddingMax != 0 || sc.SpoofPortMin != 0 {
		t.Fatal("setup again kept previous stealth flags", sc)
	}
}
