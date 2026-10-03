package node

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func remoteBackupFixture(t *testing.T) (string, []byte) {
	t.Helper()
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	body := []byte("[server]\nbind_addr = \"0.0.0.0:443\"\n")
	if err := tw.WriteHeader(&tar.Header{Name: "edge.toml", Mode: 0600, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := os.WriteFile(path, b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return path, b.Bytes()
}

func restoreTarget(s *fakeServer) SSHTarget {
	h, p := s.addr()
	return SSHTarget{Host: h, Port: p, User: s.user, Password: s.pass}
}

func TestRemoteRestoreStreamsPrivateSnapshotAndSkipsInstalledBinary(t *testing.T) {
	for _, needsInstall := range []bool{false, true} {
		t.Run(fmt.Sprint(needsInstall), func(t *testing.T) {
			path, archive := remoteBackupFixture(t)
			dir := t.TempDir()
			binary := filepath.Join(dir, "backpack")
			captured := filepath.Join(dir, "received.tar.gz")
			// The actual remote shell runs; only Backpack/systemd are replaced.
			body := "#!/bin/sh\ncp \"$3\" " + quote(captured) + "\nstat -c '%a' \"$3\" > " + quote(filepath.Join(dir, "mode")) + "\nprintf '%s\\n' '{\"restore_protocol\":1,\"Files\":1,\"Started\":1,\"Failed\":0}'\n"
			if err := os.WriteFile(binary, []byte(body), 0700); err != nil {
				t.Fatal(err)
			}
			s := newFakeServer(t, "root", "a password ' with spaces")
			var installed atomic.Bool
			installed.Store(!needsInstall)
			var installs atomic.Int32
			s.mu.Lock()
			s.execHandler = func(cmd string, ch ssh.Channel) uint32 {
				switch {
				case strings.Contains(cmd, "uname -s"):
					fmt.Fprintln(ch, "root")
					return 0
				case strings.Contains(cmd, "backup capabilities"):
					if !installed.Load() {
						fmt.Fprintln(ch.Stderr(), "old binary")
						return 2
					}
					fmt.Fprintln(ch, "SSH banner\n{\"restore_protocol\":1}")
					return 0
				case strings.Contains(cmd, "install_script=$(mktemp)"):
					installs.Add(1)
					installed.Store(true)
					return 0
				default:
					cmd = strings.ReplaceAll(cmd, "/usr/local/bin/backpack", binary)
					cmd = strings.ReplaceAll(cmd, "/tmp/backpack-restore.", dir+"/backpack-restore.")
					p := exec.Command("sh", "-c", cmd)
					p.Stdin, p.Stdout, p.Stderr = ch, ch, ch.Stderr()
					if err := p.Run(); err != nil {
						return 1
					}
					return 0
				}
			}
			s.mu.Unlock()
			r, err := PrepareRemoteRestore(context.Background(), restoreTarget(s), path)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			if r.NeedsInstall != needsInstall || r.Fingerprint == "" || len(r.Report.Tunnels) != 1 {
				t.Fatalf("preflight: %+v", r)
			}
			snapshot := r.file.Name()
			// Replacing the input after confirmation cannot change the upload.
			if err := os.WriteFile(path, []byte("changed later"), 0600); err != nil {
				t.Fatal(err)
			}
			result, err := r.Restore(context.Background(), nil)
			if err != nil || result.Started != 1 {
				t.Fatalf("restore: %+v, %v", result, err)
			}
			got, err := os.ReadFile(captured)
			if err != nil || !bytes.Equal(got, archive) {
				t.Fatalf("upload differs: %v", err)
			}
			mode, _ := os.ReadFile(filepath.Join(dir, "mode"))
			if strings.TrimSpace(string(mode)) != "600" {
				t.Fatalf("private upload mode: %s", mode)
			}
			left, _ := filepath.Glob(dir + "/backpack-restore.*")
			if len(left) != 0 {
				t.Fatalf("remote archive leaked: %v", left)
			}
			wantInstalls := int32(0)
			if needsInstall {
				wantInstalls = 1
			}
			if installs.Load() != wantInstalls {
				t.Fatal("unexpected installation")
			}
			r.Close()
			if _, err := os.Stat(snapshot); !os.IsNotExist(err) {
				t.Fatal("local snapshot leaked")
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			for _, cmd := range s.ran {
				if strings.Contains(cmd, s.pass) {
					t.Fatal("password leaked into a remote command")
				}
			}
		})
	}
}

func TestRemoteRestoreRejectsCorruptionBeforeRunningRestoreAndCleansUpload(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "restored")
	binary := filepath.Join(dir, "backpack")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\ntouch "+quote(marker)+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte("expected archive")))
	script := strings.ReplaceAll(remoteRestoreScript(checksum), "/usr/local/bin/backpack", binary)
	script = strings.ReplaceAll(script, "/tmp/backpack-restore.", dir+"/backpack-restore.")
	cmd := exec.Command("sh", "-c", script)
	cmd.Stdin = strings.NewReader("corrupted archive")
	if out, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(out), "FAILED") {
		t.Fatalf("corruption accepted: %v: %s", err, out)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("restore ran before checksum verification")
	}
	left, _ := filepath.Glob(dir + "/backpack-restore.*")
	if len(left) != 0 {
		t.Fatalf("upload leaked: %v", left)
	}
}

func TestRemoteRestoreInvalidArchiveAndAuthenticationFailBeforeMutation(t *testing.T) {
	path, _ := remoteBackupFixture(t)
	s := newFakeServer(t, "root", "correct")
	target := restoreTarget(s)
	target.Password = "wrong"
	if r, err := PrepareRemoteRestore(context.Background(), target, path); err == nil {
		r.Close()
		t.Fatal("wrong password accepted")
	}
	target = restoreTarget(s)
	target.Fingerprint = "SHA256:not-the-destination-key"
	if r, err := PrepareRemoteRestore(context.Background(), target, path); err == nil {
		r.Close()
		t.Fatal("changed host key accepted")
	}
	if err := os.WriteFile(path, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if r, err := PrepareRemoteRestore(context.Background(), restoreTarget(s), path); err == nil {
		r.Close()
		t.Fatal("invalid backup accepted")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.ran) != 0 {
		t.Fatal("commands ran before archive/authentication checks")
	}
}

func TestRemoteRestoreSudoAndInstallFailureDoNotUploadBackup(t *testing.T) {
	path, _ := remoteBackupFixture(t)
	s := newFakeServer(t, "operator", "secret")
	var installCalls, uploads atomic.Int32
	s.mu.Lock()
	s.execHandler = func(cmd string, ch ssh.Channel) uint32 {
		if strings.Contains(cmd, "uname -s") {
			fmt.Fprintln(ch, "sudo")
			return 0
		}
		if !strings.HasPrefix(cmd, "sudo -n sh -c ") {
			fmt.Fprintln(ch.Stderr(), "not elevated")
			return 1
		}
		if strings.Contains(cmd, "backup capabilities") {
			return 127
		}
		if strings.Contains(cmd, "install_script=$(mktemp)") {
			installCalls.Add(1)
			fmt.Fprintln(ch.Stderr(), "installer failed")
			return 22
		}
		uploads.Add(1)
		return 0
	}
	s.mu.Unlock()
	r, err := PrepareRemoteRestore(context.Background(), restoreTarget(s), path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if installCalls.Load() != 0 || uploads.Load() != 0 {
		t.Fatal("preflight mutated destination before confirmation")
	}
	if _, err := r.Restore(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "installer failed") {
		t.Fatalf("install failure: %v", err)
	}
	if installCalls.Load() != 1 || uploads.Load() != 0 {
		t.Fatal("archive uploaded after installer failure")
	}
}

func TestRemoteRestoreReportsPartialFailureAndCancellation(t *testing.T) {
	for _, scenario := range []string{"failed-tunnel", "failed-panel", "cancel", "invalid-response"} {
		t.Run(scenario, func(t *testing.T) {
			path, _ := remoteBackupFixture(t)
			s := newFakeServer(t, "root", "secret")
			s.mu.Lock()
			s.execHandler = func(cmd string, ch ssh.Channel) uint32 {
				if strings.Contains(cmd, "uname -s") {
					fmt.Fprintln(ch, "root")
					return 0
				}
				if strings.Contains(cmd, "backup capabilities") {
					fmt.Fprintln(ch, "{\"restore_protocol\":1}")
					return 0
				}
				io.Copy(io.Discard, ch)
				switch scenario {
				case "cancel":
					time.Sleep(100 * time.Millisecond)
					return 0
				case "failed-tunnel":
					fmt.Fprintln(ch, "{\"restore_protocol\":1,\"Files\":1,\"Failed\":2}")
					return 4
				case "failed-panel":
					fmt.Fprintln(ch, "{\"restore_protocol\":1,\"Files\":1,\"panel_error\":\"port in use\"}")
					return 4
				default:
					fmt.Fprintln(ch, "{}")
					return 0
				}
			}
			s.mu.Unlock()
			r, err := PrepareRemoteRestore(context.Background(), restoreTarget(s), path)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			if scenario != "cancel" {
				ctx = context.Background()
			}
			if _, err := r.Restore(ctx, nil); err == nil {
				t.Fatal("failed/cancelled restore reported as success")
			}
		})
	}
}
