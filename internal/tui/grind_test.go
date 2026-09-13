package tui

// The grind budget's gating: a run that stops short of its minimum spend on a
// stop the model chose is continued with one notice, the cap bounds how many
// times that happens, and a session with no budget behaves exactly as it did
// before. The continuation is driven through settle on a real agent built
// over an idle stub backend, so send's own path (and the notice it appends to
// the conversation) is exercised without a network.

import (
	"context"
	"iter"
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

// idle is a backend whose runs end immediately: the shape of a continuation
// the test never has to pump, because the stream closes before anyone reads
// it.
type idle struct{}

func (idle) Name() string                       { return "idle" }
func (idle) Capabilities() nacelle.Capabilities { return nacelle.Capabilities{} }

func (idle) CountTokens(context.Context, nacelle.Request) (int64, error) { return 0, nil }

func (idle) Stream(context.Context, nacelle.Request) iter.Seq2[nacelle.Event, error] {
	return func(yield func(nacelle.Event, error) bool) {}
}

func grindAgent(t *testing.T) *nacelle.Agent {
	t.Helper()
	agent, err := nacelle.New(nacelle.Config{Backend: idle{}, System: "grind"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return agent
}

// stoppedBelowTheMinimum stages a run that has just ended: the final stop on
// the run and a spend under any floor a test sets.
func stoppedBelowTheMinimum(m *Model, stop nacelle.Stop) {
	m.run.busy = true
	m.run.cancel = func() {}
	m.run.stop = stop
	m.run.usage = nacelle.Usage{OutputTokens: 100, Cost: 0.01}
}

func lastUserText(t *testing.T, m *Model) string {
	t.Helper()
	last := m.conversation[len(m.conversation)-1]
	var said []string
	for _, part := range last.Parts {
		if text, ok := part.(nacelle.Text); ok {
			said = append(said, text.Text)
		}
	}
	return strings.Join(said, " ")
}

func TestGrindContinuesARunBelowTheMinimum(t *testing.T) {
	m := sized()
	m.agent = grindAgent(t)
	m.grind = grindBudget{tokens: 10_000, cap: 2}
	m.conversation = append(m.conversation, nacelle.UserText("do the task"))
	stoppedBelowTheMinimum(m, nacelle.StopEnd)

	cmd := m.settle()

	if cmd == nil {
		t.Fatal("settle returned no cmd, want the continuation run started")
	}
	if !m.run.busy {
		t.Error("run not busy after settle, want the continuation running")
	}
	if m.grind.used != 1 {
		t.Errorf("continuations used = %d, want 1", m.grind.used)
	}
	if len(m.conversation) != 3 {
		t.Fatalf("conversation length = %d, want question, finished turn, notice", len(m.conversation))
	}
	if notice := lastUserText(t, m); !strings.Contains(notice, "grind budget") {
		t.Errorf("continuation = %q, want the grind notice on the conversation", notice)
	}
	if said := strings.Join(spoken(m), "\n"); !strings.Contains(said, "grind budget") {
		t.Errorf("transcript = %q, want the notice shown to the reader", said)
	}
}

func TestGrindRespectsItsCap(t *testing.T) {
	m := sized()
	m.agent = grindAgent(t)
	m.grind = grindBudget{tokens: 10_000, cap: 1}
	m.conversation = append(m.conversation, nacelle.UserText("do the task"))
	stoppedBelowTheMinimum(m, nacelle.StopEnd)
	m.settle()

	stoppedBelowTheMinimum(m, nacelle.StopEnd)
	m.settle()

	if m.run.busy {
		t.Error("run busy after the second settle, want the cap to end the grinding")
	}
	if m.grind.used != 1 {
		t.Errorf("continuations used = %d, want the cap of 1 to hold", m.grind.used)
	}
	if len(m.conversation) != 4 {
		t.Errorf("conversation length = %d, want no second notice past the cap", len(m.conversation))
	}
}

func TestNoGrindBudgetChangesNothing(t *testing.T) {
	m := sized()
	m.agent = grindAgent(t)
	m.conversation = append(m.conversation, nacelle.UserText("do the task"))
	stoppedBelowTheMinimum(m, nacelle.StopEnd)

	m.settle()

	if m.run.busy {
		t.Error("run busy after settle, want no continuation without a budget")
	}
	if m.grind.used != 0 {
		t.Errorf("continuations used = %d, want 0", m.grind.used)
	}
	if len(m.conversation) != 2 {
		t.Errorf("conversation length = %d, want only the question and the finished turn", len(m.conversation))
	}
	if said := strings.Join(spoken(m), "\n"); strings.Contains(said, "grind") {
		t.Errorf("transcript = %q, want no grind notice", said)
	}
}

func TestGrindOnlyContinuesStopsTheModelChose(t *testing.T) {
	for _, tc := range []struct {
		name string
		stop nacelle.Stop
		want bool
	}{
		{"finished", nacelle.StopEnd, true},
		{"refused", nacelle.StopRefusal, true},
		{"cut off at the token limit", nacelle.StopMaxTokens, true},
		{"out of context", nacelle.StopContext, false},
		{"iteration limit", nacelle.StopIterations, false},
		{"tools never ends a run", nacelle.StopTools, false},
		{"unknown", nacelle.StopOther, false},
		{"errored", "", false},
		{"abandoned by the reader", abandoned, false},
	} {
		m := sized()
		m.grind = grindBudget{cost: 1, cap: 2}
		if got := m.grind.owed(tc.stop, nacelle.Usage{}); got != tc.want {
			t.Errorf("%s: owed = %t, want %t", tc.name, got, tc.want)
		}
	}
}

func TestGrindLeavesARunThatSpentTheMinimum(t *testing.T) {
	m := sized()
	m.grind = grindBudget{cost: 0.5, cap: 2}

	if m.grind.owed(nacelle.StopEnd, nacelle.Usage{Cost: 0.5}) {
		t.Error("a run that met the floor still owed a continuation")
	}
	if !m.grind.owed(nacelle.StopEnd, nacelle.Usage{Cost: 0.25}) {
		t.Error("a run under the floor was not owed one")
	}
}

func TestGrindWithBothFloorsWantsBoth(t *testing.T) {
	g := grindBudget{cost: 0.5, tokens: 10_000, cap: 2}

	if !g.owed(nacelle.StopEnd, nacelle.Usage{Cost: 0.6}) {
		t.Error("the cost floor alone satisfied both")
	}
	if !g.owed(nacelle.StopEnd, nacelle.Usage{OutputTokens: 11_000}) {
		t.Error("the token floor alone satisfied both")
	}
	if g.owed(nacelle.StopEnd, nacelle.Usage{Cost: 0.6, OutputTokens: 11_000}) {
		t.Error("a run over both floors was owed a continuation")
	}
}

func TestTheFooterShowsTheRemainingBudgetOnlyWhenOneIsSet(t *testing.T) {
	m := sized()
	m.grind = grindBudget{cost: 0.5, cap: 2}
	if foot := strings.Join(m.footer(), " "); !strings.Contains(foot, "grind $0.50") {
		t.Errorf("footer = %q, want the full budget as remaining", foot)
	}

	m.run.usage = nacelle.Usage{Cost: 0.2}
	if foot := strings.Join(m.footer(), " "); !strings.Contains(foot, "grind $0.30") {
		t.Errorf("footer = %q, want the run's spend subtracted", foot)
	}

	m.run.usage = nacelle.Usage{Cost: 0.6}
	if foot := strings.Join(m.footer(), " "); strings.Contains(foot, "grind") {
		t.Errorf("footer = %q, want the grind label gone once the floor is met", foot)
	}

	m.grind = grindBudget{}
	if foot := strings.Join(m.footer(), " "); strings.Contains(foot, "grind") {
		t.Errorf("footer = %q, want no grind label without a budget", foot)
	}
}

func TestADispatchedPromptResetsTheGrindCounter(t *testing.T) {
	m := sized()
	m.agent = grindAgent(t)
	m.grind = grindBudget{tokens: 10_000, cap: 2, used: 2}

	m.dispatch("next task")

	if m.grind.used != 0 {
		t.Errorf("continuations used = %d, want the fresh prompt to reset the counter", m.grind.used)
	}
}
