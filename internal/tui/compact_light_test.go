package tui

// Tests for the light lever, the tier dispatch and the thrash guard.

import (
	"context"
	"strings"
	"testing"

	"github.com/FacileStudio/kori/internal/compaction"
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
