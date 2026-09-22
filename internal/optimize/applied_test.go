package optimize

import (
	"os"
	"strings"
	"path/filepath"
	"testing"
)

// WasApplied is what lets an update repair an older Optimize's settings without
// tuning a machine that never asked for it. Getting it backwards either way is
// bad: false on a tuned machine leaves the wide port range in place forever,
// and true on an untouched one retunes a kernel nobody consented to.
func TestWasAppliedFollowsTheFileOptimizeOwns(t *testing.T) {
	dir := t.TempDir()
	old := sysctlFile
	t.Cleanup(func() { sysctlFile = old })

	sysctlFile = filepath.Join(dir, "99-backpack.conf")
	if WasApplied() {
		t.Error("reported as applied with no file present — an update would retune a " +
			"machine whose operator never ran Optimize")
	}

	if err := os.WriteFile(sysctlFile, []byte("# managed by backpack\n"), 0o644); err != nil {
		t.Fatalf("writing the stand-in sysctl file: %v", err)
	}
	if !WasApplied() {
		t.Error("reported as not applied with the file present — a server carrying an " +
			"older Optimize's port range would never be repaired")
	}
}


func TestBootServiceReappliesPersistentSettings(t *testing.T) {
	unit := BootServiceContent()
	for _, want := range []string{
		"After=local-fs.target systemd-sysctl.service",
		"Before=network-pre.target",
		"ExecStart=/sbin/sysctl -p /etc/sysctl.d/99-backpack.conf",
		"WantedBy=multi-user.target",
	} {
		if !strings.Contains(unit, want) {
			t.Errorf("boot service is missing %q", want)
		}
	}
}
