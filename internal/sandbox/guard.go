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
	Runner          Runner
}

// GuardResult stores verified attributes returned by the VM probe.
type GuardResult struct {
	User    string
	Workdir string
}

// DefaultGuardOptions returns default isolation verification options.
func DefaultGuardOptions() GuardOptions {
	return GuardOptions{
		ExpectedUser: "boite",
		Runner:       NewDefaultRunner(),
	}
}

// BuildGuardProbeCommand generates the shell probe command executed in the target.
func BuildGuardProbeCommand(workdir string) string {
	cmd := "whoami; pwd"
	if workdir != "" {
		cmd += fmt.Sprintf("; mkdir -p %q 2>/dev/null; if [ -d %q ]; then echo WORKDIR_OK; else echo WORKDIR_MISSING; fi", workdir, workdir)
	}
	return cmd
}

// ParseGuardOutput parses the stdout lines produced by the probe command.
func ParseGuardOutput(output string) (*GuardResult, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("insufficient probe output: %q", output)
	}
	res := &GuardResult{
		User:    strings.TrimSpace(lines[0]),
		Workdir: strings.TrimSpace(lines[1]),
	}
	return res, nil
}

// VerifyIsolation ensures that user identity and directory match expectations.
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
	if opts.ExpectedWorkdir != "" && !strings.Contains(rawOut, "WORKDIR_OK") {
		return fmt.Errorf("isolation guard failed: project directory %q does not exist in VM", opts.ExpectedWorkdir)
	}
	return nil
}

func runProbe(ctx context.Context, target *Target, user string, cmd string, runner Runner) ([]byte, error) {
	host := "127.0.0.1"
	if target.Host != "" {
		host = target.Host
	}
	port := 22
	if target.Port > 0 {
		port = target.Port
	} else if target.Backend == "boite" {
		port = 2226
	}
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ForwardAgent=no",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-p", strconv.Itoa(port),
	}
	if target.KeyPath != "" {
		args = append(args, "-i", target.KeyPath)
	}
	u := user
	if u == "" {
		u = target.User
	}
	if u != "" {
		args = append(args, fmt.Sprintf("%s@%s", u, host))
	} else {
		args = append(args, host)
	}
	args = append(args, cmd)
	return runner.Run(ctx, "ssh", args...)
}

// PreflightCheck verifies target state, connectivity, user isolation, and workspace directory.
func PreflightCheck(ctx context.Context, target *Target, opts GuardOptions) (*GuardResult, error) {
	if target == nil {
		return nil, errors.New("target is nil")
	}
	if target.Backend == "boite" && target.Status != "running" {
		return nil, fmt.Errorf("sandbox %q is %s, must be running (try: boite start %s)", target.Name, target.Status, target.Name)
	}
	runner := opts.Runner
	if runner == nil {
		runner = NewDefaultRunner()
	}
	user := opts.ExpectedUser
	if user == "" {
		user = target.User
	}
	if user == "" && target.Backend == "boite" {
		user = "boite"
	}
	probeCmd := BuildGuardProbeCommand(opts.ExpectedWorkdir)
	out, err := runProbe(ctx, target, user, probeCmd, runner)
	if err != nil {
		return nil, fmt.Errorf("preflight isolation probe failed on sandbox %s: %w (%s)", target.Name, err, strings.TrimSpace(string(out)))
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
