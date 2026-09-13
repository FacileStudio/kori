package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/FacileStudio/nacelle"
)

// A tool's line is held until its result so the duration can fold into it, and
// the status line names the tool while it is in flight — but the line itself
// still has to reach the screen the moment it is said, ahead of the wait for
// whatever comes next. Update flushing the queue before the blocking command
// is what guarantees that, and it is the half of this that is still load
// bearing.
// The closed channel is not scenery: reading what consume printed also runs
// the wait it re-armed, and a nil one never sends.
func TestAToolLineIsPrintedAheadOfTheWaitForTheNextEvent(t *testing.T) {
	m := sized()
	m.run.busy = true

	ended := make(chan result)
	close(ended)
	m.run.results = ended

	m.Update(result{event: nacelle.Event{
		Kind: nacelle.KindToolCall,
		Tool: &nacelle.ToolEvent{ID: "1", Name: "run_command"},
	}})
	_, cmd := m.Update(result{event: nacelle.Event{
		Kind: nacelle.KindToolResult,
		Tool: &nacelle.ToolEvent{ID: "1", Name: "run_command"},
	}})

	if printed := printedBy(cmd); !strings.Contains(printed, "run_command") {
		t.Errorf("printed = %q, want the line said before the wait for the next event", printed)
	}
}

// The question has to reach the screen when it is asked, not when it is
// answered. waitFor blocks until the model's first token, a batch is not
// finished until every command in it is, and Update sequences the print queue
// behind whatever the routed message returned — so the echo sat there until
// the reply arrived. What that looked like was the prompt emptying and nothing
// else happening, which reads as a client that swallowed the question.
func TestTheQuestionIsPrintedWithoutWaitingOnTheAnswer(t *testing.T) {
	m := sized()
	m.agent = answering(t)
	m.prompt.SetValue("a question")

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	printed := printedBy(cmd)
	defer m.run.cancel()

	if !strings.Contains(printed, "a question") {
		t.Errorf("printed = %q, want the question handed over before the run is waited on", printed)
	}
}

func TestKindTurnRendersBoundaryWithoutBreakingConversation(t *testing.T) {
	m := sized()
	m.run.busy = true
	m.run.cancel = func() {}
	m.run.turnBegan = time.Now().Add(-1500 * time.Millisecond)

	m.absorb(nacelle.Event{Kind: nacelle.KindText, Text: "the answer\n"})
	m.record(nacelle.Event{Kind: nacelle.KindText, Text: "the answer\n"})
	m.absorb(nacelle.Event{Kind: nacelle.KindTurn, Usage: nacelle.Usage{InputTokens: 100, OutputTokens: 50, Cost: 0.0025}})
	m.absorb(nacelle.Event{Kind: nacelle.KindDone, Stop: nacelle.StopEnd})
	m.settle()

	saidLines := spoken(m)
	if len(saidLines) != 2 || !strings.Contains(saidLines[0], "the answer") {
		t.Fatalf("spoken = %v, want answer and run recap", saidLines)
	}
	if !strings.Contains(saidLines[1], "150 tokens") || !strings.Contains(saidLines[1], "$0.0025") {
		t.Errorf("run recap = %q, want tokens and cost", saidLines[1])
	}
	raw := m.unprinted[len(m.unprinted)-1]
	if !strings.Contains(raw, "\x1b[38;5;2") {
		t.Errorf("turn boundary raw = %q, want muted style", raw)
	}
	if len(m.conversation) != 1 || said(m.conversation[0]) != "the answer\n" {
		t.Errorf("conversation = %v, want full answer preserved", m.conversation)
	}
}

// The run recap is said once, at settle: the run's duration, tokens and cost
// on one muted line, replacing the per-turn boundary lines that used to land
// after every turn.
func TestWidgetLinesAreSeparatedByBlankRows(t *testing.T) {
	m := sized()
	m.run.answer.WriteString("the answer")
	m.turn(nacelle.Event{Usage: nacelle.Usage{InputTokens: 10}})
	m.say(fromTool, "$ read_file(x)")

	said := visible(strings.Join(m.unprinted, "\n\n"))
	if strings.Contains(said, "10 tokens") {
		t.Fatalf("said = %q, want no per-turn boundary in the transcript", said)
	}
	m.settle()
	said = visible(strings.Join(m.unprinted, "\n\n"))
	recapAt := strings.Index(said, "10 tokens")
	toolAt := strings.Index(said, "$ read_file(x)")
	if recapAt < 0 || toolAt < 0 || toolAt >= recapAt {
		t.Fatalf("said = %q, want the tool line before the run recap", said)
	}
	if gap := said[toolAt:recapAt]; strings.Count(gap, "\n") < 2 {
		t.Errorf("said = %q, want a blank row between the tool line and the run recap", said)
	}
}

// The run recap closes the answer it follows, so it must be held apart from
// that answer by a blank row — a recap gluing straight to the answer reads as
// the timing being part of the answer.
func TestRunRecapIsSeparatedFromTheAnswerAboveIt(t *testing.T) {
	m := sized()
	m.run.busy = true
	m.run.cancel = func() {}
	m.run.answer.WriteString("Code committed")
	m.turn(nacelle.Event{Usage: nacelle.Usage{InputTokens: 10}})
	m.settle()

	said := visible(strings.Join(m.unprinted, "\n\n"))
	recapAt := strings.Index(said, "10 tokens")
	answerAt := strings.Index(said, "Code committed")
	if recapAt < 0 || answerAt < 0 || answerAt >= recapAt {
		t.Fatalf("said = %q, want the answer before the recap", said)
	}
	if gap := said[answerAt:recapAt]; strings.Count(gap, "\n") < 2 {
		t.Errorf("said = %q, want a blank row between the answer and the recap", said)
	}
}

// The reader's question is a block of its own: it sits a blank row under the
// thinking trace and a blank row over whatever follows, the same breathing
// room a turn boundary gets, so it reads as the thing you scrolled up to find
// rather than a question glued to the tool call it prompted.
func TestReaderQuestionIsSeparatedFromTheTraceAroundItByBlankRows(t *testing.T) {
	m := sized()
	m.say(fromThinking, "thinking out loud")
	m.say(fromReader, "a question")
	m.say(fromTool, "$ read_file(x)")

	said := visible(strings.Join(m.unprinted, "\n\n"))
	thinkingAt := strings.Index(said, "thinking out loud")
	questionAt := strings.Index(said, "a question")
	toolAt := strings.Index(said, "$ read_file(x)")
	if thinkingAt < 0 || questionAt < 0 || toolAt < 0 {
		t.Fatalf("said = %q, want thinking, question, and tool line", said)
	}
	if above := said[thinkingAt:questionAt]; strings.Count(above, "\n") < 2 {
		t.Errorf("said = %q, want a blank row between the thinking trace and the question", said)
	}
	if below := said[questionAt:toolAt]; strings.Count(below, "\n") < 2 {
		t.Errorf("said = %q, want a blank row between the question and the tool line", said)
	}
}
