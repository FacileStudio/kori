package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/nacelle"
)

// The row shows the subagent's currently-running tool after the title's colon:
// a yellow spinner leads while the task runs (replacing the old ≫ marker) and
// the tool's own glyph follows the title, coloured by the tool's rules.
func TestParallelTaskRowShowsTheRunningTool(t *testing.T) {
	m := sized()
	m.parallelTasks = make(map[string][]parallelTaskInfo)
	m.parallelTasks["t0"] = make([]parallelTaskInfo, 1)
	m.parallelTasks["t0"][0] = parallelTaskInfo{
		Task:   "analyze the codebase for todos",
		Title:  "scan site for todos",
		Tool:   "run_command",
		Began:  time.Now(),
		Active: true,
	}

	raw := m.taskRow(m.parallelTasks["t0"][0])
	got := visible(raw)

	if strings.Contains(got, "≫") {
		t.Errorf("row = %q, the spinner replaced the ≫ marker", got)
	}
	if !strings.HasPrefix(got, m.spin.View()) {
		t.Errorf("row = %q, want a leading spinner while the task runs", got)
	}
	if !strings.Contains(got, "scan site for todos:") {
		t.Errorf("row = %q, want the short title followed by a colon", got)
	}
	if !strings.Contains(got, "$ run_command") {
		t.Errorf("row = %q, want the tool glyph with its name", got)
	}
	if !strings.Contains(raw, "\x1b[33m$") {
		t.Errorf("running tool glyph not yellow while the call is in flight: %q", raw)
	}
	if !strings.Contains(raw, "\x1b[35m run_command") {
		t.Errorf("tool name not in the tool's own tone: %q", raw)
	}
}

// Between tool calls the row names the agent's own state: waiting before the
// first tokens arrive, thinking once the model has streamed some.
func TestTaskRowShowsThinkingAndWaitingBetweenTools(t *testing.T) {
	m := sized()
	m.parallelTasks = make(map[string][]parallelTaskInfo)
	m.parallelTasks["t0"] = make([]parallelTaskInfo, 1)
	m.parallelTasks["t0"][0] = parallelTaskInfo{Task: "one", Began: time.Now(), Active: true}

	raw := m.taskRow(m.parallelTasks["t0"][0])
	if !strings.Contains(raw, "waiting") {
		t.Errorf("row = %q, want a waiting status before the first tokens", raw)
	}

	m.parallelTasks["t0"][0].Usage = nacelle.Usage{InputTokens: 10, OutputTokens: 5}
	raw = m.taskRow(m.parallelTasks["t0"][0])
	if strings.Contains(raw, "waiting") || !strings.Contains(raw, "thinking") {
		t.Errorf("row = %q, want thinking once tokens have streamed", raw)
	}
}

// A finished, clean task row leads with a green check; a failed task leads with
// a red cross, so the outcome reads at a glance without opening the results.
func TestFinishedTaskRowShowsAGreenCheck(t *testing.T) {
	m := sized()
	m.parallelTasks = make(map[string][]parallelTaskInfo)
	m.parallelTasks["t0"] = make([]parallelTaskInfo, 1)
	m.parallelTasks["t0"][0] = parallelTaskInfo{Task: "one", Began: time.Now(), End: time.Now(), Active: false}

	raw := m.taskRow(m.parallelTasks["t0"][0])
	got := visible(raw)
	if !strings.Contains(got, "✓") {
		t.Errorf("row = %q, want a green check on a finished task", got)
	}
	if !strings.Contains(raw, "\x1b[32m✓") {
		t.Errorf("finished check not green: %q", raw)
	}
}

func TestFailedTaskRowShowsARedCross(t *testing.T) {
	m := sized()
	m.parallelTasks = make(map[string][]parallelTaskInfo)
	m.parallelTasks["t0"] = make([]parallelTaskInfo, 1)
	m.parallelTasks["t0"][0] = parallelTaskInfo{Task: "one", Err: "boom", Began: time.Now(), End: time.Now(), Active: false}

	raw := m.taskRow(m.parallelTasks["t0"][0])
	got := visible(raw)
	if !strings.Contains(got, "✗") {
		t.Errorf("row = %q, want a red cross on a failed task", got)
	}
	if !strings.Contains(raw, "\x1b[31m✗") {
		t.Errorf("failed cross not red: %q", raw)
	}
}

// The tool glyph flips green the moment a running task's call succeeds, and red
// when it fails, while the task itself keeps running.
func TestToolGlyphTurnsGreenOnSuccess(t *testing.T) {
	m := sized()
	m.parallelTasks = make(map[string][]parallelTaskInfo)
	m.parallelTasks["t0"] = make([]parallelTaskInfo, 1)
	m.parallelTasks["t0"][0] = parallelTaskInfo{Task: "one", Tool: "read_file", ToolOut: "ok", Active: true}

	raw := m.taskRow(m.parallelTasks["t0"][0])
	if !strings.Contains(raw, "\x1b[32m⏺") {
		t.Errorf("succeeded tool glyph not green: %q", raw)
	}
	if !strings.Contains(raw, "\x1b[34m read_file") {
		t.Errorf("tool name not in the tool's own tone after success: %q", raw)
	}
}

func TestToolGlyphTurnsRedOnFailure(t *testing.T) {
	m := sized()
	m.parallelTasks = make(map[string][]parallelTaskInfo)
	m.parallelTasks["t0"] = make([]parallelTaskInfo, 1)
	m.parallelTasks["t0"][0] = parallelTaskInfo{Task: "one", Tool: "run_command", ToolOut: "boom", Active: true}

	raw := m.taskRow(m.parallelTasks["t0"][0])
	if !strings.Contains(raw, "\x1b[31m$") {
		t.Errorf("failed tool glyph not red: %q", raw)
	}
}

// The elapsed clock wears the row's run-state tone: yellow while the task
// runs, green once its duration freezes, red when it failed.
func TestClockColourFollowsTheRunState(t *testing.T) {
	m := sized()
	m.parallelTasks = make(map[string][]parallelTaskInfo)
	m.parallelTasks["t0"] = make([]parallelTaskInfo, 1)
	pt := &m.parallelTasks["t0"][0]
	began := time.Now().Add(-time.Minute)

	*pt = parallelTaskInfo{Task: "one", Began: began, Active: true}
	if raw := m.taskRow(*pt); !strings.HasSuffix(raw, taskTone(*pt).Render(taskClock(*pt))) {
		t.Errorf("running row %q, want the live clock yellow at its end", raw)
	}

	*pt = parallelTaskInfo{Task: "one", Began: began, End: began.Add(time.Minute), Active: false}
	if raw := m.taskRow(*pt); !strings.HasSuffix(raw, taskTone(*pt).Render(taskClock(*pt))) {
		t.Errorf("finished row %q, want the frozen clock green at its end", raw)
	}

	*pt = parallelTaskInfo{Task: "one", Began: began, End: began.Add(time.Minute), Err: "boom", Active: false}
	if raw := m.taskRow(*pt); !strings.HasSuffix(raw, taskTone(*pt).Render(taskClock(*pt))) {
		t.Errorf("failed row %q, want the frozen clock red at its end", raw)
	}
}
