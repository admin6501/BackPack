package manage

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/backpack/backpack/config"
	"github.com/backpack/backpack/internal/app"
	"github.com/backpack/backpack/internal/tui"
)

// tunnelSecret reads the current on-disk config; the menu's tunnel list can
// become stale after a config edit in another process.
func tunnelSecret(name string) (string, error) {
	if err := checkName(name); err != nil {
		return "", err
	}
	return secretFromConfig(app.ConfigPath(name), name)
}

func secretFromConfig(path, name string) (string, error) {
	var cfg config.Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return "", err
	}
	var secret string
	switch {
	case cfg.Server.BindAddr != "":
		secret = cfg.Server.Token
	case cfg.Client.RemoteAddr != "":
		secret = cfg.Client.Token
	case cfg.Direct.Role != "":
		secret = cfg.Direct.Token
	case cfg.L3.Mode != "":
		secret = cfg.L3.Token
	default:
		return "", fmt.Errorf("no tunnel configuration in %q", name)
	}
	if secret == "" {
		return "", fmt.Errorf("no secret configured for %q", name)
	}
	return secret, nil
}

func showTunnelSecret(name string) {
	// A deliberate, local reveal only. Never log or send the value to the panel.
	tui.Clear()
	tui.Title("Show secret — " + name)
	secret, err := tunnelSecret(name)
	if err != nil {
		tui.Error("Could not read tunnel secret: " + err.Error())
		tui.PressEnter()
		return
	}
	tui.Warn("Anyone who sees this secret can connect to this tunnel. Keep your terminal private.")
	fmt.Println()
	// Escape control characters so a config value cannot manipulate the terminal.
	fmt.Println(strings.NewReplacer("\r", `\r`, "\n", `\n`, "\x1b", `\x1b`).Replace(secret))
	fmt.Println()
	tui.PressEnter()
}
