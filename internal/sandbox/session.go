package sandbox

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// SessionOptions configures the full lifecycle of running a session inside a sandbox VM.
type SessionOptions struct {
	VMName      string
	WorkDir     string
	User        string
	Sync        bool
	NoSync      bool
	Snapshot    bool
	SnapshotTag string
	PrintPrompt string
	Args        []string
	SkipGuard   bool
	Runner      Runner
}

func checkGuard(ctx context.Context, inst *InstanceState, opts SessionOptions, r Runner) error {
	if opts.SkipGuard {
		return nil
	}
	guardOpts := GuardOptions{
		ExpectedUser:    opts.User,
		ExpectedWorkdir: opts.WorkDir,
		CheckHarness:    true,
		Runner:          r,
	}
	_, err := PreflightCheck(ctx, inst, guardOpts)
	return err
}

func syncBinary(ctx context.Context, inst *InstanceState, opts SessionOptions, r Runner) error {
	if !opts.Sync || opts.NoSync {
		return nil
	}
	syncOpts := SyncOptions{
		User:   opts.User,
		Runner: r,
	}
	if _, err := EnsureKoriBinary(ctx, inst, syncOpts); err != nil {
		return fmt.Errorf("syncing binary: %w", err)
	}
	return nil
}

func execSSH(ctx context.Context, inst *InstanceState, opts SessionOptions, r Runner) error {
	execOpts := DefaultExecOptions()
	if opts.User != "" {
		execOpts.User = opts.User
	}
	if opts.WorkDir != "" {
		execOpts.WorkDir = opts.WorkDir
	}
	execOpts.Args = opts.Args
	if opts.PrintPrompt != "" {
		execOpts.Args = append(execOpts.Args, "--print", opts.PrintPrompt)
	}
	execOpts.Runner = r
	return RunSSH(ctx, inst, execOpts)
}

func takePostSnapshot(ctx context.Context, opts SessionOptions, r Runner) error {
	if !opts.Snapshot {
		return nil
	}
	tag := opts.SnapshotTag
	if tag == "" {
		tag = fmt.Sprintf("session-%d", time.Now().Unix())
	}
	snapOpts := SnapshotOptions{Runner: r}
	if err := TakeSnapshot(ctx, opts.VMName, tag, snapOpts); err != nil {
		return fmt.Errorf("snapshot error: %w", err)
	}
	return nil
}

// RunSession launches a complete sandboxed session in the named boite VM.
func RunSession(ctx context.Context, opts SessionOptions) error {
	if opts.VMName == "" {
		return errors.New("vm name is required")
	}
	inst, err := LoadInstance(opts.VMName)
	if err != nil {
		return fmt.Errorf("loading sandbox instance %s: %w", opts.VMName, err)
	}
	runner := opts.Runner
	if runner == nil {
		runner = NewDefaultRunner()
	}
	if err := checkGuard(ctx, inst, opts, runner); err != nil {
		return err
	}
	if err := syncBinary(ctx, inst, opts, runner); err != nil {
		return err
	}
	execErr := execSSH(ctx, inst, opts, runner)
	if err := takePostSnapshot(ctx, opts, runner); err != nil {
		return err
	}
	return execErr
}
