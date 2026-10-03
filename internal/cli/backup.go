package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/backpack/backpack/internal/app"
	"github.com/backpack/backpack/internal/manage"
	"github.com/backpack/backpack/internal/webui"
)

func runBackup(args []string) Result {
	asJSON, args := takeJSONFlag(args)
	if len(args) == 1 && args[0] == "capabilities" {
		if asJSON {
			return ok("{\"restore_protocol\":1}\n")
		}
		return ok("Backup restore protocol: 1\n")
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
	if len(args) != 3 || args[0] != "restore" || args[2] != "--yes" {
		return fail(CodeUsage, "Use backup capabilities, backup check <file>, or backup restore <file> --yes [--json].\n")
	}
	if os.Geteuid() != 0 {
		return fail(CodeFailed, "Backup restore requires root.\n")
	}
	return restoreBackupFile(args[1], asJSON)
}

// Called only after the public command has checked root and --yes.
func restoreBackupFile(path string, asJSON bool) Result {
	f, err := os.Open(path)
	if err != nil {
		return fail(CodeFailed, "Cannot open backup: %v\n", err)
	}
	defer f.Close()
	r, err := manage.Restore(f)
	if err != nil {
		return fail(CodeFailed, "Restore failed: %v\n", err)
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
	if r.Failed > 0 || panelErr != nil {
		code = CodeUnhealthy
	}
	var out string
	if asJSON {
		b, _ := json.Marshal(struct {
			manage.RestoreResult
			Protocol   int    `json:"restore_protocol"`
			PanelError string `json:"panel_error,omitempty"`
		}{r, 1, errorText(panelErr)})
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
