package l3

import (
	"strings"
	"testing"
)

func TestTunnelEndsSubnetRouting(t *testing.T) {
	tests := []struct {
		name, local, peer string
		wantError         string
	}{
		{"normal /30", "10.10.2.2/30", "10.10.2.1", ""},
		{"missing reverse route", "10.10.1.4/30", "10.10.1.3", "outside local_ip subnet"},
		{"network as local", "10.10.1.4/30", "10.10.1.5", "network and broadcast"},
		{"broadcast as peer", "10.10.1.5/30", "10.10.1.7", "network and broadcast"},
		{"broadcast as local", "10.10.1.3/30", "10.10.1.2", "network and broadcast"},
		{"network as peer", "10.10.1.2/30", "10.10.1.0", "network and broadcast"},
		{"point-to-point", "10.10.1.4", "10.10.1.3", ""},
		{"host prefix", "10.10.1.4/32", "10.10.1.3", ""},
		{"/31 pair", "10.10.1.4/31", "10.10.1.5", ""},
		{"IPv6 pair", "fd00::1/126", "fd00::2", ""},
		{"IPv6 outside prefix", "fd00::1/126", "fd00::4", "outside local_ip subnet"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTunnelEnds(tt.local, tt.peer)
			if tt.wantError == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantError != "" && (err == nil || !strings.Contains(err.Error(), tt.wantError)) {
				t.Fatalf("got %v, want %q", err, tt.wantError)
			}
		})
	}
}
