package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ExecOptions configures interactive SSH session execution into the sandbox.
type ExecOptions struct {
	Stdin        io.Reader
	Stdout       io.Writer
	Stderr       io.Writer
	User         string
	WorkDir      string
	RemoteBinary string
	Args         []string
	Runner       Runner
}

// DefaultExecOptions returns standard SSH execution options.
func DefaultExecOptions() ExecOptions {
	return ExecOptions{
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		User:    "boite",
		WorkDir: "/workspace",
		Runner:  NewDefaultRunner(),
	}
}

func hasRootArg(args []string) bool {
	for _, arg := range args {
		if arg == "-root" || arg == "--root" || strings.HasPrefix(arg, "-root=") || strings.HasPrefix(arg, "--root=") {
			return true
		}
	}
	return false
}

func quoteShellArg(p string) string {
	if strings.ContainsAny(p, " \t\n\"'$`\\") {
		return fmt.Sprintf("%q", p)
	}
	return p
}

// FormatRemoteCommand formats the kori invocation string to execute inside the VM.
func FormatRemoteCommand(remoteBin string, workDir string, args []string) string {
	bin := remoteBin
	if bin == "" {
		bin = "kori"
	}
	parts := []string{bin}
	if workDir != "" && !hasRootArg(args) {
		parts = append(parts, "-root", workDir)
	}
	for _, arg := range args {
		parts = append(parts, quoteShellArg(arg))
	}
	return strings.Join(parts, " ")
}

// BuildSSHArgs constructs the argument list for an interactive SSH invocation with TTY.
func BuildSSHArgs(inst *InstanceState, user string, remoteCmd string) []string {
	u := user
	if u == "" {
		u = "boite"
	}
	args := []string{
		"-tt",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-p", strconv.Itoa(inst.SSHPort),
		"-i", inst.KeyPath,
		fmt.Sprintf("%s@127.0.0.1", u),
	}
	if remoteCmd != "" {
		args = append(args, remoteCmd)
	}
	return args
}

// BuildSSHCommand creates an exec.Cmd configured for interactive SSH into the VM.
func BuildSSHCommand(ctx context.Context, inst *InstanceState, opts ExecOptions) (*exec.Cmd, error) {
	if inst == nil {
		return nil, errors.New("instance state is nil")
	}
	remoteCmd := FormatRemoteCommand(opts.RemoteBinary, opts.WorkDir, opts.Args)
	sshArgs := BuildSSHArgs(inst, opts.User, remoteCmd)
	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	cmd.Stdin = opts.Stdin
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	return cmd, nil
}

// RunSSH launches and awaits an interactive SSH session in the VM sandbox.
func RunSSH(ctx context.Context, inst *InstanceState, opts ExecOptions) error {
	cmd, err := BuildSSHCommand(ctx, inst, opts)
	if err != nil {
		return err
	}
	runner := opts.Runner
	if runner == nil {
		runner = NewDefaultRunner()
	}
	return runner.RunInteractive(ctx, cmd)
}
