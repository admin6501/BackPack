package manage

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/app"
)

var trafficQuotaLine = regexp.MustCompile(`(?m)^traffic_limit_gb\s*=\s*\d+\s*\n?`)
var trafficQuotaModeLine = regexp.MustCompile(`(?m)^traffic_limit_mode\s*=\s*"[^"]*"\s*\n?`)

func readTrafficLimit(name string) int64 {
	c, err := LoadTunnelConfig(name)
	if err != nil {
		return 0
	}
	return c.TrafficLimitGB
}

func readTrafficLimitMode(name string) string {
	c, err := LoadTunnelConfig(name)
	if err != nil {
		return "both"
	}
	return config.NormalizeTrafficLimitMode(c.TrafficLimitMode)
}

// SetTrafficQuota sets a cumulative allowance for either tunnel role or
// direction. The history file stays in place, so a restart cannot reset it.
func SetTrafficQuota(name string, gb int64) error {
	return SetTrafficQuotaWithMode(name, gb, readTrafficLimitMode(name))
}

// SetTrafficQuotaWithMode changes the allowance and accounting direction in
// one config write and one service restart. Existing usage is retained.
func SetTrafficQuotaWithMode(name string, gb int64, mode string) error {
	if gb < 0 || uint64(gb) > ^uint64(0)>>30 {
		return fmt.Errorf("traffic quota must be between 0 and %d GiB", ^uint64(0)>>30)
	}
	if mode != "both" && mode != "download" && mode != "upload" {
		return fmt.Errorf("traffic quota mode must be both, download, or upload")
	}
	path := app.ConfigPath(name)
	old, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	newBody := trafficQuotaLine.ReplaceAllString(string(old), "")
	newBody = trafficQuotaModeLine.ReplaceAllString(newBody, "")
	if mode != "both" {
		newBody = fmt.Sprintf("traffic_limit_mode = %q\n", mode) + strings.TrimLeft(newBody, "\n")
	}
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
