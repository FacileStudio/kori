package agent

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// shellGrace is what a command gets between being asked to stop and being
// made to, the same window the stateless runner allows.
const shellGrace = 2 * time.Second

// shellTimedOut is a command killed for running too long. The stateless
// runner's error prints its grace constant here; this one prints the
// timeout the command actually had.
type shellTimedOut struct{ after time.Duration }

func (e shellTimedOut) Error() string { return fmt.Sprintf("timed out after %s", e.after) }

// shellBounded is the timeout one call actually gets: the model may ask for
// less than the ceiling and never for more. Anything absent, zero, negative
// or above the ceiling falls back to the ceiling, because a bad guess at a
// timeout should cost a shorter command, not a failed tool call. The
// comparison happens in seconds on purpose: converting first can overflow a
// Duration negative, which would expire the moment it was set.
func shellBounded(seconds int, ceiling time.Duration) time.Duration {
	if seconds <= 0 || time.Duration(seconds) > ceiling/time.Second {
		return ceiling
	}
	return time.Duration(seconds) * time.Second
}

// shellReport shapes a tool result the way the stateless runner does:
// output trimmed of trailing newlines and capped with a truncation notice,
// "(no output)" when empty, and a bracketed status line when the command
// did not end cleanly. rc is the exit status when it is known; failure
// carries a timeout, a cancelled context, or the death of the shell itself.
func shellReport(output string, rc int, failure error, limit int) string {
	body := shellTruncate(strings.TrimRight(output, "\n"), limit)
	if body == "" {
		body = "(no output)"
	}

	switch {
	case failure == nil:
		if rc != 0 {
			return fmt.Sprintf("%s\n\n[exit status %d]", body, rc)
		}
		return body
	case errors.As(failure, &shellTimedOut{}):
		return body + "\n\n[" + failure.Error() + "; the command and its children were killed]"
	default:
		if exit, ok := errors.AsType[*exec.ExitError](failure); ok {
			return fmt.Sprintf("%s\n\n[exit status %d]", body, exit.ExitCode())
		}
		return fmt.Sprintf("%s\n\n[%s]", body, failure)
	}
}

// shellTruncate caps text at limit, saying so rather than trimming in
// silence, cutting at a line boundary when one fits.
func shellTruncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	kept := text[:limit]
	if cut := strings.LastIndexByte(kept, '\n'); cut > limit/2 {
		kept = kept[:cut]
	}
	return kept + fmt.Sprintf("\n\n[truncated: %d of %d bytes shown]", len(kept), len(text))
}

// shellKillGroup signals the shell and everything it started, reporting
// whether there was still a group there to receive it. The negative pid is
// the whole point: kill(-pgid) reaches the group, where kill(pid) reaches
// only the shell that spawned the work.
func shellKillGroup(cmd *exec.Cmd, signal syscall.Signal) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	return syscall.Kill(-cmd.Process.Pid, signal) == nil
}

// shellExitCode reports the exit status an ExitError carries, and -1 for
// anything else.
func shellExitCode(err error) int {
	if exit, ok := errors.AsType[*exec.ExitError](err); ok {
		return exit.ExitCode()
	}
	return -1
}
