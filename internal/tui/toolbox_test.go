package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

// A run_command's output is drawn as a full-width box sharing the diff's pane,
// left border green for a call that worked, so a command's stdout and stderr
// are as visible in the transcript as the file edits around them.
func TestARunCommandOutputRendersAsABox(t *testing.T) {
	m := sized()
	m.absorb(called("c", "run_command", `{"command":"go test"}`))
	m.absorb(nacelle.Event{Kind: nacelle.KindToolResult, Tool: &nacelle.ToolEvent{
		ID: "c", Name: "run_command", Input: `{"command":"go test"}`, Result: "ok   package\nPASS",
	}})

	said := strings.Join(m.unprinted, "\n")
	if strings.Contains(said, "38;5;3") {
		t.Errorf("output = %q, want no ANSI256 grey-range border escape", said)
	}
	if !strings.Contains(visible(said), "ok   package") || !strings.Contains(visible(said), "PASS") {
		t.Errorf("output = %q, want the command's output in the box", visible(said))
	}
	for line := range strings.SplitSeq(said, "\n") {
		if !strings.HasPrefix(visible(line), "▌") {
			continue
		}
		if !strings.HasPrefix(line, "\x1b[32;48;5;237m▌") {
			t.Errorf("box row = %q, want the success box's green SGR spine", line)
		}
		if len([]rune(visible(line))) != 80 {
			t.Errorf("box row = %q, want a full 80-cell pane", line)
		}
	}
}

// A failed command still shows its output in a box, but the left border turns
// red so the eye lands on the call that went wrong.
func TestAFailedRunCommandRendersARedOutputBox(t *testing.T) {
	m := sized()
	m.absorb(called("f", "run_command", `{"command":"bad"}`))
	m.absorb(nacelle.Event{Kind: nacelle.KindToolResult, Tool: &nacelle.ToolEvent{
		ID: "f", Name: "run_command", Input: `{"command":"bad"}`, Result: "boom: lost the build", Err: errors.New("exit 1"),
	}})
	m.stranded()

	said := strings.Join(m.unprinted, "\n")
	if strings.Contains(said, "38;5;3") {
		t.Errorf("output = %q, want no ANSI256 grey-range border escape", said)
	}
	if !strings.Contains(visible(said), "boom: lost the build") {
		t.Errorf("output = %q, want the failed command's output in a box", visible(said))
	}
	for line := range strings.SplitSeq(said, "\n") {
		if strings.HasPrefix(visible(line), "▌") && !strings.HasPrefix(line, "\x1b[31;48;5;237m▌") {
			t.Errorf("box row = %q, want the failure box's red SGR spine", line)
		}
	}
}

// A finished edit draws its file diff in a box whose footer sums the change as
// "+x -y", additions counted and coloured separately from removals.
func TestAFinishedEditRendersARecapBox(t *testing.T) {
	m := sized()
	m.run.diffs = true
	m.absorb(called("e", "edit_file", `{"path":"view.go","old":"old","new":"new"}`))
	m.absorb(nacelle.Event{Kind: nacelle.KindToolResult, Tool: &nacelle.ToolEvent{
		ID: "e", Name: "edit_file", Input: `{"path":"view.go","old":"old","new":"new"}`,
	}})

	said := visible(strings.Join(m.unprinted, "\n"))
	if !strings.Contains(said, "view.go") {
		t.Errorf("diff = %q, want the changed file named in the box", said)
	}
	if !strings.Contains(said, "+1 -1") {
		t.Errorf("diff = %q, want the +1 -1 recap", said)
	}
}

// While an edit or command is still running, its live region row is the same
// box the result will fill, its left border and held line both wearing the
// running state's yellow (basic "3", SGR 33).
func TestTheLiveBoxIsYellowWhileRunning(t *testing.T) {
	m := sized()
	m.run.busy = true
	m.absorb(called("l", "edit_file", `{"path":"view.go"}`))

	view := m.View().Content
	if !strings.Contains(view, "\x1b[33;48;5;237m▌") {
		t.Errorf("view = %q, want the running edit box's yellow spine", view)
	}
	if !strings.Contains(view, "\x1b[33m⏺") {
		t.Errorf("view = %q, want the running glyph in that same yellow", view)
	}
	if strings.Contains(view, "38;5;3") {
		t.Errorf("view = %q, want no ANSI256 grey-range border escape", view)
	}
}

// A read call is not boxed while running — only edits and commands get the
// pane treatment, so the box means "this changed the tree" (its spine carries
// the 48;5;237 backdrop). The held line still wears the running yellow.
func TestANonEditToolIsNotBoxedWhileRunning(t *testing.T) {
	m := sized()
	m.run.busy = true
	m.absorb(called("r", "read_file", `{"path":"view.go"}`))

	if strings.Contains(m.View().Content, "48;5;237m▌") {
		t.Error("a read call was boxed, want the ordinary held line")
	}
	if !strings.Contains(m.View().Content, "\x1b[33m⏺") {
		t.Error("a read call's held line is not painted the running yellow")
	}
}

// A command that returns no output has nothing to box, so it stays the plain
// one-line report rather than drawing an empty pane.
func TestARunCommandWithNoOutputRendersNoBox(t *testing.T) {
	m := sized()
	m.absorb(called("x", "run_command", `{"command":"true"}`))
	m.absorb(nacelle.Event{Kind: nacelle.KindToolResult, Tool: &nacelle.ToolEvent{
		ID: "x", Name: "run_command", Input: `{"command":"true"}`, Result: "",
	}})

	if strings.Contains(visible(strings.Join(m.unprinted, "\n")), "▌") {
		t.Error("a command with no output still drew a box")
	}
}

// Streamed output from a running command fills the box live, made visible in
// the running tool's live region under the held line.
func TestAStreamingCommandFillsTheLiveBox(t *testing.T) {
	m := sized()
	m.run.busy = true
	m.absorb(called("s", "run_command", `{"command":"make"}`))
	m.absorb(nacelle.Event{Kind: nacelle.KindToolOutput, Tool: &nacelle.ToolEvent{ID: "s", Name: "run_command"}, Text: "compling...\n"})

	view := m.View().Content
	if !strings.Contains(visible(view), "compling...") {
		t.Errorf("view = %q, want the streamed line in the running box", visible(view))
	}
}

// Streamed output that filled the box wins over the result's copy of the same
// output, so a backend that streams never doubles the box when the result
// arrives carrying the whole output again.
func TestStreamedOutputIsNotDuplicatedAtResult(t *testing.T) {
	m := sized()
	m.absorb(called("d", "run_command", `{"command":"make"}`))
	m.absorb(nacelle.Event{Kind: nacelle.KindToolOutput, Tool: &nacelle.ToolEvent{ID: "d", Name: "run_command"}, Text: "go build\n"})
	m.absorb(nacelle.Event{Kind: nacelle.KindToolResult, Tool: &nacelle.ToolEvent{
		ID: "d", Name: "run_command", Input: `{"command":"make"}`, Result: "go build\n",
	}})

	said := visible(strings.Join(m.unprinted, "\n"))
	if !strings.Contains(said, "go build") {
		t.Errorf("output = %q, want the streamed line in the committed box", said)
	}
	if strings.Count(said, "go build") != 1 {
		t.Errorf("output = %q, want the streamed line once, not duplicated by the result", said)
	}
}

// A streamed fragment is one completed line without its trailing newline, so
// the live box must split the accumulated buffer per fragment and grow a row
// per line as it arrives. Concatenating the fragments raw squishes every line
// onto one.
func TestStreamedFragmentsFillTheLiveBoxOneRowEach(t *testing.T) {
	m := sized()
	m.run.busy = true
	m.absorb(called("s", "run_command", `{"command":"make"}`))
	for _, frag := range []string{"syntax ok", "compiling", "linking", "done"} {
		m.absorb(nacelle.Event{Kind: nacelle.KindToolOutput, Tool: &nacelle.ToolEvent{ID: "s", Name: "run_command"}, Text: frag})
	}

	view := visible(m.View().Content)
	for _, line := range []string{"▌ syntax ok", "▌ compiling", "▌ linking", "▌ done"} {
		if !strings.Contains(view, line) {
			t.Errorf("view = %q, want streamed line %q in the running box", view, line)
		}
	}
	rows := 0
	for line := range strings.SplitSeq(m.View().Content, "\n") {
		if strings.Contains(line, "48;5;237m▌") {
			rows++
		}
	}
	if rows != 1+4 {
		t.Errorf("view = %q, want the held line plus 4 streamed rows, got %d", view, rows)
	}
	if strings.Contains(view, "syntax okcompilinglinkingdone") {
		t.Errorf("view = %q, want no squished single row", view)
	}
}

// The same split must survive into the finished box: the streamed fragments
// win over the result's whole-output copy, so the committed pane is one row
// per line rather than one squished row.
func TestStreamedFragmentsStaySeparateRowsInTheFinishedBox(t *testing.T) {
	m := sized()
	m.absorb(called("f", "run_command", `{"command":"make"}`))
	m.absorb(nacelle.Event{Kind: nacelle.KindToolOutput, Tool: &nacelle.ToolEvent{ID: "f", Name: "run_command"}, Text: "syntax ok"})
	m.absorb(nacelle.Event{Kind: nacelle.KindToolOutput, Tool: &nacelle.ToolEvent{ID: "f", Name: "run_command"}, Text: "compiling"})
	m.absorb(nacelle.Event{Kind: nacelle.KindToolOutput, Tool: &nacelle.ToolEvent{ID: "f", Name: "run_command"}, Text: "linking"})
	m.absorb(nacelle.Event{Kind: nacelle.KindToolOutput, Tool: &nacelle.ToolEvent{ID: "f", Name: "run_command"}, Text: "done"})
	m.absorb(nacelle.Event{Kind: nacelle.KindToolResult, Tool: &nacelle.ToolEvent{
		ID: "f", Name: "run_command", Input: `{"command":"make"}`, Result: "syntax ok\ncompiling\nlinking\ndone\n",
	}})
	m.stranded()

	said := visible(strings.Join(m.unprinted, "\n"))
	rows := 0
	for line := range strings.SplitSeq(said, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "▌") {
			rows++
		}
	}
	if rows != 4 {
		t.Errorf("output = %q, want 4 finished box rows, got %d", said, rows)
	}
	raw := strings.Join(m.unprinted, "\n")
	if !strings.Contains(raw, "\x1b[32;48;5;237m▌") {
		t.Errorf("output = %q, want the finished streamed box's green spine", raw)
	}
	if strings.Contains(said, "syntax okcompilinglinkingdone") {
		t.Errorf("output = %q, want no squished single row", said)
	}
}

// A running command's live box border wears the same yellow as an edit's — one
// running colour for every tool — while a finished box keeps green or red.
func TestTheLiveBoxBorderIsYellowForRunCommandToo(t *testing.T) {
	m := sized()
	m.run.busy = true
	m.absorb(called("c", "run_command", `{"command":"make"}`))

	view := m.View().Content
	if !strings.Contains(view, "\x1b[33;48;5;237m▌") {
		t.Errorf("view = %q, want the running command box's yellow spine", view)
	}
	if strings.Contains(view, "38;5;3") {
		t.Errorf("view = %q, want no ANSI256 grey-range border escape", view)
	}
}
