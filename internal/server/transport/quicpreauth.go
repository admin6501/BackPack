package transport

import (
	"net"
	"net/netip"
	"strings"
	"sync"
)

const (
	// QUIC transport handshakes are not tunnel authentication. Keep the
	// number of connections waiting for a valid tunnel token bounded across
	// the whole listener, and per peer so one source cannot consume the pool.
	maxUnauthenticatedQUICConnections       = 128
	maxUnauthenticatedQUICConnectionsPerPeer = 8
)

type quicPreAuthLimiter struct {
	mu       sync.Mutex
	maxTotal int
	maxPeer  int
	total    int
	byPeer   map[string]int
}

func newQUICPreAuthLimiter(maxTotal, maxPeer int) *quicPreAuthLimiter {
	return &quicPreAuthLimiter{
		maxTotal: maxTotal,
		maxPeer:  maxPeer,
		byPeer:   make(map[string]int),
	}
}

func (l *quicPreAuthLimiter) acquire(remote net.Addr) (*quicPreAuthLease, bool) {
	peer := quicPeerKey(remote)
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.total >= l.maxTotal || l.byPeer[peer] >= l.maxPeer {
		return nil, false
	}
	l.total++
	l.byPeer[peer]++
	return &quicPreAuthLease{limiter: l, peer: peer}, true
}

func (l *quicPreAuthLimiter) release(peer string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.total > 0 {
		l.total--
	}
	if count := l.byPeer[peer]; count > 1 {
		l.byPeer[peer] = count - 1
	} else {
		delete(l.byPeer, peer)
	}
}

type quicPreAuthLease struct {
	limiter *quicPreAuthLimiter
	peer    string
	once    sync.Once
}

func (l *quicPreAuthLease) release() {
	if l == nil || l.limiter == nil {
		return
	}
	l.once.Do(func() { l.limiter.release(l.peer) })
}

// quicPeerKey groups IPv6 addresses by /64. IPv6 clients can rotate privacy
// addresses, but doing so should not let one routed subnet consume the whole
// pre-authentication pool. IPv4 is counted per address.
func quicPeerKey(remote net.Addr) string {
	if remote == nil {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(remote.String())
	if err != nil {
		host = remote.String()
	}
	if ip, err := netip.ParseAddr(strings.Trim(host, "[]")); err == nil {
		ip = ip.Unmap().WithZone("")
		if ip.Is6() {
			return netip.PrefixFrom(ip, 64).Masked().String()
		}
		return ip.String()
	}
	return strings.ToLower(host)
}
