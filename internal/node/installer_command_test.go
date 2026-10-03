package node

import (
	"crypto/sha256"
	"errors"
	"fmt"
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

func TestPinnedInstallerRejectsChangedScriptBeforeExecution(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "executed")
	script := "#!/bin/sh\ntouch " + quote(marker) + "\n"
	curl := filepath.Join(dir, "curl")
	if err := os.WriteFile(curl, []byte("#!/bin/sh\nprintf '%s' "+quote(script)+" > \"$4\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, valid := range []bool{false, true} {
		digest := strings.Repeat("0", 64)
		if valid {
			digest = fmt.Sprintf("%x", sha256.Sum256([]byte(script)))
		}
		cmd := exec.Command("sh", "-c", installerCommand("https://example.test/pinned/install.sh", digest))
		cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"))
		out, err := cmd.CombinedOutput()
		if !valid {
			if err == nil || !strings.Contains(string(out), "refusing to execute") {
				t.Fatalf("tampered installer accepted: %v %s", err, out)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatal("tampered installer executed")
			}
		} else if err != nil {
			t.Fatalf("verified installer rejected: %v %s", err, out)
		}
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("verified installer did not execute")
	}
	current, err := os.ReadFile("../../install.sh")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(current)); got != restoreInstallerSHA256 {
		t.Fatal("installer changed: audit it and update the pinned commit/digest together")
	}
}
