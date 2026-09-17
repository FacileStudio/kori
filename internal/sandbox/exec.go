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

// ExecOptions configures interactive SSH execution.
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

func quoteShellArg(p string) string {
	if strings.ContainsAny(p, " \t\n\"'$`\\") {
		return fmt.Sprintf("%q", p)
	}
	return p
}

// FormatRemoteCommand formats the kori invocation string to execute inside the target.
func FormatRemoteCommand(remoteBin string, workDir string, args []string) string {
	bin := remoteBin
	if bin == "" {
		bin = "kori"
	}
	parts := []string{bin}
	hasRoot := false
	for _, arg := range args {
		if arg == "-root" || arg == "--root" || strings.HasPrefix(arg, "-root=") || strings.HasPrefix(arg, "--root=") {
			hasRoot = true
			break
		}
	}
	if workDir != "" && !hasRoot {
		parts = append(parts, "-root", quoteShellArg(workDir))
	}
	for _, arg := range args {
		parts = append(parts, quoteShellArg(arg))
	}
	return strings.Join(parts, " ")
}

func resolveTargetHostPort(target *Target) (string, int) {
	host := "127.0.0.1"
	port := 22
	if target == nil {
		return host, port
	}
	if target.Host != "" {
		host = target.Host
	}
	if target.Port > 0 {
		port = target.Port
	} else if target.Backend == "boite" {
		port = 2226
	}
	return host, port
}

func resolveSSHUserDestination(target *Target, user, host string) string {
	u := user
	if u == "" && target != nil {
		u = target.User
	}
	if u != "" {
		return fmt.Sprintf("%s@%s", u, host)
	}
	return host
}

// BuildSSHArgs constructs the argument list for an interactive SSH invocation with TTY.
func BuildSSHArgs(target *Target, user string, remoteCmd string) []string {
	host, port := resolveTargetHostPort(target)
	args := []string{
		"-tt",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-p", strconv.Itoa(port),
	}
	if target != nil && target.KeyPath != "" {
		args = append(args, "-i", target.KeyPath)
	}
	args = append(args, resolveSSHUserDestination(target, user, host))
	if remoteCmd != "" {
		args = append(args, remoteCmd)
	}
	return args
}

// BuildSSHCommand creates an exec.Cmd configured for interactive SSH into the target.
func BuildSSHCommand(ctx context.Context, target *Target, opts ExecOptions) (*exec.Cmd, error) {
	if target == nil {
		return nil, errors.New("target is nil")
	}
	remoteCmd := FormatRemoteCommand(opts.RemoteBinary, opts.WorkDir, opts.Args)
	sshArgs := BuildSSHArgs(target, opts.User, remoteCmd)
	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	cmd.Stdin = opts.Stdin
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	return cmd, nil
}

// RunSSH launches and awaits an interactive SSH session in the sandbox target.
func RunSSH(ctx context.Context, target *Target, opts ExecOptions) error {
	cmd, err := BuildSSHCommand(ctx, target, opts)
	if err != nil {
		return err
	}
	runner := opts.Runner
	if runner == nil {
		runner = NewDefaultRunner()
	}
	return runner.RunInteractive(ctx, cmd)
}
