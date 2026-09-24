package tui

import (
	"testing"

	"github.com/FacileStudio/nacelle"
)

// TestATurnIsReportedOncePerModelCall pins the turn counter the editor shows.
// nacelle owns the tool loop, so a turn is inferred from what the stream does
// rather than announced: KindTurn ends one model call, the next streaming event
// proves another is under way, and a run that ends instead must not have a turn
// invented for it.
func TestATurnIsReportedOncePerModelCall(t *testing.T) {
	surface := &fakeSurface{}
	m := attached(surface)

	m.absorb(nacelle.Event{Kind: nacelle.KindText, Text: "one"})
	m.absorb(nacelle.Event{Kind: nacelle.KindThinking, Text: "still the same call"})
	m.absorb(nacelle.Event{Kind: nacelle.KindTurn})
	m.ideTurn(nacelle.KindToolResult)
	m.absorb(nacelle.Event{Kind: nacelle.KindText, Text: "two"})
	m.absorb(nacelle.Event{Kind: nacelle.KindTurn})
	m.ideTurn(nacelle.KindDone)

	want := []int{1, 2}
	if len(surface.turns) != len(want) {
		t.Fatalf("turns = %v, want %v", surface.turns, want)
	}
	for i, n := range want {
		if surface.turns[i] != n {
			t.Errorf("turn %d = %d, want %d", i, surface.turns[i], n)
		}
	}
}

// TestATurnAfterTheLastOneIsNotInvented pins the other half: a run whose model
// stopped asking for tools ends, and the events that end it are not a new call.
func TestATurnAfterTheLastOneIsNotInvented(t *testing.T) {
	surface := &fakeSurface{}
	m := attached(surface)

	m.absorb(nacelle.Event{Kind: nacelle.KindText, Text: "done"})
	m.absorb(nacelle.Event{Kind: nacelle.KindTurn})
	m.absorb(nacelle.Event{Kind: nacelle.KindDone})

	if len(surface.turns) != 1 {
		t.Errorf("turns = %v, want the one call that happened", surface.turns)
	}
}

// TestDoneNamesWhyTheRunEnded pins the four words the protocol carries against
// the three ways a run actually stops, plus the one that is not a stop at all:
// a run a provider error ended keeps the last turn's stop, and calling that a
// finished answer would tell the editor a broken run was a complete one.
func TestDoneNamesWhyTheRunEnded(t *testing.T) {
	for _, tc := range []struct {
		name   string
		stop   nacelle.Stop
		failed bool
		want   string
	}{
		{"the model finished", nacelle.StopEnd, false, "end_turn"},
		{"nothing was recorded", "", false, "end_turn"},
		{"esc", abandoned, false, "cancelled"},
		{"the iteration budget", nacelle.StopIterations, false, "max_iterations"},
		{"a provider error", nacelle.StopEnd, true, "error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			surface := &fakeSurface{}
			m := attached(surface)
			m.run.stop, m.ide.failed = tc.stop, tc.failed
			m.run.usage = nacelle.Usage{Cost: 0.0123}

			m.ideDone()

			if len(surface.reasons) != 1 || surface.reasons[0] != tc.want {
				t.Fatalf("reasons = %v, want %q", surface.reasons, tc.want)
			}
			if surface.cost != 0.0123 {
				t.Errorf("cost = %v, want the run's own spend", surface.cost)
			}
		})
	}
}

// TestASessionWithNoEditorReportsNothing pins that the run's shape costs a
// session with no editor nothing at all, which is what lets these calls sit in
// the streaming path rather than behind a check at every caller.
func TestASessionWithNoEditorReportsNothing(t *testing.T) {
	m := sized()

	m.ideTurn(nacelle.KindText)
	m.ideDone()

	if m.ide.turns != 0 || m.ide.open {
		t.Errorf("turns = %d open = %t, want the no-editor path to count nothing", m.ide.turns, m.ide.open)
	}
}
