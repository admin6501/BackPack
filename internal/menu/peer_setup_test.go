package menu

import (
	"github.com/backpack/backpack/internal/node"
	"strings"
	"testing"

	"github.com/backpack/backpack/internal/manage"
)

func TestPeerApplyRequestMirrorsBothDirections(t *testing.T) {
	for _, tc := range []struct {
		name, kind, from, host, wantRole string
	}{
		{"iran-reverse", "reverse", "iran", "203.0.113.10", "client"},
		{"kharej-reverse", "reverse", "kharej", "", "server"},
		{"iran-direct", "direct", "iran", "", "kharej"},
		{"kharej-direct", "direct", "kharej", "203.0.113.11", "iran"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			link := manage.ShareLink{Kind: tc.kind, From: tc.from, Name: "test", Tok: "secret",
				Tr: "udp", Port: "9000", Host: tc.host, Ports: "443", LocalIP: "10.10.0.1/30", PeerIP: "10.10.0.2"}
			req, name, err := peerApplyRequest(link)
			if err != nil || name == "" {
				t.Fatalf("peer request: %q, %v", name, err)
			}
			if tc.kind == "reverse" {
				if req.Tunnel == nil || req.Tunnel.Role != tc.wantRole || req.Tunnel.Token != "secret" {
					t.Fatalf("wrong reverse peer: %+v", req.Tunnel)
				}
			} else if req.Direct == nil || req.Direct.Side != tc.wantRole || req.Direct.Token != "secret" {
				t.Fatalf("wrong direct peer: %+v", req.Direct)
			}
		})
	}
}

func TestPeerApplyRequestRefusesMissingDialAddress(t *testing.T) {
	for _, link := range []manage.ShareLink{
		{Kind: "reverse", From: "iran", Name: "test", Tok: "secret", Tr: "tcp", Port: "9000"},
		{Kind: "direct", From: "kharej", Name: "test", Tok: "secret", Tr: "udp", Port: "9000"},
	} {
		if _, _, err := peerApplyRequest(link); err == nil {
			t.Errorf("accepted a peer with no dial address: %+v", link)
		}
	}
}

func TestDirectPeerFromKharejAsksForIranPortsBeforeSSH(t *testing.T) {
	for _, input := range []string{"443=127.0.0.1:2096\n", "\n", "bad-port\n"} {
		req := node.ApplyRequest{Kind: "direct", Direct: &manage.NewDirectTunnel{Side: "iran"}}
		var err error
		output := drive(t, input, func() { err = completePeerPorts(&req) })
		if !strings.Contains(output, "Ports to expose on the Iran server") {
			t.Fatal("Iran ports were not requested")
		}
		if (err == nil) != (strings.HasPrefix(input, "443=")) {
			t.Fatalf("input %q accepted incorrectly: %v", input, err)
		}
		if err == nil && req.Direct.Ports != "443=127.0.0.1:2096" {
			t.Fatal("ports not passed into peer request")
		}
	}
}

func TestGREFOUPeerSetupPreservesInitiationAndRequiresListenerAddress(t *testing.T) {
	for _, side := range []string{"iran", "kharej"} {
		for _, mode := range []string{"dial", "listen"} {
			direction := "direct"
			if (side == "iran") != (mode == "dial") {
				direction = "reverse"
			}
			local := manage.Tunnel{Role: side, Transport: "l3/gre-fou", Direction: direction}
			if peerSetupNeedsHost(local) != (mode == "listen") {
				t.Fatalf("%s/%s: wrong reachable-address prompt", side, mode)
			}
			link := manage.ShareLink{Kind: "direct", From: side, Mode: mode, Tr: "gre-fou", Name: "existing", Tok: "same-secret", Port: "1992", LocalIP: "10.10.1.1/30", PeerIP: "10.10.1.2", GREKey: 17}
			if mode == "listen" {
				if _, _, err := peerApplyRequest(link); err == nil {
					t.Fatal("dialing peer accepted without a reachable address")
				}
				link.Host = "203.0.113.7"
			}
			req, _, err := peerApplyRequest(link)
			if err != nil {
				t.Fatalf("%s/%s: %v", side, mode, err)
			}
			wantMode := "listen"
			if mode == "listen" {
				wantMode = "dial"
			}
			if req.Direct == nil || req.Direct.Mode != wantMode || req.Direct.Side == side || req.Direct.Token != link.Tok || req.Direct.LocalIP != "10.10.1.2/30" || req.Direct.PeerIP != "10.10.1.1" || req.Direct.GREKey != 17 {
				t.Fatalf("bad mirrored request: %+v", req.Direct)
			}
			if wantMode == "dial" && req.Direct.PeerAddr != link.Host {
				t.Fatal("listener address not carried to dialer")
			}
			if wantMode == "listen" && req.Direct.PeerAddr != "" {
				t.Fatal("listener was given a dial address")
			}
		}
	}
}

func TestPeerSetupHostPromptKeepsLegacyDirections(t *testing.T) {
	for _, tc := range []struct {
		role, transport string
		want            bool
	}{
		{"server", "tcp", true}, {"client", "tcp", false},
		{"iran", "direct/tcp", false}, {"kharej", "direct/tcp", true},
		{"iran", "l3/spoof", true}, {"kharej", "l3/spoof", true},
	} {
		if peerSetupNeedsHost(manage.Tunnel{Role: tc.role, Transport: tc.transport}) != tc.want {
			t.Fatalf("wrong host prompt for %s/%s", tc.role, tc.transport)
		}
	}
}
