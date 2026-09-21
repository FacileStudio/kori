// Package tui — how a compaction pass presents itself while it runs: the
// spinner must keep ticking, and the running-row elapsed timer with it, even
// on the idle and /compact paths where run.busy stays false; and a pass that
// gets no summary back must say so rather than leave only a weak mask card.
package tui

import (
	"context"
	"strings"
	"testing"

	"charm.land/bubbles/v2/spinner"
)

func TestSpinnerKeepsTickingWhileCompacting(t *testing.T) {
	m := sized()
	m.conversation = bigConversation()
	m.compacting = true
	msg, ok := m.spin.Tick().(spinner.TickMsg)
	if !ok {
		t.Fatal("Tick did not produce a spinner.TickMsg")
	}
	if cmd := m.spun(msg); cmd == nil {
		t.Error("the spinner stopped re-arming during a compaction pass (the timer would freeze)")
	}
}

func TestCompactionArmsTheSpinnerAndStampsTheTimer(t *testing.T) {
	m := sized()
	m.conversation = bigConversation()
	if cmd := m.beginCompaction(context.Background()); cmd == nil {
		t.Fatal("beginCompaction returned no command, want the spinner-armed wait")
	}
	defer func() { m.compacting = false }()
	if !m.compacting {
		t.Error("beginCompaction did not mark the session as compacting")
	}
	if m.compactBegan.IsZero() {
		t.Error("beginCompaction did not stamp when the pass began")
	}
	if row := strings.Join(m.inFlightGroups(), "\n"); !strings.Contains(row, "compacting session") {
		t.Errorf("in-flight row = %q, want the compaction row drawn live", row)
	} else if !strings.Contains(row, " · ") {
		t.Errorf("in-flight row = %q, want the elapsed timer after the spinner", row)
	}
}

func TestSettleCompactionSaysWhenTheSummaryCameBackEmpty(t *testing.T) {
	m := sized()
	m.conversation = bigConversation()
	outcome := compactOutcome{before: int64(125_000), plan: m.plan()}
	m.settleCompaction(outcome)
	if m.compacting {
		t.Errorf("compacting still true after the empty fallback")
	}
	joined := strings.Join(spoken(m), "\n")
	if !strings.Contains(joined, "compaction summary came back empty") {
		t.Errorf("spoken = %v, want the empty-summary note rather than a silent fallback", spoken(m))
	}
}
