package backup

import (
	"fmt"
	"strings"
)

type restoreSystemctl func(...string) (string, error)

func pauseRestoreWriters(services []string, run restoreSystemctl) (func() []string, error) {
	var paused []string
	resume := func() []string {
		var failed []string
		// Start tunnels first, monitor last, so it never samples half-restored
		// services or restarts them while their accounting baseline is changing.
		for i := len(paused) - 1; i >= 0; i-- {
			if _, err := run("start", paused[i]); err != nil {
				failed = append(failed, paused[i])
			}
		}
		return failed
	}
	for _, service := range services {
		state, err := run("is-active", service)
		switch strings.TrimSpace(state) {
		case "inactive", "failed", "unknown":
			continue
		case "active", "activating", "reloading":
		default:
			return resume, fmt.Errorf("cannot verify restore writer %s: %v (%s)", service, err, state)
		}
		paused = append(paused, service)
		if _, err := run("stop", service); err != nil {
			return resume, fmt.Errorf("cannot stop %s before restoring usage: %w", service, err)
		}
	}
	return resume, nil
}
