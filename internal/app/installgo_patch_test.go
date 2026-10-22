package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Source installs must not keep a compiler whose security fixes are older than
// this module's minimum, including when only the patch version differs.
func TestSourceInstallerRejectsUnpatchedToolchains(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not installed")
	}
	sh, err := os.ReadFile("../../install.sh")
	if err != nil {
		t.Fatal(err)
	}
	mod, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	want := goDirective(string(mod))
	if fallback := shellValue(string(sh), "GO_VERSION"); fallback != want {
		t.Fatalf("standalone installer compiler %s differs from module minimum %s", fallback, want)
	}
	start := strings.Index(string(sh), "go_new_enough() {")
	if start < 0 {
		t.Fatal("installer has no compiler version check")
	}
	helper := string(sh[start:])
	end := strings.Index(helper, "\n}")
	if end < 0 {
		t.Fatal("installer compiler version check is incomplete")
	}
	helper = helper[:end+2]
	for _, tc := range []struct {
		version string
		accept  bool
	}{
		{"1.26.6", false}, {"1.26.9", false},
		{"1.26.10", true}, {"1.26.11", true},
		{"1.27.0", false}, {"1.27.1", false}, {"1.27.2", false}, {"1.27.3", true},
	} {
		t.Run(tc.version, func(t *testing.T) {
			fake := filepath.Join(t.TempDir(), "go")
			if err := os.WriteFile(fake, []byte("#!/bin/sh\nprintf 'go version go"+tc.version+" linux/amd64\\n'\n"), 0700); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(bash, "-c", "GO_VERSION=\"$1\"\n"+helper+"\ngo_new_enough \"$2\"", "installer-test", want, fake)
			out, err := cmd.CombinedOutput()
			if accepted := err == nil; accepted != tc.accept {
				t.Fatalf("compiler %s accepted=%v, want %v: %s", tc.version, accepted, tc.accept, out)
			}
		})
	}
}
