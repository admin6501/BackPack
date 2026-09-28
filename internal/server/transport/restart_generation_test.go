package transport

import (
	"context"
	"testing"
)

// A delayed failure from a retired run must not tear down its replacement.
func TestRetiredTCPRestartCannotStopCurrentRun(t *testing.T) {
	for _, tc := range []struct {
		name    string
		restart func(context.Context, context.Context) context.Context
	}{
		{"tcp", func(old, current context.Context) context.Context {
			s := &TcpTransport{}
			s.run.set(current, func() {})
			s.restart(old)
			return s.run.context()
		}},
		{"tcpmux", func(old, current context.Context) context.Context {
			s := &TcpMuxTransport{}
			s.run.set(current, func() {})
			s.restart(old)
			return s.run.context()
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old, cancel := context.WithCancel(context.Background())
			defer cancel()
			current := context.Background()
			if got := tc.restart(old, current); got != current {
				t.Fatal("retired run replaced the current generation")
			}
		})
	}
}
