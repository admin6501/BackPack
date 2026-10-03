package backup

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/backpack/backpack/config"
)

// Only archived listener configurations are edited. Clients/diallers, forged
// sources, real peer addresses, private tunnel IPs and all JSON/accounting files
// are left byte-for-byte as staged from the backup.
func migrateServerIPs(stage string, names []string, destination string) ([]string, error) {
	ip := net.ParseIP(destination)
	if ip == nil || ip.IsUnspecified() || ip.IsMulticast() {
		return nil, fmt.Errorf("invalid destination server IP %q", destination)
	}
	var migrated []string
	for _, name := range names {
		path := filepath.Join(stage, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var cfg config.Config
		if _, err := toml.Decode(string(data), &cfg); err != nil {
			return nil, fmt.Errorf("cannot migrate %s: %w", name, err)
		}
		section, key := "", ""
		switch {
		case cfg.Client.RemoteAddr != "":
			continue
		case cfg.Server.BindAddr != "":
			section, key = "server", "bind_addr"
		case cfg.Direct.Enabled() && cfg.Direct.ResolvedRole() == "origin":
			section, key = "direct", "addr"
		case cfg.L3.Enabled() && strings.EqualFold(strings.TrimSpace(cfg.L3.Mode), "listen"):
			section, key = "l3", "addr"
		default:
			continue
		}
		var tree map[string]any
		if _, err := toml.Decode(string(data), &tree); err != nil {
			return nil, err
		}
		table, ok := tree[section].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("missing %s table in %s", section, name)
		}
		changed := false
		if addr, ok := table[key].(string); ok {
			next, err := migrateBindAddress(addr, destination)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			if next != addr {
				table[key], changed = next, true
			}
		}
		// A reverse server may also pin individual exposed ports. Rewrite only
		// their local (left) endpoint, never any backend after '='.
		if section == "server" {
			if ports, ok := table["ports"].([]any); ok {
				for i, value := range ports {
					p, ok := value.(string)
					if !ok {
						continue
					}
					parts := strings.SplitN(p, "=", 2)
					next, err := migrateBindAddress(parts[0], destination)
					if err != nil {
						return nil, fmt.Errorf("%s port %q: %w", name, p, err)
					}
					if next != parts[0] {
						parts[0] = next
						ports[i] = strings.Join(parts, "=")
						changed = true
					}
				}
			}
		}
		if !changed {
			continue
		}
		var out bytes.Buffer
		if err := toml.NewEncoder(&out).Encode(tree); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, out.Bytes(), 0600); err != nil {
			return nil, err
		}
		migrated = append(migrated, strings.TrimSuffix(filepath.Base(name), ".toml"))
	}
	return migrated, nil
}

func migrateBindAddress(addr, destination string) (string, error) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		// A port or port range alone binds all interfaces already.
		if !strings.Contains(addr, ":") {
			return addr, nil
		}
		return "", fmt.Errorf("invalid listener address %q", addr)
	}
	ip := net.ParseIP(host)
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() {
		return addr, nil
	}
	return net.JoinHostPort(destination, port), nil
}

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
