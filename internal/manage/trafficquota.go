package manage

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/backpack/backpack/internal/app"
)

var trafficQuotaLine = regexp.MustCompile(`(?m)^traffic_limit_gb\s*=\s*\d+\s*\n?`)

func readTrafficLimit(name string) int64 {
	c, err := LoadTunnelConfig(name)
	if err != nil {
		return 0
	}
	return c.TrafficLimitGB
}

// SetTrafficQuota sets a cumulative allowance for either tunnel role or
// direction. The history file stays in place, so a restart cannot reset it.
func SetTrafficQuota(name string, gb int64) error {
	if gb < 0 || uint64(gb) > ^uint64(0)>>30 {
		return fmt.Errorf("traffic quota must be between 0 and %d GiB", ^uint64(0)>>30)
	}
	path := app.ConfigPath(name)
	old, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	newBody := trafficQuotaLine.ReplaceAllString(string(old), "")
	if gb > 0 {
		newBody = fmt.Sprintf("traffic_limit_gb = %d\n", gb) + strings.TrimLeft(newBody, "\n")
	}
	if err := app.WriteFileAtomic(path, []byte(newBody), app.TunnelConfigMode); err != nil {
		return err
	}
	if err := RestartService(app.ServiceName(name)); err != nil {
		_ = app.WriteFileAtomic(path, old, app.TunnelConfigMode)
		_ = RestartService(app.ServiceName(name))
		return fmt.Errorf("could not restart tunnel with traffic quota: %w", err)
	}
	return nil
}
