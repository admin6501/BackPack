package telegram

import (
	"errors"
	"testing"
)

func TestAutomaticRelayUsesDirectTelegramWhenThereIsNoRelayTunnel(t *testing.T) {
	noTunnel := errors.New("no connected tunnel is available to relay through")
	calledDirect := false

	name, port, err := chooseAutomaticRelay(
		func() (string, int, error) { return "", 0, noTunnel },
		func() bool { calledDirect = true; return true },
	)
	if err != nil {
		t.Fatalf("automatic mode rejected direct Telegram access: %v", err)
	}
	if !calledDirect {
		t.Fatal("automatic mode never checked the direct route")
	}
	if name != "" || port != 0 {
		t.Fatalf("direct route resolved as tunnel %q on port %d", name, port)
	}
}

func TestAutomaticRelayKeepsTheTunnelErrorWhenDirectTelegramIsBlocked(t *testing.T) {
	noTunnel := errors.New("no connected tunnel is available to relay through")
	name, port, err := chooseAutomaticRelay(
		func() (string, int, error) { return "", 0, noTunnel },
		func() bool { return false },
	)
	if !errors.Is(err, noTunnel) {
		t.Fatalf("automatic mode error = %v, want the relay error", err)
	}
	if name != "" || port != 0 {
		t.Fatalf("missing relay returned tunnel %q on port %d", name, port)
	}
}

func TestAutomaticRelayPrefersAPreparedTunnel(t *testing.T) {
	calledDirect := false
	name, port, err := chooseAutomaticRelay(
		func() (string, int, error) { return "server-2433", 31234, nil },
		func() bool { calledDirect = true; return true },
	)
	if err != nil || name != "server-2433" || port != 31234 {
		t.Fatalf("automatic choice = %q:%d, %v", name, port, err)
	}
	if calledDirect {
		t.Fatal("automatic mode probed direct access despite having a usable tunnel")
	}
}
