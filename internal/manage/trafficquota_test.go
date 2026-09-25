package manage

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/backpack/backpack/config"
)

func TestTrafficQuotaRendersForEveryTunnelFamily(t *testing.T) {
	for name, body := range map[string]string{
		"reverse": (TunnelSpec{Name: "example", Role: "server", TrafficLimitGB: 5}).Render(),
		"direct":  (directSpec{TrafficLimitGB: 5}).render(),
		"l3":      (l3Spec{TrafficLimitGB: 5}).render(),
	} {
		var cfg config.Config
		if _, err := toml.Decode(body, &cfg); err != nil {
			t.Fatalf("%s TOML: %v", name, err)
		}
		if cfg.TrafficLimitGB != 5 || !strings.Contains(body, "traffic_limit_gb = 5") {
			t.Fatalf("%s did not retain quota", name)
		}
	}
}

func TestOlderLimitEditorDoesNotClearTrafficQuota(t *testing.T) {
	s := TunnelSpec{TrafficLimitGB: 7}
	if err := (TunnelLimits{BandwidthMbps: 50}).apply(&s); err != nil || s.TrafficLimitGB != 7 {
		t.Fatalf("older editor changed quota: %d, %v", s.TrafficLimitGB, err)
	}
	zero := int64(0)
	if err := (TunnelLimits{TrafficLimitGB: &zero}).apply(&s); err != nil || s.TrafficLimitGB != 0 {
		t.Fatalf("explicit zero failed to clear quota: %d, %v", s.TrafficLimitGB, err)
	}
}
