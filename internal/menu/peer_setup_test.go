package menu

import (
	"strings"
	"testing"

	"github.com/backpack/backpack/internal/manage"
	"github.com/backpack/backpack/internal/node"
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

func TestKharejInitiatedPeerSetupAsksForIranPortsInBothDirections(t *testing.T) {
	for _, kind := range []string{"reverse", "direct"} {
		t.Run(kind, func(t *testing.T) {
			request, _, err := peerApplyRequest(manage.ShareLink{Kind: kind, From: "kharej", Host: "203.0.113.9", Tr: "udp", Port: "9000", Tok: "secret"})
			if err != nil {
				t.Fatal(err)
			}
			out := drive(t, "23298=127.0.0.1:2096, 53835\n", func() { err = completePeerPorts(&request) })
			if err != nil || !strings.Contains(out, "Ports to expose on the Iran server") {
				t.Fatalf("port prompt: %v %s", err, out)
			}
			ports := ""
			if request.Tunnel != nil {
				ports = request.Tunnel.Ports
			}
			if request.Direct != nil {
				ports = request.Direct.Ports
			}
			if ports != "23298=127.0.0.1:2096, 53835" {
				t.Fatal("Iran ports lost")
			}
		})
	}
	request := node.ApplyRequest{Direct: &manage.NewDirectTunnel{Side: "iran"}}
	var err error
	drive(t, "\n", func() { err = completePeerPorts(&request) })
	if err == nil {
		t.Fatal("empty Iran ports accepted")
	}
	request.Tunnel = &manage.NewTunnel{Role: "server", Ports: "not-a-port"}
	request.Direct = nil
	if err := completePeerPorts(&request); err == nil {
		t.Fatal("invalid Iran ports accepted")
	}
	request = node.ApplyRequest{Tunnel: &manage.NewTunnel{Role: "client"}}
	if err := completePeerPorts(&request); err != nil {
		t.Fatal("kharej client was asked for exposed ports")
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
