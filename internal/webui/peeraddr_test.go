package webui

import (
	"encoding/json"
	"testing"
)

func TestTunnelPeerAddrUsesTheFarEndForEachRole(t *testing.T) {
	tests := []struct {
		name       string
		dialsOut   bool
		resolved   string
		peers      []peerConn
		snapshot   string
		want       string
	}{
		{
			name:     "client uses resolved remote IP",
			dialsOut: true,
			resolved: "203.0.113.8",
			want:     "203.0.113.8",
		},
		{
			name:       "client prefers the currently connected peer over DNS",
			dialsOut:   true,
			resolved:   "203.0.113.8",
			snapshot:   "203.0.113.9:2433",
			want:       "203.0.113.9",
		},
		{
			name:     "client does not label an unresolved hostname as an IP",
			dialsOut: true,
			want:     "",
		},
		{
			name:     "listener uses connected peer IP",
			peers:    []peerConn{{IP: "198.51.100.14", RTT: 30}},
			snapshot: "198.51.100.15:44120",
			want:     "198.51.100.14",
		},
		{
			name:     "listener falls back to transport snapshot",
			snapshot: "198.51.100.15:44120",
			want:     "198.51.100.15",
		},
		{
			name: "listener has no far-end IP before a peer connects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tunnelPeerAddr(tt.dialsOut, tt.resolved, tt.peers, tt.snapshot)
			if got != tt.want {
				t.Errorf("tunnelPeerAddr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTunnelPeerAddrIsInTheTunnelAPI(t *testing.T) {
	raw, err := json.Marshal(TunnelInfo{PeerAddr: "203.0.113.8"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["peerAddr"] != "203.0.113.8" {
		t.Errorf("peerAddr in API = %v, want 203.0.113.8", got["peerAddr"])
	}
}
