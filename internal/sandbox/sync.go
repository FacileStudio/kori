package sandbox

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// SyncOptions configures file synchronization options.
type SyncOptions struct {
	User   string
	Runner Runner
}

// DefaultSyncOptions returns default sync configuration.
func DefaultSyncOptions() SyncOptions {
	return SyncOptions{
		User:   "boite",
		Runner: NewDefaultRunner(),
	}
}

func buildSCPArgs(target *Target, user, local, remote string) []string {
	host, port := resolveTargetHostPort(target)
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ForwardAgent=no",
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

// CopyFileToVM copies a file to the target over SCP.
func CopyFileToVM(ctx context.Context, target *Target, opts SyncOptions, local, remote string) error {
	if target == nil {
		return errors.New("target is nil")
	}
	runner := opts.Runner
	if runner == nil {
		runner = NewDefaultRunner()
	}
	args := buildSCPArgs(target, opts.User, local, remote)
	if out, err := runner.Run(ctx, "scp", args...); err != nil {
		return fmt.Errorf("scp failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
