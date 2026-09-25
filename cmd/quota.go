package cmd

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/metrics"
)

type quotaContextKey struct{}

type quotaGuard struct {
	limit  uint64
	cancel context.CancelFunc
}

// trafficQuotaBytes uses binary gigabytes, matching the panel's byte display.
func trafficQuotaBytes(gb int64) (uint64, error) {
	if gb < 0 || uint64(gb) > math.MaxUint64>>30 {
		return 0, fmt.Errorf("traffic_limit_gb must be between 0 and %d", uint64(math.MaxUint64)>>30)
	}
	return uint64(gb) << 30, nil
}

func quotaExhausted(path string, cfg *config.Config) (bool, error) {
	limit, err := trafficQuotaBytes(cfg.TrafficLimitGB)
	if err != nil || limit == 0 {
		return false, err
	}
	s, err := metrics.Read(filepath.Dir(path), tunnelNameFromPath(path))
	if err != nil {
		// A missing history is normal for a newly created tunnel. A damaged
		// history must not silently reset a configured allowance.
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("cannot read traffic history: %w", err)
	}
	return s.BytesIn >= limit || s.BytesOut >= limit || s.BytesIn >= limit-s.BytesOut, nil
}

// watch ends a generation when its cumulative traffic reaches the allowance.
// The engine's normal shutdown closes forwarded listeners and live sessions.
func (q *quotaGuard) watch(ctx context.Context, c *metrics.Collector) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		s := c.Snapshot()
		if s.BytesIn >= q.limit || s.BytesOut >= q.limit || s.BytesIn >= q.limit-s.BytesOut {
			if err := c.Write(); err != nil {
				logger.Errorf("traffic quota reached; cannot persist history: %v", err)
			}
			logger.Warnf("traffic quota reached for %s (%d GiB); pausing tunnel", s.Name, q.limit>>30)
			q.cancel()
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
