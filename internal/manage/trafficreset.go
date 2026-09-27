package manage

import (
	"fmt"

	"github.com/backpack/backpack/internal/app"
	"github.com/backpack/backpack/internal/metrics"
)

// ResetTunnelTraffic resets only this server's recorded in/out totals. A quota
// on this end starts again at its configured limit. The other server's history
// is independent and is deliberately left alone.
func ResetTunnelTraffic(name string) error {
	if err := CheckName(name); err != nil {
		return err
	}
	t, ok := Find(name)
	if !ok {
		return fmt.Errorf("no such tunnel %q", name)
	}
	wasRunning := IsActive(t.Service)
	if err := StopService(t.Service); err != nil {
		return fmt.Errorf("could not stop tunnel before resetting traffic: %w", err)
	}
	if err := metrics.ResetTraffic(app.ConfigDir, name); err != nil {
		if wasRunning {
			if startErr := StartService(t.Service); startErr != nil {
				return fmt.Errorf("traffic was not reset (%v); tunnel could not restart: %w", err, startErr)
			}
		}
		return fmt.Errorf("traffic was not reset: %w", err)
	}
	if wasRunning {
		if err := StartService(t.Service); err != nil {
			return fmt.Errorf("traffic was reset, but tunnel could not restart: %w", err)
		}
	}
	return nil
}
