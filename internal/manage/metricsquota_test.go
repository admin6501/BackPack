package manage

import (
	"strings"
	"testing"
)

func TestTrafficQuotaStatusTracksRemainingAllowance(t *testing.T) {
	const gib = uint64(1 << 30)
	for _, tc := range []struct {
		name  string
		limit int64
		in    uint64
		out   uint64
		want  string
	}{
		{"unlimited", 0, gib, gib, "unlimited"},
		{"partly used", 3, gib, gib / 2, "1.5 GiB remaining"},
		{"exhausted", 1, gib / 2, gib / 2, "0 B remaining (exhausted)"},
		{"over limit", 1, gib, gib, "0 B remaining (exhausted)"},
		{"raised limit keeps usage", 4, gib, gib, "2.0 GiB remaining"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := trafficQuotaStatus(tc.limit, tc.in, tc.out)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("quota status %q does not contain %q", got, tc.want)
			}
		})
	}
}
