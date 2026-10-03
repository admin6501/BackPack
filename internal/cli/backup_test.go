package cli

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backpack/backpack/internal/manage"
)

func TestBackupRestoreRequiresExplicitConfirmation(t *testing.T) {
	for _, args := range [][]string{{"backup"}, {"backup", "restore", "anything"}, {"backup", "restore", "anything", "--no"}, {"backup", "check"}, {"backup", "capabilities", "extra"}} {
		if r := Run(args); r.Code != CodeUsage {
			t.Fatalf("Run(%v) = %+v", args, r)
		}
	}
	if !IsCommand("backup") {
		t.Fatal("backup command is not routed")
	}
}

func TestBackupRestoreCommandReadsArchiveAndReportsFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := os.WriteFile(path, []byte("the selected archive"), 0600); err != nil {
		t.Fatal(err)
	}
	old := manage.Restore
	t.Cleanup(func() { manage.Restore = old })
	manage.Restore = func(r io.Reader) (manage.RestoreResult, error) {
		b, err := io.ReadAll(r)
		if err != nil || string(b) != "the selected archive" {
			t.Error("wrong archive passed to restore")
		}
		return manage.RestoreResult{Files: 3, Started: 1, Warnings: []string{"fleet key needed"}}, nil
	}
	for _, asJSON := range []bool{false, true} {
		r := restoreBackupFile(path, asJSON)
		if r.Code != CodeOK {
			t.Fatalf("restore: %+v", r)
		}
		if asJSON {
			var result struct {
				manage.RestoreResult
				Protocol int `json:"restore_protocol"`
			}
			if json.Unmarshal([]byte(r.Out), &result) != nil || result.Protocol != 1 || result.Files != 3 {
				t.Fatalf("result: %+v", r)
			}
		} else if !strings.Contains(r.Out, "fleet key needed") {
			t.Fatal("restore warning lost")
		}
	}
	manage.Restore = func(io.Reader) (manage.RestoreResult, error) {
		return manage.RestoreResult{}, errors.New("bad archive")
	}
	if r := restoreBackupFile(path, true); r.Code != CodeFailed || !strings.Contains(r.Err, "bad archive") {
		t.Fatalf("restore failure: %+v", r)
	}
	if r := restoreBackupFile(path+".missing", false); r.Code != CodeFailed {
		t.Fatal("missing archive accepted")
	}
}

func TestBackupRestoreResultsReportPartialServices(t *testing.T) {
	for _, asJSON := range []bool{false, true} {
		if r := backupRestoreResult(manage.RestoreResult{Files: 2, Failed: 1}, nil, asJSON); r.Code != CodeUnhealthy {
			t.Fatalf("failed tunnel reported as success: %+v", r)
		}
		if r := backupRestoreResult(manage.RestoreResult{Files: 2}, errors.New("panel failed"), asJSON); r.Code != CodeUnhealthy || r.Err != "panel failed" {
			t.Fatalf("failed panel: %+v", r)
		}
	}
}

func TestBackupCapabilitiesAndValidationNeverRestore(t *testing.T) {
	r := Run([]string{"backup", "capabilities", "--json"})
	var c struct {
		Protocol int `json:"restore_protocol"`
	}
	if r.Code != 0 || json.Unmarshal([]byte(r.Out), &c) != nil || c.Protocol != 1 {
		t.Fatalf("capabilities: %+v", r)
	}
	path := filepath.Join(t.TempDir(), "bad.tar.gz")
	if err := os.WriteFile(path, []byte("invalid archive"), 0600); err != nil {
		t.Fatal(err)
	}
	if r := Run([]string{"backup", "check", path, "--json"}); r.Code != CodeFailed {
		t.Fatalf("invalid backup accepted: %+v", r)
	}
}
