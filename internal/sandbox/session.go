package sandbox

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// SessionOptions configures the full lifecycle of running a session inside a sandbox target.
type SessionOptions struct {
	Target      *Target
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

func checkGuard(ctx context.Context, target *Target, opts SessionOptions, r Runner) error {
	if opts.SkipGuard {
		return nil
	}
	guardOpts := GuardOptions{
		ExpectedUser:    opts.User,
		ExpectedWorkdir: opts.WorkDir,
		Runner:          r,
	}
	_, err := PreflightCheck(ctx, target, guardOpts)
	return err
}

func execSSH(ctx context.Context, target *Target, opts SessionOptions, remoteBin string, r Runner) error {
	execOpts := DefaultExecOptions()
	if opts.User != "" {
		execOpts.User = opts.User
	}
	if opts.WorkDir != "" {
		execOpts.WorkDir = opts.WorkDir
	}
	if remoteBin != "" {
		execOpts.RemoteBinary = remoteBin
	}
	execOpts.Args = opts.Args
	if opts.PrintPrompt != "" {
		execOpts.Args = append(execOpts.Args, "--print", opts.PrintPrompt)
	}
	execOpts.Runner = r
	return RunSSH(ctx, target, execOpts)
}

func takePostSnapshot(ctx context.Context, target *Target, opts SessionOptions, r Runner) error {
	if !opts.Snapshot || target.Backend != "boite" {
		return nil
	}
	tag := opts.SnapshotTag
	if tag == "" {
		tag = fmt.Sprintf("session-%d", time.Now().Unix())
	}
	snapOpts := SnapshotOptions{Runner: r}
	if err := TakeSnapshot(ctx, target.Name, tag, snapOpts); err != nil {
		return fmt.Errorf("snapshot error: %w", err)
	}
	return nil
}

// RunSession launches a complete sandboxed session in the target VM or remote SSH host.
func RunSession(ctx context.Context, opts SessionOptions) error {
	target := opts.Target
	if target == nil {
		if opts.VMName == "" {
			return errors.New("sandbox target is required")
		}
		inst, err := LoadInstance(opts.VMName)
		if err != nil {
			return fmt.Errorf("loading sandbox instance %s: %w", opts.VMName, err)
		}
		target = TargetFromInstance(inst)
	}
	runner := opts.Runner
	if runner == nil {
		runner = NewDefaultRunner()
	}
	if err := checkGuard(ctx, target, opts, runner); err != nil {
		return err
	}
	execErr := execSSH(ctx, target, opts, "", runner)
	if err := takePostSnapshot(ctx, target, opts, runner); err != nil {
		return err
	}
	return execErr
}
