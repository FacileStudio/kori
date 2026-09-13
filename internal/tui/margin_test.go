// Package tui's live-region margin tests. Kept out of view_test.go, which sits
// at the filet per-file line cap (250): the repo's pattern for a capped file is
// a dedicated test file next to it (compact_pair_test.go, parallel_*_test.go).
package tui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/FacileStudio/nacelle"
)

// Every live row the assembly draws carries a single space on both sides, so
// streaming, status, queue and tasks text never touches the screen edges and
// no row is pushed past the width into a wrap — the margin is one space added
// to a row shaved to m.width-2, never an extra row.
func TestLiveRowsCarryAMargin(t *testing.T) {
	m := sized()
	m.Add("a queued message")
	m.run.busy = true
	m.Expanded = true
	m.run.reasoning.WriteString("some reasoning")
	m.absorb(nacelle.Event{Kind: nacelle.KindText, Text: "an answer that runs on and on and on and on and on and on and on and on"})
	b := "batch"
	m.parallelTasks = map[string][]parallelTaskInfo{b: {{Task: "one", Active: true, Began: time.Now()}}}
	for row := range strings.SplitSeq(strings.Join(m.aboveContent(), "\n"), "\n") {
		got := visible(row)
		if got == "" {
			continue
		}
		if !strings.HasSuffix(got, " ") || (!strings.HasPrefix(got, " ") && !strings.HasPrefix(got, "▌")) {
			t.Errorf("row %q does not carry the margin spaces", row)
		}
		if w := lipgloss.Width(row); w > m.width {
			t.Errorf("row is %d cells wide, want at most %d so the margin never wraps it", w, m.width)
		}
	}
}

// The queue sits one blank row clear of the status/stats line above it, so
// waiting messages never touch the counts.
func TestTheQueueSitsOneBlankRowBelowTheStats(t *testing.T) {
	m := sized()
	m.Add("a queued message")
	rows := strings.Split(strings.Join(m.aboveContent(), "\n"), "\n")
	queueAt := -1
	for i, row := range rows {
		if strings.Contains(visible(row), "a queued message") {
			queueAt = i
		}
	}
	if queueAt < 3 {
		t.Fatalf("no queue row in\n%s", strings.Join(rows, "\n"))
	}
	if rows[queueAt-1] != "" {
		t.Errorf("row above the queue = %q, want a blank line between the stats and the queue", rows[queueAt-1])
	}
	if stats := visible(rows[queueAt-2]); !strings.Contains(stats, "↓") {
		t.Errorf("row above the blank = %q, want the stats line", rows[queueAt-2])
	}
}

// A running tool call keeps one blank row above it in the live region, the
// same separator the finished block gets when it commits — otherwise the row
// sits flush under the last printed line only while it runs.
func TestLiveToolGroupKeepsABlankRowAbove(t *testing.T) {
	m := sized()
	m.run.busy = true
	tool := &nacelle.ToolEvent{ID: "1", Name: "run_command", Input: `{"command":"ls"}`}
	m.absorb(nacelle.Event{Kind: nacelle.KindToolCall, Tool: tool})
	rows := strings.Split(strings.Join(m.aboveContent(), "\n"), "\n")
	if rows[0] != "" {
		t.Errorf("first live row = %q, want a blank line above the running tool call", visible(rows[0]))
	}
	if !strings.Contains(visible(strings.Join(rows, "\n")), "run_command") {
		t.Fatalf("no live tool row in\n%s", strings.Join(rows, "\n"))
	}
}

// The same margin applies to the parallel-task rows under the prompt.
func TestParallelTaskRowsAreMarginedToo(t *testing.T) {
	m := sized()
	b := "batch"
	m.parallelTasks = map[string][]parallelTaskInfo{b: {{Task: "one", Active: true, Began: time.Now()}}}
	rows := strings.Split(m.belowContent(), "\n")
	if len(rows) == 0 {
		t.Fatalf("no parallel rows under the prompt")
	}
	for _, row := range rows {
		got := visible(row)
		if got == "" {
			continue
		}
		if !strings.HasPrefix(got, " ") || !strings.HasSuffix(got, " ") {
			t.Errorf("parallel row %q does not carry the margin spaces", row)
		}
		if w := lipgloss.Width(row); w > m.width {
			t.Errorf("parallel row is %d cells wide, want at most %d", w, m.width)
		}
	}
}
