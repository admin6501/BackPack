package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSystemctlUsesLinuxExecutableName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "systemctl"), []byte("#!/bin/sh\nprintf active"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	out, err := Systemctl("is-active", "test.service")
	if err != nil || out != "active" {
		t.Fatalf("out=%q err=%v", out, err)
	}
}
