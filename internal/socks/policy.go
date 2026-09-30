package socks

import (
	"context"
	"errors"
	"net"
	"syscall"
	"time"
)

// The proxy is often reached through a forwarded port. Refuse destinations
// that could expose services local to the exit server or cloud metadata.
// Target is replaceable by loopback-only integration tests.
var Target = PublicOrPrivate

func PublicOrPrivate(ip net.IP) bool {
	return ip != nil && !(ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() ||
		ip.Equal(net.ParseIP("fd00:ec2::254")))
}

var ErrRefusedTarget = errors.New("proxy destination is local or link-local")

// Control receives the address after DNS resolution, which also blocks names
// that resolve to loopback or metadata IPs.
func targetControl(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return ErrRefusedTarget
	}
	ip := net.ParseIP(host)
	if !Target(ip) {
		return ErrRefusedTarget
	}
	return nil
}

func DialTargetContext(ctx context.Context, network, addr string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout, Control: targetControl}
	return d.DialContext(ctx, network, addr)
}

func DialTarget(network, addr string, timeout time.Duration) (net.Conn, error) {
	return DialTargetContext(context.Background(), network, addr, timeout)
}
