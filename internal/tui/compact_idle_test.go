package tui

// Tests for the post-turn compaction trigger: shouldCompactIdle decides, and
// maybeCompactIdle starts a pass. The decision logic is the testable seam — a
// pass is an async goroutine — so these drive the decision across the guards
// that keep it from racing a run or thrashing.

import (
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

func TestShouldCompactIdleTriggersWhenIdleAndOverThreshold(t *testing.T) {
	m := sized()
	m.conversation = bigConversation()
	m.size = compactAt + 25_000

	if !m.shouldCompactIdle() {
		t.Errorf("shouldCompactIdle = false, want true when idle and the conversation is over the threshold")
	}
}

func TestShouldCompactIdleSkipsUnderTheThreshold(t *testing.T) {
	m := sized()
	m.size = compactAt - 1

	if m.shouldCompactIdle() {
		t.Errorf("shouldCompactIdle = true under the threshold, want false")
	}
}

func TestShouldCompactIdleSkipsWhenCompactionIsAlreadyRunning(t *testing.T) {
	m := sized()
	m.size = compactAt + 25_000
	m.compacting = true

	if m.shouldCompactIdle() {
		t.Errorf("shouldCompactIdle = true while a pass is already running, want false")
	}
}

func TestShouldCompactIdleSkipsWhenAMessageIsQueued(t *testing.T) {
	m := sized()
	m.size = compactAt + 25_000
	m.Add("follow up typed during the run")

	if m.shouldCompactIdle() {
		t.Errorf("shouldCompactIdle = true with a queued send, want false — send pre-flights instead")
	}
}

func TestShouldCompactIdleSkipsWhenCompactionIsDisabled(t *testing.T) {
	m := sized()
	m.compactAt = 0
	m.size = 200_000

	if m.shouldCompactIdle() {
		t.Errorf("shouldCompactIdle = true with compaction disabled, want false")
	}
}

func TestShouldCompactIdleSkipsWhenThrashed(t *testing.T) {
	m := sized()
	m.size = compactAt + 25_000
	m.thrashCount = thrashLimit

	if m.shouldCompactIdle() {
		t.Errorf("shouldCompactIdle = true at the thrash limit, want the auto trigger backed off")
	}
	m.thrashCount = thrashLimit - 1
	if !m.shouldCompactIdle() {
		t.Errorf("shouldCompactIdle = false below the thrash limit, want one near-miss not to disable auto-compaction")
	}
}

func TestCompactCmdRefusesWhenDisabled(t *testing.T) {
	m := sized()
	m.compactAt = 0

	m.compactCmd()

	if said := strings.Join(spoken(m), " "); !strings.Contains(said, "compaction is disabled") {
		t.Errorf("said = %q, want the disabled notice", said)
	}
}

func TestCompactCmdRefusesWhileBusy(t *testing.T) {
	m := sized()
	m.conversation = bigConversation()
	m.run.busy = true

	m.compactCmd()

	if said := strings.Join(spoken(m), " "); !strings.Contains(said, "finish the current run") {
		t.Errorf("said = %q, want the busy refusal", said)
	}
}

func TestCompactCmdRefusesWhileAlreadyCompacting(t *testing.T) {
	m := sized()
	m.conversation = bigConversation()
	m.compacting = true

	m.compactCmd()

	if said := strings.Join(spoken(m), " "); !strings.Contains(said, "already compacting") {
		t.Errorf("said = %q, want the in-progress notice", said)
	}
}

func TestCompactCmdRefusesWhenTheConversationIsTooShort(t *testing.T) {
	m := sized()
	m.conversation = []nacelle.Message{
		{Role: nacelle.RoleUser},
		{Role: nacelle.RoleAssistant},
	}

	m.compactCmd()

	if said := strings.Join(spoken(m), " "); !strings.Contains(said, "nothing to compact") {
		t.Errorf("said = %q, want the too-short notice", said)
	}
}

// A message typed while a post-turn pass is in flight must queue, not dispatch
// into a conversation a pass is about to replace. ask only checks run.busy,
// which stays false on the idle path, so it has to treat an in-flight pass the
// same way — otherwise the typed line starts a run against the old middle.
func TestAskQueuesWhileACompactionIsInFlight(t *testing.T) {
	m := sized()
	m.agent = answering(t)
	m.compacting = true

	m.prompt.SetValue("typed during the pass")
	m.ask()

	if len(m.conversation) != 0 {
		t.Errorf("conversation = %d, want the typed message withheld while the pass runs", len(m.conversation))
	}
	if m.Len() != 1 {
		t.Errorf("queue = %d, want the typed message queued", m.Len())
	}
	if m.run.busy {
		t.Errorf("busy = true, want the model left idle so the pass finishes without a run")
	}
}

// A line queued during the pass must be sent against the rebuilt conversation
// once the outcome lands, or it is silently swallowed and the reader has to
// resend it. Delivery runs inside settleCompaction here, not chained after the
// pass in settle: a sequence executes on its own goroutine and would race the
// install, sending against the pre-compaction state or stranding the line.
func TestSettleCompactionDeliversLinesQueuedDuringThePass(t *testing.T) {
	m := sized()
	m.agent = answering(t)
	m.conversation = bigConversation()
	m.compacting = true

	m.prompt.SetValue("typed during the pass")
	m.ask()
	if m.Len() != 1 {
		t.Fatalf("queue = %d, want the typed line queued while the pass runs", m.Len())
	}

	outcome := compactOutcome{before: int64(125_000), plan: m.plan(), summary: "Decisions:\n- done."}
	m.settleCompaction(outcome)
	defer m.run.cancel()

	if m.compacting {
		t.Errorf("compacting still true after the outcome is installed")
	}
	if m.Len() != 0 {
		t.Errorf("queue = %d, want the queued line delivered", m.Len())
	}
	last := m.conversation[len(m.conversation)-1]
	if text, ok := last.Parts[0].(nacelle.Text); !ok || text.Text != "typed during the pass" {
		t.Errorf("last message = %v, want the queued line sent against the rebuilt conversation", last.Parts)
	}
}
