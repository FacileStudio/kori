package agent

import (
	"context"
	"encoding/json"
	"errors"
	"runtime"

	"github.com/FacileStudio/nacelle"
	"github.com/FacileStudio/nacelle/tools"
)

// shellCommandInput is the stateless runner's input, field for field, so
// the model sees one schema whichever backend ends up running the command.
type shellCommandInput struct {
	Command string `json:"command" jsonschema:"required,description=The shell command to run. It starts in the working directory"`
	Timeout int    `json:"timeout,omitempty" jsonschema:"description=Seconds to allow before the command is killed. Omit for the default. A value above the configured ceiling is clamped to it"`
}

// shellToolDescription is the stateless runner's description with two
// additions the persistent backend makes true and useful: state survives
// between calls, and a long job belongs in the background with its output
// in a file, since nothing watches it once the call returns.
const shellToolDescription = "Run a shell command in the working directory and return its output. Use it for builds, tests, version control and anything else the other tools do not cover. The shell keeps its state between calls in one session: the current directory, exported variables and background jobs survive, and the first call starts in the working directory. Output is truncated if it is very long, so prefer commands that answer a question over commands that print everything. For a long job, run it in the background with `> log 2>&1 &` and check the log later."

// newShellTool builds the run_command replacement: one persistent shell per
// session, falling back to the stateless runner — passed in as fallback —
// for every call after the persistent shell fails to start. The finalizer
// tears the session down when the tool is dropped; see shellSession for
// the other two teardown paths.
func newShellTool(session *shellSession, fallback nacelle.Tool) (nacelle.Tool, error) {
	tool, err := nacelle.NewOutputTool("run_command", shellToolDescription,
		func(ctx context.Context, in shellCommandInput, emit func(string)) (string, error) {
			return runShellCall(ctx, session, fallback, in, emit)
		})
	if err != nil {
		return nil, err
	}
	runtime.SetFinalizer(tool, func(nacelle.Tool) {
		defer func() { _ = session.Close() }()
	})
	return tool, nil
}

// runShellCall is the handler body: the stateless runner's refusals first,
// then the session, then the stateless runner itself when the persistent
// shell is down.
func runShellCall(ctx context.Context, session *shellSession, fallback nacelle.Tool, in shellCommandInput, emit func(string)) (string, error) {
	if err := session.accept(in.Command); err != nil {
		return "", err
	}
	out, err := session.call(ctx, in.Command, shellBounded(in.Timeout, tools.DefaultCommandTimeout), tools.DefaultMaxOutputBytes, emit)
	if errors.Is(err, errShellDown) {
		return shellFallback(ctx, fallback, in, emit)
	}
	return out, err
}

// shellFallback hands the call to the stateless runner, whose guards and
// result format this tool is at parity with.
func shellFallback(ctx context.Context, fallback nacelle.Tool, in shellCommandInput, emit func(string)) (string, error) {
	raw, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	if out, ok := fallback.(nacelle.OutputTool); ok {
		return out.RunOutput(ctx, raw, emit)
	}
	return fallback.Run(ctx, raw)
}
