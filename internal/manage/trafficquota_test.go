package manage

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/backpack/backpack/config"
)

func TestTrafficQuotaRendersForEveryTunnelFamily(t *testing.T) {
	for name, body := range map[string]string{
		"reverse": (TunnelSpec{Name: "example", Role: "server", TrafficLimitGB: 5, TrafficLimitMode: "download"}).Render(),
		"direct":  (directSpec{TrafficLimitGB: 5, TrafficLimitMode: "download"}).render(),
		"l3":      (l3Spec{TrafficLimitGB: 5, TrafficLimitMode: "download"}).render(),
	} {
		var cfg config.Config
		if _, err := toml.Decode(body, &cfg); err != nil {
			t.Fatalf("%s TOML: %v", name, err)
		}
		if cfg.TrafficLimitGB != 5 || cfg.TrafficLimitMode != "download" || !strings.Contains(body, "traffic_limit_gb = 5") {
			t.Fatalf("%s did not retain quota", name)
		}
	}
}

func TestQuotaModeDefaultsToBothAndRejectsUnknownValues(t *testing.T) {
	if got := config.NormalizeTrafficLimitMode(""); got != "both" {
		t.Fatalf("empty mode = %q", got)
	}
	if got := config.NormalizeTrafficLimitMode("nonsense"); got != "both" {
		t.Fatalf("invalid mode = %q", got)
	}
	if err := (TunnelLimits{TrafficLimitMode: "nonsense"}).apply(&TunnelSpec{}); err == nil {
		t.Fatal("invalid mode accepted")
	}
}

func TestOlderLimitEditorDoesNotClearTrafficQuota(t *testing.T) {
	s := TunnelSpec{TrafficLimitGB: 7, TrafficLimitMode: "upload"}
	if err := (TunnelLimits{BandwidthMbps: 50}).apply(&s); err != nil || s.TrafficLimitGB != 7 {
		t.Fatalf("older editor changed quota: %d, %v", s.TrafficLimitGB, err)
	}
	zero := int64(0)
	if err := (TunnelLimits{TrafficLimitGB: &zero}).apply(&s); err != nil || s.TrafficLimitGB != 0 {
		t.Fatalf("explicit zero failed to clear quota: %d, %v", s.TrafficLimitGB, err)
	}
	if s.TrafficLimitMode != "upload" {
		t.Fatalf("older editor changed quota mode: %q", s.TrafficLimitMode)
	}
}
