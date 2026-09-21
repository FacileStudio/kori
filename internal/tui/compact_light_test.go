package tui

// Tests for the light lever, the tier dispatch and the thrash guard.

import (
	"context"
	"strings"
	"testing"

	"github.com/FacileStudio/kori/internal/compaction"
	"github.com/FacileStudio/nacelle"
)

// windowedPolicy is a policy whose ratios can be measured: soft at 130k, mid at
// 160k and hard at 180k of a 200k window, with compact_at pinned at 100k.
func windowedPolicy() compaction.Policy {
	return compaction.Policy{
		Ratios:         compaction.Ratios{Soft: 0.65, Mid: 0.80, Hard: 0.90},
		Window:         200_000,
		Ceiling:        100_000,
		KeepTurns:      3,
		AnchorMessages: 1,
	}
}

func TestEvictionCanLandUnderRequiresTheHistoryToMatter(t *testing.T) {
	m := sized()
	m.conversation = bigConversation()
	start, end, _ := compaction.HistoryRange(m.plan())

	m.size = int64(1_000_000)
	if m.evictionCanLandUnder(start, end) {
		t.Error("evictionCanLandUnder = true, want false when the pinned ends alone overshoot the ceiling")
	}

	m.size = int64(105_000)
	if !m.evictionCanLandUnder(start, end) {
		t.Error("evictionCanLandUnder = false, want true when folding the history lands a 105k conversation under the ceiling")
	}
}

func TestBeginCompactionSkipsTheSummarizerWhenEvictionCannotLandUnder(t *testing.T) {
	m := sized()
	m.conversation = bigConversation()
	m.size = int64(1_000_000)

	if cmd := m.beginCompaction(context.Background()); cmd != nil {
		t.Error("beginCompaction = a Cmd, want nil when the history cannot land under the ceiling — no summarizer call")
	}
	if m.compacting {
		t.Error("compacting = true, want no summarizer goroutine spawned")
	}
	if said := strings.Join(spoken(m), "\n"); !strings.Contains(said, "kept tail") {
		t.Errorf("report = %q, want the kept-tail explanation on a skipped pass", said)
	}
}

func TestMaskOnlyPassReportsTheKeptTailAndCountsTowardThrash(t *testing.T) {
	m := sized()
	m.conversation = bigConversation()
	m.size = int64(1_000_000)

	cmd := m.maskOnlyPass(m.plan())

	if cmd != nil {
		t.Error("maskOnlyPass = a Cmd, want nil")
	}
	if m.thrashCount != 1 {
		t.Errorf("thrashCount = %d, want 1 after a skip left the context over the threshold", m.thrashCount)
	}
	if said := strings.Join(spoken(m), "\n"); !strings.Contains(said, "kept tail") {
		t.Errorf("report = %q, want the kept-tail explanation", said)
	}
}

// The soft tier is free: it tombstones history and returns with no goroutine and
// no model call.
func TestSoftTierTombstonesWithoutAModelCall(t *testing.T) {
	m := sized()
	m.conversation = bigConversation()
	m.policy = windowedPolicy()
	m.size = 140_000

	cmd := m.compactTiered(context.Background())

	if cmd != nil {
		t.Error("compactTiered = a Cmd at the soft tier, want nil so no pass waits")
	}
	if m.compacting {
		t.Error("compacting = true, want no goroutine at the soft tier")
	}
	if m.trimmed == 0 {
		t.Error("trimmed = 0, want the soft tier to have tombstoned the history")
	}
	if said := strings.Join(spoken(m), " "); strings.Contains(said, "summarized") {
		t.Errorf("report = %q, want no summary on a soft pass", said)
	}
}

// The soft tier is free to run but not free of consequences: clearing a history
// result rewrites the messages after it, which is the prefix the provider had
// already cached. Below the clear floor the pass would trade that re-write for a
// few hundred bytes, so it stays quiet instead of trimming on a marginal
// overshoot.
func TestSoftTierSkipsAPassBelowTheClearFloor(t *testing.T) {
	m := sized()
	m.policy = windowedPolicy()
	m.size = 140_000
	m.conversation = []nacelle.Message{
		nacelle.UserText("the task"),
		{Role: nacelle.RoleUser, Parts: []nacelle.Part{
			nacelle.ToolResult{ID: "small", Name: "read", Result: strings.Repeat("x", compaction.MinCleared-1)},
		}},
		nacelle.AssistantText("an answer"),
		nacelle.UserText("a newer turn"),
		nacelle.AssistantText("the newest turn"),
	}

	cmd := m.compactTiered(context.Background())

	if cmd != nil {
		t.Error("compactTiered = a Cmd, want nil")
	}
	if m.trimmed != 0 {
		t.Errorf("trimmed = %d, want nothing tombstoned under the clear floor", m.trimmed)
	}
	if m.size != 140_000 {
		t.Errorf("size = %d, want it untouched by a pass that did not run", m.size)
	}
	if said := strings.Join(spoken(m), " "); said != "" {
		t.Errorf("said = %q, want no report for a pass that did not run", said)
	}
}

// Just over the floor the same pass does run, so the gate is a floor and not a
// change of policy: the tier still trims as soon as trimming is worth it.
func TestSoftTierRunsOnceThePassClearsTheFloor(t *testing.T) {
	m := sized()
	m.policy = windowedPolicy()
	m.size = 140_000
	m.conversation = []nacelle.Message{
		nacelle.UserText("the task"),
		{Role: nacelle.RoleUser, Parts: []nacelle.Part{
			nacelle.ToolResult{ID: "big", Name: "read", Result: strings.Repeat("x", compaction.MinCleared)},
		}},
		nacelle.AssistantText("an answer"),
		nacelle.UserText("a newer turn"),
		nacelle.AssistantText("the newest turn"),
	}

	cmd := m.compactTiered(context.Background())

	if cmd != nil {
		t.Error("compactTiered = a Cmd, want nil so no pass waits")
	}
	if m.trimmed != 1 {
		t.Errorf("trimmed = %d, want the one result over the floor tombstoned", m.trimmed)
	}
}

func TestCheckThrashCountsNearMissesAndWarnsOnlyAtTheLimit(t *testing.T) {
	m := sized()
	m.size = m.compactAt + compactSlack + 1

	for i := range thrashLimit - 1 {
		m.checkThrash()
		if m.thrashed() {
			t.Errorf("thrashed after %d near-miss passes, want the guard to wait for %d", i+1, thrashLimit)
		}
		if said := strings.Join(spoken(m), " "); strings.Contains(said, "compaction keeps leaving") {
			t.Errorf("said = %q, want no stand-down warning before the limit", said)
		}
	}

	m.checkThrash()

	if !m.thrashed() {
		t.Errorf("thrashed = false after %d consecutive near-misses, want the guard stood down", thrashLimit)
	}
	if said := strings.Join(spoken(m), " "); !strings.Contains(said, "compaction keeps leaving") {
		t.Errorf("said = %q, want the stand-down warning at the limit", said)
	}
}

func TestCheckThrashResetsTheCounterWhenUnder(t *testing.T) {
	m := sized()
	m.thrashCount = thrashLimit - 1
	m.size = m.compactAt - 1

	m.checkThrash()

	if m.thrashCount != 0 {
		t.Errorf("thrashCount = %d after a pass landed under the threshold, want it reset", m.thrashCount)
	}
	if m.thrashed() {
		t.Error("thrashed = true after a pass landed under, want it cleared")
	}
}

// The pre-send guard's headroom is tuned to the ladder, and this is the
// arithmetic that ties them together. The smallest size the guard acts on —
// Trigger() + compactSlack + 1 — must already be past the mid ratio, so what it
// dispatches is a summarizing pass rather than a free tombstone that would free
// nothing on a context only a summary can shrink and leave the send to overshoot
// anyway. A compactSlack below (mid - soft) × the window inverts that: the guard
// would fire into the soft tier and become the do-nothing check it was fixed
// from being.
//
// It lives here, in the package that owns compactSlack, because it is the one
// place both halves are visible. The window is the 128k the OpenAI backend
// reports, which makes the ceiling ResolveBudget derives 0.65 × 128000 — the
// 83.2k the ladder is calibrated against.
func TestThePreSendGuardFiresIntoASummarizingTier(t *testing.T) {
	const window = 128_000

	policy := compaction.Policy{
		Ratios: compaction.Ratios{
			Soft: compaction.DefaultSoftRatio,
			Mid:  compaction.DefaultMidRatio,
			Hard: compaction.DefaultHardRatio,
		},
		Window:         window,
		Ceiling:        int64(compaction.DefaultSoftRatio * window),
		KeepTurns:      compaction.DefaultKeepTurns,
		AnchorMessages: compaction.DefaultAnchorMessages,
	}

	fires := policy.Trigger() + compactSlack + 1

	switch tier := policy.Tier(fires); tier {
	case compaction.Mid, compaction.Hard:
	default:
		t.Errorf("the guard acts from %d, tier = %s, want a summarizing tier — a soft pass there could not land the context under", fires, tier)
	}
}
