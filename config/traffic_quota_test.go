package config

import "testing"

func TestTrafficQuotaUsageUsesIranUserDirectionsOnBothEnds(t *testing.T) {
	const gib = uint64(1 << 30)
	tests := []struct {
		name, mode, role string
		in, out, want    uint64
	}{
		{"legacy sums both", "", "server", gib, gib / 2, gib + gib/2},
		{"iran download is inbound", "download", "server", gib, 0, gib},
		{"iran upload is outbound", "upload", "iran-edge", 0, gib, gib},
		{"direct iran download is inbound", "download", "edge", gib, 0, gib},
		{"direct kharej download is outbound", "download", "origin", 0, gib, gib},
		{"kharej download is outbound", "download", "client", 0, gib, gib},
		{"kharej upload is inbound", "upload", "kharej-origin", gib, 0, gib},
		{"unselected direction is excluded", "download", "client", gib, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrafficQuotaUsage(tt.mode, tt.role, tt.in, tt.out); got != tt.want {
				t.Fatalf("TrafficQuotaUsage(%q, %q, %d, %d) = %d, want %d", tt.mode, tt.role, tt.in, tt.out, got, tt.want)
			}
		})
	}
}
