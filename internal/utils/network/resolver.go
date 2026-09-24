package network

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func ResolveRemoteAddr(remoteAddr string) (int, string, error) {
	// A pipe list names several backends for health-checked load balancing
	// ("8443|127.0.0.1:8444"). Resolve each to a full host:port (so bare ports
	// still default to localhost) and pass the rebuilt list through for the pool
	// to split; the reported port is the first backend's, for metrics and logs.
	//
	// A pipe, not a comma: commas already separate whole port entries.
	if strings.Contains(remoteAddr, "|") {
		var resolved []string
		var firstPort int
		for _, part := range strings.Split(remoteAddr, "|") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			p, full, err := ResolveRemoteAddr(part)
			if err != nil {
				return 0, "", err
			}
			if len(resolved) == 0 {
				firstPort = p
			}
			resolved = append(resolved, full)
		}
		return firstPort, strings.Join(resolved, "|"), nil
	}

	host, portText := "127.0.0.1", strings.TrimSpace(remoteAddr)
	if strings.Contains(portText, ":") {
		var err error
		host, portText, err = net.SplitHostPort(portText)
		if err != nil {
			return 0, "", fmt.Errorf("invalid remote address: %w", err)
		}
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return 0, "", fmt.Errorf("invalid port format: %q", portText)
	}
	return port, net.JoinHostPort(host, strconv.Itoa(port)), nil
}
