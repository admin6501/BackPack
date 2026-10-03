package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
