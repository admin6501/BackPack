package cli

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/backpack/backpack/internal/app"
	"github.com/backpack/backpack/internal/manage"
	"github.com/backpack/backpack/internal/webui"
)

func runBackup(args []string) Result {
	asJSON, args := takeJSONFlag(args)
	if len(args) == 1 && args[0] == "capabilities" {
		if asJSON {
			return ok("{\"restore_protocol\":2}\n")
		}
		return ok("Backup restore protocol: 2\n")
	}
	if len(args) == 2 && args[0] == "check" {
		r, err := manage.TestRestore(args[1])
		if err != nil {
			return fail(CodeFailed, "Invalid backup: %v\n", err)
		}
		if asJSON {
			b, _ := json.Marshal(r)
			return ok(string(b) + "\n")
		}
		return ok(r.Summary())
	}
	serverIP := ""
	if len(args) == 5 && args[3] == "--server-ip" {
		serverIP = args[4]
		if ip := net.ParseIP(serverIP); ip == nil || ip.IsUnspecified() || ip.IsMulticast() {
			return fail(CodeUsage, "--server-ip needs a valid destination IP.\n")
		}
		args = args[:3]
	}
	if len(args) != 3 || args[0] != "restore" || args[2] != "--yes" {
		return fail(CodeUsage, "Use backup capabilities, backup check <file>, or backup restore <file> --yes [--server-ip <IP>] [--json].\n")
	}
	if os.Geteuid() != 0 {
		return fail(CodeFailed, "Backup restore requires root.\n")
	}
	return restoreBackupFile(args[1], asJSON, serverIP)
}

// Called only after the public command has checked root and --yes.
func restoreBackupFile(path string, asJSON bool, serverIP string) Result {
	f, err := os.Open(path)
	if err != nil {
		return fail(CodeFailed, "Cannot open backup: %v\n", err)
	}
	defer f.Close()
	r, err := manage.RestoreForServerIP(f, serverIP)
	if err != nil {
		return fail(CodeFailed, "Restore failed: %v\n", err)
	}
	// A fresh destination has never opened the menu, so it has no monitor
	// unit yet. Install/start it here so history, quotas and backups continue
	// automatically after recovery without a second login on that machine.
	if err := manage.EnsureMonitorService(); err != nil {
		r.ServicesFailed = append(r.ServicesFailed, app.MonitorService)
		r.Warnings = append(r.Warnings, "Monitor could not start: "+err.Error())
	}
	var panelErr error
	if r.WebUIConfig {
		_, panelErr = webui.EnsureRunning()
		if panelErr == nil {
			panelErr = manage.RestartService(app.WebUIService)
		}
	}
	return backupRestoreResult(r, panelErr, asJSON)
}

func backupRestoreResult(r manage.RestoreResult, panelErr error, asJSON bool) Result {
	code := CodeOK
	if r.Failed > 0 || len(r.ServicesFailed) > 0 || panelErr != nil {
		code = CodeUnhealthy
	}
	var out string
	if asJSON {
		b, _ := json.Marshal(struct {
			manage.RestoreResult
			Protocol   int    `json:"restore_protocol"`
			PanelError string `json:"panel_error,omitempty"`
		}{r, 2, errorText(panelErr)})
		out = string(b) + "\n"
	} else {
		out = fmt.Sprintf("Restored %d files; tunnels started: %d, failed: %d.\n", r.Files, r.Started, r.Failed)
		for _, w := range r.Warnings {
			out += "Warning: " + w + "\n"
		}
	}
	return Result{Out: out, Err: errorText(panelErr), Code: code}
}

func errorText(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
