// Package tui — the explicit triggers of context compaction: the automatic
// post-turn check and the manual /compact command.
package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/FacileStudio/kori/internal/compaction"
)

// shouldCompactIdle is the decision behind maybeCompactIdle, split out so a
// test can drive it without running a pass. It asks the questions that have
// stable checkable answers: is no pass already running, is nothing queued that
// will start a run, and is the conversation over the trigger. The absolute
// compact_at is checked before the tier trigger on purpose: an explicit 0 turns
// compaction off, while Policy.Trigger still reports a ratio for the window it
// was resolved against, so the guard is what honours the disable.
func (m *Model) shouldCompactIdle() bool {
	if m.compacting || m.nextToSend() >= 0 || m.thrashed() {
		return false
	}
	return m.compactAt > 0 && m.size > m.policy.Trigger()
}

// maybeCompactIdle triggers a compaction pass after a run ends, when the model
// is idle and the conversation was left over the threshold, so a turn that
// finished too heavy is not left sitting at an absurd size while the reader
// waits to type. It is the post-turn half of the trigger: send compacts before
// a run that needs the freed context, and this compacts after a turn that grew
// too big — beginning no run, because the pass reports through
// settleCompaction, which starts nothing while the model is idle.
//
// It only fires when nothing is queued, so it cannot race a run about to
// start: deliver hands any queued line to send, whose own pre-flight check sees
// the same context. While the pass runs the update loop waits on it exactly as
// the send path does, so a message typed meanwhile is queued, not sent into a
// compacting middle.
func (m *Model) maybeCompactIdle() tea.Cmd {
	if !m.shouldCompactIdle() {
		return nil
	}
	return m.compactTiered(context.Background())
}

// compactCmd is the manual /compact: run a compaction pass now, on demand,
// whether or not the conversation has crossed the automatic threshold. It is
// the explicit half of the trigger, for when the reader wants the context
// reclaimed at a natural break rather than waiting to overshoot. A pass is an
// LLM call, so it must not race a live run — the command only fires while
// idle, and while the pass runs the reader's next keystroke queues behind it
// (ask treats an in-flight pass like busy). A fresh manual pass clears any
// thrash flag, because asking by hand is a positive re-attempt.
func (m *Model) compactCmd() tea.Cmd {
	if m.compactAt <= 0 {
		m.say(fromClient, "compaction is off — limits.compact_at is 0")
		return nil
	}
	if m.run.busy {
		m.say(fromClient, "finish the current run, then /compact")
		return nil
	}
	if m.compacting {
		m.say(fromClient, "already compacting")
		return nil
	}
	if _, _, ok := compaction.HistoryRange(m.plan()); !ok {
		m.say(fromClient, "nothing to compact — the conversation is too short")
		return nil
	}
	m.thrashCount = 0
	return m.beginCompaction(context.Background())
}
