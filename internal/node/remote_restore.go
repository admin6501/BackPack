package node

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/backpack/backpack/internal/app"
	"github.com/backpack/backpack/internal/manage"
	"golang.org/x/crypto/ssh"
)

// RemoteRestore holds an ephemeral connection and a validated, private copy of
// the archive. Credentials are never enrolled in the fleet or written to disk.
// The same SSH connection is used before and after the operator's confirmation.
type RemoteRestore struct {
	client       *ssh.Client
	file         *os.File
	checksum     string
	sudo         bool
	Fingerprint  string
	NeedsInstall bool
	Report       manage.RestoreReport
}

const maxTransferBackup = 512 << 20

// The installer itself is trusted code: never execute mutable branch contents
// as root. This audited commit and digest are updated deliberately together.
const restoreInstallerCommit = "afe905b9f0d2394507e45232cb4362612c070e98"
const restoreInstallerSHA256 = "1c135ed17ed714bca443c9454ecb50e131d5aff4e26e675f1dae25e88205c089"

func PrepareRemoteRestore(ctx context.Context, target SSHTarget, path string) (_ *RemoteRestore, err error) {
	if net.ParseIP(target.Host) == nil || target.Port < 0 || target.Port > 65535 || target.User == "" {
		return nil, fmt.Errorf("a destination host, valid SSH port and username are required")
	}
	r := &RemoteRestore{}
	defer func() {
		if err != nil {
			r.Close()
		}
	}()
	src, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxTransferBackup {
		return nil, fmt.Errorf("backup must be a regular file no larger than 512 MiB")
	}
	r.file, err = os.CreateTemp("", "backpack-ssh-restore-*.tar.gz")
	if err != nil {
		return nil, err
	}
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(r.file, hash), io.LimitReader(src, maxTransferBackup+1))
	if err != nil {
		return nil, err
	}
	if n > maxTransferBackup {
		return nil, fmt.Errorf("backup exceeds 512 MiB")
	}
	r.checksum = hex.EncodeToString(hash.Sum(nil))
	r.Report, err = manage.TestRestore(r.file.Name())
	if err != nil {
		return nil, fmt.Errorf("invalid backup: %w", err)
	}
	r.Report.Path = path
	r.client, r.Fingerprint, err = dialSSH(ctx, target.addr(), target)
	if err != nil {
		return nil, err
	}
	out, err := restoreSSHCommand(ctx, r.client, "test \"$(uname -s)\" = Linux && command -v timeout sha256sum >/dev/null && if [ \"$(id -u)\" = 0 ]; then echo root; elif sudo -n true; then echo sudo; else exit 1; fi", nil, sshOpTimeout)
	if err != nil {
		return nil, fmt.Errorf("destination needs Linux, coreutils (timeout/sha256sum), and root or passwordless sudo: %w", err)
	}
	lines := strings.Fields(out)
	if len(lines) == 0 || (lines[len(lines)-1] != "root" && lines[len(lines)-1] != "sudo") {
		return nil, fmt.Errorf("could not verify destination privileges")
	}
	r.sudo = lines[len(lines)-1] == "sudo"
	out, err = restoreSSHCommand(ctx, r.client, r.command("timeout 10 "+app.BinPath+" backup capabilities --json"), nil, 15*time.Second)
	var exitErr *ssh.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return nil, fmt.Errorf("destination capability check failed: %w", err)
	}
	r.NeedsInstall = err != nil || !supportsRestore(out)
	return r, nil
}

func supportsRestore(out string) bool {
	// An SSH login banner may precede the final protocol response.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var c struct {
		Protocol int `json:"restore_protocol"`
	}
	return len(lines) > 0 && json.Unmarshal([]byte(lines[len(lines)-1]), &c) == nil && c.Protocol == 2
}

func (r *RemoteRestore) command(script string) string {
	if r.sudo {
		return "sudo -n sh -c " + quote(script)
	}
	return "sh -c " + quote(script)
}

// Restore must only be called after confirmation: it may install/upgrade the
// destination, replaces settings named in the archive, and restarts tunnels.
func (r *RemoteRestore) Restore(ctx context.Context, progress func(string)) (manage.RestoreResult, error) {
	var result manage.RestoreResult
	if r.client == nil || r.file == nil {
		return result, fmt.Errorf("restore connection is closed")
	}
	if r.NeedsInstall {
		if progress != nil {
			progress("Installing/updating Backpack on the destination...")
		}
		url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/install.sh", app.RepoOwner, app.RepoName, restoreInstallerCommit)
		if _, err := restoreSSHCommand(ctx, r.client, r.command(installerCommand(url, restoreInstallerSHA256)), nil, sshInstallTimeout); err != nil {
			return result, fmt.Errorf("destination installation failed: %w", err)
		}
		out, err := restoreSSHCommand(ctx, r.client, r.command(app.BinPath+" backup capabilities --json"), nil, sshOpTimeout)
		if err != nil || !supportsRestore(out) {
			return result, fmt.Errorf("destination still lacks automatic restore support after installation")
		}
	}
	if _, err := r.file.Seek(0, io.SeekStart); err != nil {
		return result, err
	}
	if progress != nil {
		progress("Transferring, verifying SHA-256 and restoring the backup...")
	}
	out, err := restoreSSHCommand(ctx, r.client, r.command(remoteRestoreScript(r.checksum)), r.file, sshInstallTimeout)
	// Even a partial restore returns its result alongside a nonzero exit status.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var response struct {
		manage.RestoreResult
		Protocol   int    `json:"restore_protocol"`
		PanelError string `json:"panel_error"`
	}
	if len(lines) > 0 && json.Unmarshal([]byte(lines[len(lines)-1]), &response) == nil && response.Protocol == 2 {
		result = response.RestoreResult
		if err == nil && response.PanelError == "" && result.Failed == 0 && len(result.ServicesFailed) == 0 {
			return result, nil
		}
		if response.PanelError != "" {
			return result, fmt.Errorf("settings restored, but the web panel failed: %s", response.PanelError)
		}
		if result.Failed > 0 {
			return result, fmt.Errorf("settings restored, but %d tunnel(s) failed to start", result.Failed)
		}
		if len(result.ServicesFailed) > 0 {
			return result, fmt.Errorf("settings restored, but services could not resume: %s", strings.Join(result.ServicesFailed, ", "))
		}
	}
	if err != nil {
		return result, fmt.Errorf("remote restore failed (verify destination before retrying): %w", err)
	}
	return result, fmt.Errorf("destination did not return a valid restore result; verify it before retrying")
}

func remoteRestoreScript(checksum string) string {
	return "set -eu; umask 077; dir=$(mktemp -d /tmp/backpack-restore.XXXXXXXXXX); " +
		"trap 'rm -rf -- \"$dir\"' EXIT; trap 'exit 1' HUP INT TERM; " +
		"cat > \"$dir/backup.tar.gz\"; " +
		"printf '%s  %s\\n' " + quote(checksum) + " \"$dir/backup.tar.gz\" | sha256sum -c - >&2; " +
		app.BinPath + " backup restore \"$dir/backup.tar.gz\" --yes --json < /dev/null"
}

func (r *RemoteRestore) Close() {
	if r.client != nil {
		r.client.Close()
		r.client = nil
	}
	if r.file != nil {
		name := r.file.Name()
		r.file.Close()
		os.Remove(name)
		r.file = nil
	}
}

// Bound diagnostic output while streaming the archive directly to SSH stdin.
// Session closure on cancellation makes remote shells receive EOF/HUP, and the
// remote cleanup trap removes the private upload on both success and failure.
type restoreOutput struct{ bytes.Buffer }

func (b *restoreOutput) Write(p []byte) (int, error) {
	n := len(p)
	if left := (1 << 20) - b.Len(); left > 0 {
		if len(p) > left {
			p = p[:left]
		}
		b.Buffer.Write(p)
	}
	return n, nil
}

func restoreSSHCommand(ctx context.Context, c *ssh.Client, cmd string, stdin io.Reader, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	stop := context.AfterFunc(ctx, func() { c.Close() })
	defer stop()
	s, err := c.NewSession()
	if err != nil {
		return "", err
	}
	defer s.Close()
	var out, stderr restoreOutput
	s.Stdout, s.Stderr, s.Stdin = &out, &stderr, stdin
	if err := s.Start(cmd); err != nil {
		return "", err
	}
	done := make(chan error, 1)
	go func() { done <- s.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return out.String(), fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return out.String(), nil
	case <-ctx.Done():
		s.Close()
		// Wait drains SSH's output goroutines before reading their buffers.
		<-done
		return out.String(), ctx.Err()
	}
}
