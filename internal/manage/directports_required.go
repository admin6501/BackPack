package manage

import (
	"fmt"

	"github.com/backpack/backpack/internal/tui"
)

// Every operator-created Iran end must expose a service. Kharej has no
// forwarded-port list: the paired Iran end owns the mappings.
func requiredDirectPorts(raw string) ([]string, error) {
	ports := parsePorts(raw)
	if len(ports) == 0 {
		return nil, fmt.Errorf("at least one forwarded port is required on the Iran side")
	}
	if err := validatePortSpecs(ports); err != nil {
		return nil, err
	}
	return ports, nil
}

func askRequiredDirectPorts(label string) ([]string, bool) {
	for {
		raw, ok := tui.PromptOrEnd(label)
		if !ok {
			return nil, false
		}
		ports, err := requiredDirectPorts(raw)
		if err == nil {
			return ports, true
		}
		tui.Error(err.Error())
	}
}
