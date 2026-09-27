package manage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTunnelSecretReadsAllTunnelKinds(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct{ name, config, want string }{
		{"reverse-server", "[server]\nbind_addr = '0.0.0.0:9000'\ntoken = 'server-secret'\n", "server-secret"},
		{"reverse-client", "[client]\nremote_addr = 'example.com:9000'\ntoken = 'client-secret'\n", "client-secret"},
		{"direct", "[direct]\nrole = 'iran'\ntoken = 'direct-secret'\n", "direct-secret"},
		{"l3", "[l3]\nmode = 'dial'\ntoken = 'l3-secret'\n", "l3-secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".toml")
			if err := os.WriteFile(path, []byte(tc.config), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := secretFromConfig(path, tc.name)
			if err != nil || got != tc.want {
				t.Fatalf("secret = %q, err = %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestTunnelSecretRejectsMissingAndUnsafeNames(t *testing.T) {
	for _, name := range []string{"../other", "bad/name"} {
		if secret, err := tunnelSecret(name); err == nil || secret != "" {
			t.Errorf("name %q: secret = %q, err = %v", name, secret, err)
		}
	}
	if secret, err := secretFromConfig(filepath.Join(t.TempDir(), "missing.toml"), "missing"); err == nil || secret != "" {
		t.Errorf("missing config: secret = %q, err = %v", secret, err)
	}
}
