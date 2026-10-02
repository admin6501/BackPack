package telegram

import (
	"crypto/tls"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/backpack/backpack/internal/manage"
)

// Choosing which tunnel carries the bot's traffic.
//
// The Iran server cannot reach Telegram directly, so every request goes through
// a tunnel to a peer abroad and out from there. Which tunnel was a question the
// user had to answer at setup — and then answer again, by hand, whenever that
// particular tunnel went down. Meanwhile the bot stayed silent, which is
// precisely when it is needed: a tunnel dropping is the thing worth being told
// about.
//
// So the choice is made automatically and re-made whenever the current one
// stops being usable. Any healthy tunnel will do; the only requirement is that
// it can reach the peer.

// AutoRelay is the sentinel stored in ViaTunnel to mean "pick one and keep
// picking". A specific tunnel name still pins it, so anyone who wants manual
// control keeps it.
const AutoRelay = "*auto*"

// relayState remembers the tunnel currently carrying the bot, so a working
// choice is not re-evaluated on every single request.
type relayState struct {
	mu       sync.Mutex
	name     string
	port     int
	direct   bool
	chosenAt time.Time
}

var relay relayState

// relayRecheck is how long a working choice is trusted before its health is
// looked at again. Short enough to notice a drop quickly; long enough that the
// bot is not re-deciding on every message.
const relayRecheck = 30 * time.Second

// resolveRelay returns the tunnel and the local port the bot should use.
//
// With a pinned tunnel it returns that one and nothing else is considered — a
// deliberate choice deserves to be honoured even when it is down, because
// silently using a different route would be worse than an error the user can
// see.
func resolveRelay(c Config) (name string, port int, err error) {
	if c.ViaTunnel != AutoRelay {
		if c.ViaTunnel == "" {
			return "", 0, nil // direct; no relay wanted
		}
		// The stored port is not trusted. A config written before the bot
		// forwarded straight to the API holds the old proxy port, and dialling
		// it now gets a SOCKS greeting where a TLS handshake was expected —
		// which surfaces as "first record does not look like a TLS handshake"
		// and no hint that the mapping is simply the wrong kind.
		port, err := manage.EnsureTelegramPort(c.ViaTunnel)
		if err != nil {
			return c.ViaTunnel, port, err
		}
		return c.ViaTunnel, port, nil
	}

	relay.mu.Lock()
	defer relay.mu.Unlock()

	// Keep the current choice while it is fresh and still healthy.
	if relay.name != "" && time.Since(relay.chosenAt) < relayRecheck {
		return relay.name, relay.port, nil
	}
	if relay.direct && time.Since(relay.chosenAt) < relayRecheck {
		return "", 0, nil
	}
	if relay.name != "" && relayUsable(relay.name) {
		relay.chosenAt = time.Now()
		return relay.name, relay.port, nil
	}

	picked, pickedPort, err := chooseAutomaticRelay(pickRelay, telegramDirectReachable)
	if err != nil {
		// Leave the previous choice in place: a momentary failure to find a
		// better one is not a reason to stop using a tunnel that may still work.
		if relay.name != "" {
			return relay.name, relay.port, nil
		}
		return "", 0, err
	}

	relay.name, relay.port, relay.direct, relay.chosenAt = picked, pickedPort, picked == "", time.Now()
	return picked, pickedPort, nil
}

// chooseAutomaticRelay uses a tunnel when one can be prepared, and falls back
// to a direct Telegram connection when this host can reach Telegram itself.
// This matters on the kharej side: its working reverse tunnel is a client, and
// only the Iran-side server can expose the loopback relay port. Treating that
// client as the absence of all connectivity made the bot fail on a host whose
// direct internet already works.
func chooseAutomaticRelay(pick func() (string, int, error), direct func() bool) (string, int, error) {
	name, port, err := pick()
	if err == nil {
		return name, port, nil
	}
	if direct() {
		return "", 0, nil
	}
	return "", 0, err
}

// telegramDirectReachable proves that the configured API host is reachable
// with a valid TLS certificate before automatic mode falls back to direct.
// A TCP-only test would accept a port-forward greeting or an interception page
// and then send the real bot into a failure it could not explain.
func telegramDirectReachable() bool {
	dialer := &net.Dialer{Timeout: 4 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", manage.TelegramHost, &tls.Config{
		ServerName: "api.telegram.org",
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// pickRelay chooses the best tunnel to send through and makes sure it forwards
// a port to the Telegram API.
func pickRelay() (string, int, error) {
	candidates := relayCandidates()
	if len(candidates) == 0 {
		return "", 0, fmt.Errorf("no connected tunnel is available to relay through")
	}

	// Try them best-first: exposing the relay port restarts the tunnel, so a
	// candidate that turns out to be unusable must not stop the next one.
	var lastErr error
	for _, name := range candidates {
		port, err := manage.EnsureTelegramPort(name)
		if err != nil {
			lastErr = err
			continue
		}
		return name, port, nil
	}
	return "", 0, fmt.Errorf("no tunnel could be prepared for relaying: %w", lastErr)
}

// relayCandidates lists the tunnels that could carry the bot, best first.
//
// Only server tunnels qualify: the relay works by exposing a port that maps to
// a port on the peer's side, and only the server side exposes ports. A tunnel that
// already has the relay port is preferred, because choosing it costs nothing —
// every other candidate has to be restarted to gain the mapping, which briefly
// interrupts whatever it is carrying.
func relayCandidates() []string {
	type candidate struct {
		name    string
		ready   bool // already has the relay port
		healthy bool
	}

	var all []candidate
	for name, h := range manage.AllHealth() {
		t, ok := manage.Find(name)
		if !ok || t.Role != "server" {
			continue
		}
		if h.State != "online" {
			continue
		}
		all = append(all, candidate{
			name:    name,
			ready:   manage.HasTelegramPort(name),
			healthy: true,
		})
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].ready != all[j].ready {
			return all[i].ready // an already-prepared tunnel first
		}
		return all[i].name < all[j].name // stable, so the choice does not churn
	})

	out := make([]string, 0, len(all))
	for _, c := range all {
		out = append(out, c.name)
	}
	return out
}

// relayUsable reports whether a tunnel is still fit to carry the bot.
func relayUsable(name string) bool {
	h, ok := manage.AllHealth()[name]
	return ok && h.State == "online"
}

// RelayStatus describes the current relay choice, for the CLI and the panel.
func RelayStatus() string {
	c := Load()
	switch c.ViaTunnel {
	case "":
		return "direct (no relay)"
	case AutoRelay:
		name, _, err := resolveRelay(c)
		if err != nil {
			return "automatic — no connected tunnel available right now"
		}
		if name == "" {
			return "automatic — direct (Telegram is reachable without a tunnel)"
		}
		return "automatic — currently using " + name
	default:
		return "pinned to " + c.ViaTunnel
	}
}

// PrepareAutoRelay picks a tunnel now and makes sure it exposes the relay port,
// so setup can report something concrete instead of "it will sort itself out".
func PrepareAutoRelay() (name string, port int, err error) {
	return pickRelay()
}

// explainSendFailure turns a transport-level error into something a person can
// act on.
//
// The bare errors here name the wrong thing. A request that never leaves this
// machine reports `dial tcp 127.0.0.1:31138: connection refused`, which reads
// like a local bug rather than "the tunnel is not exposing the port it was told
// to". Each case below says which machine to look at.
//
// The error is classified before anything else is consulted. Working out which
// relay is in use can itself fail — a missing config, a tunnel that was deleted
// — and when it did, the diagnosis used to be replaced by that secondary
// failure, throwing away the more useful information.
func explainSendFailure(c Config, err error) error {
	if err == nil {
		return nil
	}
	if c.ViaTunnel == "" {
		return fmt.Errorf("%w\n\nSending directly, so this server needs its own access to "+
			"api.telegram.org. From Iran it will not have it — set the relay to Automatic", err)
	}

	// Best-effort context; the advice below does not depend on it.
	name, port := c.ViaTunnel, c.SocksPort
	if n, p, rerr := resolveRelay(c); rerr == nil {
		if c.ViaTunnel == AutoRelay && n == "" {
			return fmt.Errorf("%w\n\nAutomatic mode found no relay tunnel and tried Telegram directly. "+
				"If direct Telegram access is blocked here, set the bot relay on the Iran server "+
				"to Automatic and keep a server-side tunnel to kharej online", err)
		}
		if n != "" {
			name, port = n, p
		}
	}
	if name == AutoRelay {
		name = "the automatically chosen tunnel"
	}

	msg := err.Error()
	switch {
	case strings.Contains(msg, "connection refused"):
		return fmt.Errorf("%w\n\nNothing is listening on 127.0.0.1:%d on THIS server.\n"+
			"Tunnel %q is supposed to be forwarding that port to Telegram but is not.\n\n"+
			"Run:  sudo backpack → Telegram Bot → Diagnose relay\n"+
			"or check:  journalctl -u backpack-%s -n 30", err, port, name, name)

	case strings.Contains(msg, "does not look like a TLS handshake"):
		return fmt.Errorf("%w\n\nSomething answered, but not Telegram. Tunnel %q is still forwarding\n"+
			"its relay port to the old proxy instead of straight to the API.\n\n"+
			"Fix it with:  sudo backpack → Telegram Bot → Configure", err, name)

	case strings.Contains(msg, "EOF"), strings.Contains(msg, "reset by peer"):
		return fmt.Errorf("%w\n\nThe tunnel carried the connection but the far end could not\n"+
			"reach api.telegram.org. The OTHER server is what dials out — check its\n"+
			"internet access and that its side of %q is connected.\n\n"+
			"Run:  sudo backpack → Telegram Bot → Diagnose relay", err, name)

	case strings.Contains(msg, "certificate"), strings.Contains(msg, "x509"):
		return fmt.Errorf("%w\n\nThe TLS handshake with Telegram failed through tunnel %q.\n"+
			"Something is intercepting the connection on the far side", err, name)

	default:
		return fmt.Errorf("%w\n\n(relaying through %q on local port %d)\n"+
			"Run:  sudo backpack → Telegram Bot → Diagnose relay", err, name, port)
	}
}
