package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/FacileStudio/nacelle"
)

type commandInput struct {
	Command string `json:"command" jsonschema:"required,description=The shell command to run. It starts in the working directory"`
	Timeout int    `json:"timeout,omitempty" jsonschema:"description=Seconds to allow before the command is killed. Omit for the default. A value above the configured ceiling is clamped to it"`
}

func buildCommandTool(s *remoteSession) (nacelle.Tool, error) {
	return nacelle.NewTool("run_command",
		"Run a shell command in the working directory and return its output. Use it for builds, tests, version control and anything else the other tools do not cover. Output is truncated if it is very long, so prefer commands that answer a question over commands that print everything.",
		func(ctx context.Context, in commandInput) (string, error) {
			return runCommand(ctx, s, in)
		})
}

func boundTimeout(seconds int, ceiling time.Duration) time.Duration {
	if seconds <= 0 || time.Duration(seconds) > ceiling/time.Second {
		return ceiling
	}
	return time.Duration(seconds) * time.Second
}

func reportCommandOutput(output string, err error, limit int) string {
	body := truncateText(strings.TrimRight(output, "\n"), limit)
	if body == "" {
		body = "(no output)"
	}
	if err == nil {
		return body
	}
	if exit, ok := errors.AsType[*exec.ExitError](err); ok {
		return fmt.Sprintf("%s\n\n[exit status %d]", body, exit.ExitCode())
	}
	return fmt.Sprintf("%s\n\n[%s]", body, err)
}

func truncateText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	kept := text[:limit]
	if cut := strings.LastIndexByte(kept, '\n'); cut > limit/2 {
		kept = kept[:cut]
	}
	return kept + fmt.Sprintf("\n\n[truncated: %d of %d bytes shown]", len(kept), len(text))
}

func runCommand(ctx context.Context, s *remoteSession, in commandInput) (string, error) {
	if strings.TrimSpace(in.Command) == "" {
		return "", fmt.Errorf("no command given")
	}
	timeout := boundTimeout(in.Timeout, s.opts.CommandTimeout)
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	remoteCmd := fmt.Sprintf("%s(%s\n)", workdirPrefix(s.opts.WorkDir), in.Command)
	out, err := s.runSSH(cmdCtx, remoteCmd)
	if errors.Is(cmdCtx.Err(), context.Canceled) {
		return reportCommandOutput(string(out), cmdCtx.Err(), s.opts.MaxOutputBytes), cmdCtx.Err()
	}
	if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
		return reportCommandOutput(string(out), fmt.Errorf("timed out after %s", timeout), s.opts.MaxOutputBytes), nil
	}
	return reportCommandOutput(string(out), err, s.opts.MaxOutputBytes), nil
}
