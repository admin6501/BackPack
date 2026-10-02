package config

import "math"

// NormalizeTrafficLimitMode keeps old and partially edited configs on the
// historical behavior. Unknown values are treated as "both" by readers.
func NormalizeTrafficLimitMode(mode string) string {
	switch mode {
	case "download", "upload":
		return mode
	default:
		return "both"
	}
}

// TrafficQuotaUsage returns the bytes charged to a tunnel quota. Download and
// upload are named from the Iran user's perspective, so client/origin counters
// are reversed from server/edge counters.
func TrafficQuotaUsage(mode, role string, in, out uint64) uint64 {
	switch NormalizeTrafficLimitMode(mode) {
	case "download":
		if role == "client" || role == "kharej-origin" || role == "origin" {
			return out
		}
		return in
	case "upload":
		if role == "client" || role == "kharej-origin" || role == "origin" {
			return in
		}
		return out
	default:
		if math.MaxUint64-in < out {
			return math.MaxUint64
		}
		return in + out
	}
}
