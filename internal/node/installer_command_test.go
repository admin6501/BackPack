package node

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallerCommandExecutesDownloadedScriptAndReportsDownloadFailure(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "installed")
	curl := filepath.Join(dir, "curl")
	// Stand in for the remote download: the installer must execute from its
	// downloaded file even though its own stdin is /dev/null.
	if err := os.WriteFile(curl, []byte("#!/bin/sh\nprintf '#!/bin/bash\\ntouch \"%s\"\\n' \"$MARKER\" > \"$4\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", "-c", installerCommand("https://example.test/install.sh"))
	cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "MARKER="+marker)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("install command: %v: %s", err, out)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("the downloaded installer did not run: %v", err)
	}

	if err := os.WriteFile(curl, []byte("#!/bin/sh\nexit 22\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("sh", "-c", installerCommand("https://example.test/install.sh"))
	cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"))
	if out, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(out), "could not download Backpack installer (curl exit 22)") {
		t.Fatalf("download failure must identify its stage and exit code: %v: %s", err, out)
	}
	if msg := installFailureMessage("", errors.New("exit status 17")); !strings.Contains(msg, "exit status 17") {
		t.Fatalf("silent remote failures must retain their exit status: %q", msg)
	}
	if err := os.WriteFile(curl, []byte("#!/bin/sh\nprintf '#!/bin/bash\\nexit 31\\n' > \"$4\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("sh", "-c", installerCommand("https://example.test/install.sh"))
	cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"))
	if out, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(out), "Backpack installer exited with status 31") {
		t.Fatalf("installer failure must identify its stage and exit code: %v: %s", err, out)
	}
}
