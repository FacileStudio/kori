package sandbox

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// GuardOptions configures pre-flight isolation guard verification.
type GuardOptions struct {
	ExpectedUser    string
	ExpectedWorkdir string
	CheckHarness    bool
	Runner          Runner
}

// GuardResult stores verified attributes returned by the VM probe.
type GuardResult struct {
	User    string
	Workdir string
	Harness string
}

// DefaultGuardOptions returns default isolation verification options.
func DefaultGuardOptions() GuardOptions {
	return GuardOptions{
		ExpectedUser: "boite",
		CheckHarness: true,
		Runner:       NewDefaultRunner(),
	}
}

// BuildGuardProbeCommand generates the shell probe command executed in the VM.
func BuildGuardProbeCommand(workdir string) string {
	cmd := "whoami; pwd; echo VM_HARNESS=$(date +%s)"
	if workdir != "" {
		cmd += fmt.Sprintf("; if [ -d %q ]; then echo WORKDIR_OK; else echo WORKDIR_MISSING; fi", workdir)
	}
	return cmd
}

// ParseGuardOutput parses the stdout lines produced by the VM probe command.
func ParseGuardOutput(output string) (*GuardResult, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("insufficient probe output: %q", output)
	}
	res := &GuardResult{
		User:    strings.TrimSpace(lines[0]),
		Workdir: strings.TrimSpace(lines[1]),
	}
	for _, line := range lines[2:] {
		trimmed := strings.TrimSpace(line)
		if h, ok := strings.CutPrefix(trimmed, "VM_HARNESS="); ok {
			res.Harness = h
		}
	}
	return res, nil
}

// VerifyIsolation ensures that user identity, harness sentinel, and directory match expectations.
func VerifyIsolation(res *GuardResult, rawOut string, opts GuardOptions) error {
	if res == nil {
		return errors.New("guard result is nil")
	}
	if res.User == "root" {
		return errors.New("isolation guard failed: VM user is root, expected unprivileged user")
	}
	if opts.ExpectedUser != "" && res.User != opts.ExpectedUser {
		return fmt.Errorf("isolation guard failed: VM user %q does not match expected %q", res.User, opts.ExpectedUser)
	}
	if opts.CheckHarness && res.Harness == "" {
		return errors.New("isolation guard failed: VM harness check failed")
	}
	if opts.ExpectedWorkdir != "" && !strings.Contains(rawOut, "WORKDIR_OK") {
		return fmt.Errorf("isolation guard failed: project directory %q does not exist in VM", opts.ExpectedWorkdir)
	}
	return nil
}

func runProbe(ctx context.Context, inst *InstanceState, user string, cmd string, runner Runner) ([]byte, error) {
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-p", strconv.Itoa(inst.SSHPort),
		"-i", inst.KeyPath,
		fmt.Sprintf("%s@127.0.0.1", user),
		cmd,
	}
	return runner.Run(ctx, "ssh", args...)
}

// PreflightCheck verifies VM state, connectivity, user isolation, and workspace directory.
func PreflightCheck(ctx context.Context, inst *InstanceState, opts GuardOptions) (*GuardResult, error) {
	if inst == nil {
		return nil, errors.New("instance state is nil")
	}
	if inst.Status != "running" {
		return nil, fmt.Errorf("sandbox %q is %s, must be running (try: boite start %s)", inst.Name, inst.Status, inst.Name)
	}
	runner := opts.Runner
	if runner == nil {
		runner = NewDefaultRunner()
	}
	user := opts.ExpectedUser
	if user == "" {
		user = "boite"
	}
	probeCmd := BuildGuardProbeCommand(opts.ExpectedWorkdir)
	out, err := runProbe(ctx, inst, user, probeCmd, runner)
	if err != nil {
		return nil, fmt.Errorf("preflight isolation probe failed on sandbox %s: %w (%s)", inst.Name, err, strings.TrimSpace(string(out)))
	}
	res, err := ParseGuardOutput(string(out))
	if err != nil {
		return nil, err
	}
	if err := VerifyIsolation(res, string(out), opts); err != nil {
		return nil, err
	}
	return res, nil
}
