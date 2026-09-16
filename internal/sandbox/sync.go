package sandbox

import (
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

// CheckRemoteBinary verifies if kori is already executable inside the VM.
func CheckRemoteBinary(ctx context.Context, inst *InstanceState, opts SyncOptions) (string, error) {
	checkCmd := "if command -v kori >/dev/null 2>&1; then command -v kori; elif [ -x ~/.local/bin/kori ]; then echo ~/.local/bin/kori; elif [ -x /tmp/kori ]; then echo /tmp/kori; else exit 1; fi"
	user := opts.User
	if user == "" {
		user = "boite"
	}
	runner := opts.Runner
	if runner == nil {
		runner = NewDefaultRunner()
	}
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-p", strconv.Itoa(inst.SSHPort),
		"-i", inst.KeyPath,
		fmt.Sprintf("%s@127.0.0.1", user),
		checkCmd,
	}
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

func copyFileSCP(ctx context.Context, inst *InstanceState, opts SyncOptions, local, remote string) error {
	user := opts.User
	if user == "" {
		user = "boite"
	}
	runner := opts.Runner
	if runner == nil {
		runner = NewDefaultRunner()
	}
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-P", strconv.Itoa(inst.SSHPort),
		"-i", inst.KeyPath,
		local,
		fmt.Sprintf("%s@127.0.0.1:%s", user, remote),
	}
	if out, err := runner.Run(ctx, "scp", args...); err != nil {
		return fmt.Errorf("scp failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// CopyBinaryToVM copies a local executable binary to the remote VM and sets execute permissions.
func CopyBinaryToVM(ctx context.Context, inst *InstanceState, opts SyncOptions, localPath string) error {
	user := opts.User
	if user == "" {
		user = "boite"
	}
	runner := opts.Runner
	if runner == nil {
		runner = NewDefaultRunner()
	}
	remote := opts.RemotePath
	if remote == "" {
		remote = "/tmp/kori"
	}
	if err := copyFileSCP(ctx, inst, opts, localPath, remote); err != nil {
		return err
	}
	chmodArgs := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-p", strconv.Itoa(inst.SSHPort),
		"-i", inst.KeyPath,
		fmt.Sprintf("%s@127.0.0.1", user),
		fmt.Sprintf("chmod +x %s", remote),
	}
	if out, err := runner.Run(ctx, "ssh", chmodArgs...); err != nil {
		return fmt.Errorf("remote chmod failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func resolveLocalBinary(localPath string) (string, error) {
	if localPath != "" {
		return localPath, nil
	}
	found, err := FindLocalBinary()
	if err != nil {
		return "", fmt.Errorf("cannot sync kori binary: %w", err)
	}
	return found, nil
}

// EnsureKoriBinary ensures the kori binary is installed and executable inside the VM sandbox.
func EnsureKoriBinary(ctx context.Context, inst *InstanceState, opts SyncOptions) (string, error) {
	if inst == nil {
		return "", errors.New("instance state is nil")
	}
	if !opts.Force {
		if path, err := CheckRemoteBinary(ctx, inst, opts); err == nil && path != "" {
			return path, nil
		}
	}
	localPath, err := resolveLocalBinary(opts.LocalPath)
	if err != nil {
		return "", err
	}
	remoteTarget := opts.RemotePath
	if remoteTarget == "" {
		remoteTarget = "/tmp/kori"
	}
	if err := CopyBinaryToVM(ctx, inst, opts, localPath); err != nil {
		return "", err
	}
	return remoteTarget, nil
}
