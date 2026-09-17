package sandbox

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// SyncOptions configures local-to-VM binary deployment.
type SyncOptions struct {
	LocalPath  string
	RemotePath string
	User       string
	Force      bool
	Runner     Runner
}

// DefaultSyncOptions returns the default binary synchronization configuration.
func DefaultSyncOptions() SyncOptions {
	return SyncOptions{
		RemotePath: "/tmp/kori",
		User:       "boite",
		Runner:     NewDefaultRunner(),
	}
}

// FindLocalBinary locates the running or installed kori binary on the host.
func FindLocalBinary() (string, error) {
	if exe, err := os.Executable(); err == nil && filepath.Base(exe) == "kori" {
		return exe, nil
	}
	if info, err := os.Stat("kori"); err == nil && !info.IsDir() {
		abs, err := filepath.Abs("kori")
		if err == nil {
			return abs, nil
		}
	}
	if path, err := exec.LookPath("kori"); err == nil {
		return path, nil
	}
	return "", errors.New("kori executable not found on host")
}

func buildSyncSSHBaseArgs(target *Target, user string) []string {
	host, port := resolveTargetHostPort(target)
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-p", strconv.Itoa(port),
	}
	if target != nil && target.KeyPath != "" {
		args = append(args, "-i", target.KeyPath)
	}
	args = append(args, resolveSSHUserDestination(target, user, host))
	return args
}

// CheckRemoteBinary verifies if kori is already executable inside the target.
func CheckRemoteBinary(ctx context.Context, target *Target, opts SyncOptions) (string, error) {
	if target == nil {
		return "", errors.New("target is nil")
	}
	runner := resolveRunner(opts.Runner)
	args := buildSyncSSHBaseArgs(target, opts.User)
	checkCmd := "if command -v kori >/dev/null 2>&1; then command -v kori; elif [ -x ~/.local/bin/kori ]; then echo ~/.local/bin/kori; elif [ -x /tmp/kori ]; then echo /tmp/kori; else exit 1; fi"
	args = append(args, checkCmd)
	out, err := runner.Run(ctx, "ssh", args...)
	if err != nil {
		return "", err
	}
	remote := strings.TrimSpace(string(out))
	if remote == "" {
		return "", errors.New("no remote binary detected")
	}
	return remote, nil
}

func buildSCPArgs(target *Target, user, local, remote string) []string {
	host, port := resolveTargetHostPort(target)
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-P", strconv.Itoa(port),
	}
	if target != nil && target.KeyPath != "" {
		args = append(args, "-i", target.KeyPath)
	}
	dest := fmt.Sprintf("%s:%s", resolveSSHUserDestination(target, user, host), remote)
	return append(args, local, dest)
}

func copyFileSCP(ctx context.Context, target *Target, opts SyncOptions, local, remote string) error {
	runner := resolveRunner(opts.Runner)
	args := buildSCPArgs(target, opts.User, local, remote)
	if out, err := runner.Run(ctx, "scp", args...); err != nil {
		return fmt.Errorf("scp failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// CopyBinaryToVM copies a local executable binary to the remote target and sets execute permissions.
func CopyBinaryToVM(ctx context.Context, target *Target, opts SyncOptions, localPath string) error {
	if target == nil {
		return errors.New("target is nil")
	}
	remote := cmp.Or(opts.RemotePath, "/tmp/kori")
	if err := copyFileSCP(ctx, target, opts, localPath, remote); err != nil {
		return err
	}
	runner := resolveRunner(opts.Runner)
	chmodArgs := append(buildSyncSSHBaseArgs(target, opts.User), fmt.Sprintf("chmod +x %s", remote))
	if out, err := runner.Run(ctx, "ssh", chmodArgs...); err != nil {
		return fmt.Errorf("remote chmod failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// EnsureKoriBinary ensures the kori binary is installed and executable inside the target sandbox.
func EnsureKoriBinary(ctx context.Context, target *Target, opts SyncOptions) (string, error) {
	if target == nil {
		return "", errors.New("target is nil")
	}
	if !opts.Force {
		if path, err := CheckRemoteBinary(ctx, target, opts); err == nil && path != "" {
			return path, nil
		}
	}
	localPath := opts.LocalPath
	if localPath == "" {
		found, err := FindLocalBinary()
		if err != nil {
			return "", fmt.Errorf("cannot sync kori binary: %w", err)
		}
		localPath = found
	}
	remoteTarget := cmp.Or(opts.RemotePath, "/tmp/kori")
	if err := CopyBinaryToVM(ctx, target, opts, localPath); err != nil {
		return "", err
	}
	return remoteTarget, nil
}
