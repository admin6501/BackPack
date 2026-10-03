package menu

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/backpack/backpack/internal/node"
	"github.com/backpack/backpack/internal/tui"
)

func restoreRemoteBackup() {
	path := chooseBackup()
	if path == "" {
		return
	}
	host := tui.Prompt("Destination server IP: ")
	if host == "" {
		return
	}
	if net.ParseIP(host) == nil {
		tui.Error("Enter a valid IPv4 or IPv6 address.")
		tui.PressEnter()
		return
	}
	port := tui.PromptInt("SSH port", 22)
	if port < 1 || port > 65535 {
		tui.Error("SSH port must be between 1 and 65535.")
		tui.PressEnter()
		return
	}
	user := tui.PromptDefault("SSH username", "root")
	password, err := askSSHPassword()
	if err != nil {
		tui.Error(err.Error())
		tui.PressEnter()
		return
	}
	tui.Info("Validating the backup and checking the destination (no settings changed yet)...")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()
	r, err := node.PrepareRemoteRestore(ctx, node.SSHTarget{Host: host, Port: port, User: user, Password: password}, path)
	password = ""
	if err != nil {
		tui.Error(err.Error())
		tui.PressEnter()
		return
	}
	defer r.Close()
	fmt.Println()
	tui.Info(fmt.Sprintf("Destination: %s@%s (SSH port %d)", user, host, port))
	tui.Info("SSH host fingerprint: " + r.Fingerprint)
	tui.Info(r.Report.Summary())
	if r.NeedsInstall {
		tui.Warn("Backpack is missing or too old for automatic restore; it will be installed/updated.")
	}
	tui.Warn("This overwrites matching tunnels/settings on the DESTINATION and restarts its tunnels.")
	tui.Warn("Peer addresses and ports are kept as saved; check them when moving to a new IP.")
	if r.Report.FleetSealed {
		tui.Warn("Managed-server passwords need the separate fleet key. It is not transferred here.")
	}
	if !tui.Confirm("Restore this backup on the destination now", false) {
		return
	}
	res, err := r.Restore(ctx, func(message string) { tui.Info(message) })
	if err != nil {
		tui.Error(err.Error())
	} else {
		tui.Success("Backup restored on the destination server.")
	}
	if res.Files > 0 {
		tui.Info(fmt.Sprintf("%d files restored; %d tunnels started, %d failed.", res.Files, res.Started, res.Failed))
	}
	for _, warning := range res.Warnings {
		tui.Warn(warning)
	}
	tui.PressEnter()
}
