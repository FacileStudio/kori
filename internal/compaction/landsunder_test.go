package compaction

import (
	"fmt"
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

// This file covers the estimate the forcing decision is made on: LandsUnder,
// estimateRebuilt and ledgerBound. Fold.Forced itself lives in force_test.go.

// carriedLedgerSample is a conversation whose ledger carries a tool pair: the
// ledger message holds a ToolCall and the reply answering it sits in the ledger
// zone. It is the shape assembly leaves behind once it has absorbed a kept tool
// block, and the point of it here is that neither message is history — no fold
// and no prune can reach either, so both ride along verbatim on every pass.
func carriedLedgerSample() []nacelle.Message {
	ledger := BuildLedger("", "the state")
	ledger.Parts = append(ledger.Parts, nacelle.ToolCall{ID: "c1", Name: "read", Finished: true})
	return []nacelle.Message{
		nacelle.UserText("the task"),
		ledger,
		resultMessage("c1", "read", strings.Repeat("x", 40_000)),
		nacelle.UserText("u2"),
		nacelle.AssistantText("a3"),
		nacelle.UserText("u4"),
		nacelle.AssistantText("a5"),
	}
}

// carriedLedgerWithHistory is the shape that makes the carry omission visible: a
// ledger carrying a tool pair, plus enough foldable history that folding really
// does shrink the conversation. Without the history the pass is refused — nothing
// was dropped and the rebuilt ledger adds a sentinel — and a refused pass installs
// nothing, so the bound has nothing to cover and the omission hides.
func carriedLedgerWithHistory() []nacelle.Message {
	conv := carriedLedgerSample()
	for i := range 4 {
		conv = appendTurn(conv, fmt.Sprintf("h%d", i))
	}
	return conv
}

// The invariant the whole forcing decision rests on: estimateRebuilt is a bound
// on what Apply actually assembles. LandsUnder is answered off it, and answering
// "does this fold land" off an underestimate reports a fold landing when it does
// not, which is the unsafe direction.
//
// This is the test the carry omission needed. The ledger zone holds messages no
// fold can reach, so a term left out of the estimate is invisible to every other
// test in this package while it silently makes the bound wrong.
func TestEstimateRebuiltBoundsWhatApplyAssembles(t *testing.T) {
	tests := map[string][]nacelle.Message{
		"a plain history":               judgeSample(),
		"a carried ledger":              carriedLedgerSample(),
		"a carried ledger with history": carriedLedgerWithHistory(),
		"a heavy absorbed":              absorbedPairSample(),
		"a single turn": {
			nacelle.UserText("the task"),
			nacelle.AssistantText("done"),
		},
	}
	for name, conv := range tests {
		t.Run(name, func(t *testing.T) {
			plan := Plan(conv, Policy{AnchorMessages: 1, KeepTurns: 1})
			gentle := Fold{Kept: Blocks(conv, plan)}
			assertEstimateBounds(t, conv, plan, gentle)
			assertEstimateBounds(t, conv, plan, gentle.Forced())
		})
	}
}

// assertEstimateBounds runs one fold through the real assembly and checks the
// estimate covered it. A refused or stale pass is skipped: neither installs a
// conversation, so there is nothing for the bound to cover.
func assertEstimateBounds(t *testing.T, conv []nacelle.Message, plan []Span, fold Fold) {
	t.Helper()
	out, stats := Apply(conv, plan, "a summary", fold.Survives, false)
	if stats.Stale || stats.Refused {
		return
	}
	if est, real := estimateRebuilt(conv, plan, fold), Bytes(out); est < real {
		t.Errorf("estimate = %d bytes, want at least the %d Apply assembled", est, real)
	}
}

// The bound has to cover the *merged* body, not just the one in place. The
// default fold merges, and MergeLedger is monotone: the new body keeps every line
// the old one had and adds the summary's. Bounding the ledger by the previous
// body alone under-counts by up to a summary, which is the direction that makes
// LandsUnder report a landing the merge had already undone.
//
// The fixture is a ledger already near its budget, because that is where the gap
// is widest: with a small body the summary floor covers the merge by accident,
// and only a substantial body shows the difference between "what is there" and
// "what is there plus what is about to be added".
func TestEstimateRebuiltLeavesRoomForTheMerge(t *testing.T) {
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		BuildLedger("", strings.Repeat("a recorded decision\n", 300)),
		nacelle.UserText("u2"),
		nacelle.AssistantText("a3"),
	}
	plan := Plan(conv, Policy{AnchorMessages: 1, KeepTurns: 1})

	previous := LedgerText(conv, plan)
	if len(previous) < MaxLedgerTokens*bytesPerToken/2 {
		t.Fatalf("the fixture ledger is %d bytes, want a substantial one for the merge to grow", len(previous))
	}
	grown := previous + strings.Repeat("x", MaxLedgerTokens*bytesPerToken)

	if bound, real := ledgerBound(conv, plan), MsgBytes(newLedgerMessage(grown)); bound < real {
		t.Errorf("ledgerBound = %d bytes, want at least the %d a merged body weighs", bound, real)
	}
}

// LandsUnder answers off a lower bound, so the boundary is deliberately strict:
// an estimate that lands exactly on the trigger does not count as landing, which
// leaves the margin a real rebuild needs. Pinning the operator keeps a later
// reading of the estimate from quietly making the comparison inclusive.
func TestLandsUnderIsStrictAboutTheBoundary(t *testing.T) {
	conv := judgeSample()
	plan := judgePlan(conv)
	fold := Fold{Kept: Blocks(conv, plan)}
	est := EstTokens(estimateRebuilt(conv, plan, fold))

	if est <= 0 {
		t.Fatalf("estimate = %d, want a positive figure to sit on", est)
	}
	if LandsUnder(conv, plan, fold, est) {
		t.Errorf("a fold estimated at exactly the trigger %d reported it lands, want the strict comparison", est)
	}
	if !LandsUnder(conv, plan, fold, est+1) {
		t.Errorf("a fold estimated at %d reported it does not land under %d", est, est+1)
	}
}

// LandsUnder answers the question the forced fold is derived from: would keeping
// what the judge kept leave the conversation under its trigger? A fold that keeps
// a history larger than the trigger does not land; the same fold forced does,
// because forcing folds that history away rather than keeping it.
func TestLandsUnderTellsAGentleFoldFromAForcedOne(t *testing.T) {
	conv := judgeSample()
	plan := judgePlan(conv)
	keeps := Fold{Kept: Blocks(conv, plan)}
	trigger := EstTokens(Bytes(conv)) / 2

	if LandsUnder(conv, plan, keeps, trigger) {
		t.Error("a fold keeping the whole history reported it lands under half its own size")
	}
	if !LandsUnder(conv, plan, keeps.Forced(), trigger) {
		t.Error("the forced fold reported it does not land, want the history folded away to land")
	}
	if !LandsUnder(conv, plan, keeps, 0) {
		t.Error("a session with no trigger reported it does not land, want the question to be moot")
	}
}

// The reserve is capped at 64k, so above a 320k window it stops growing and the
// ratios drift toward their raw fractions: the old top rung sat at 72% of the raw
// window at 200k but 84% at 1M, later than the 70–75% the field converges on for
// a forced fold. The decision is arithmetic now, so what a fold is measured
// against is the trigger and nothing else. This pins the window where the old
// rung was furthest off.
func TestForcingIsMeasuredAgainstTheTriggerOnALargeWindow(t *testing.T) {
	policy := Policy{Ratios: Ratios{Soft: 0.65, Smart: 0.80}, Window: 1_000_000, Reserve: 64_000}
	trigger := policy.Trigger()
	if want := int64(608_400); trigger != want {
		t.Fatalf("trigger = %d, want %d — 0.65 of the 936000 a turn can fill", trigger, want)
	}

	big := strings.Repeat("x", 100_000)
	conv := []nacelle.Message{nacelle.UserText("the task")}
	for i := range 30 {
		id := fmt.Sprintf("c%d", i)
		conv = append(conv,
			callMessage(id, "read"),
			resultMessage(id, "read", big),
			nacelle.AssistantText("done"),
			nacelle.UserText("continue"),
		)
	}
	plan := Plan(conv, Policy{AnchorMessages: 1, KeepTurns: 1})
	keeps := Fold{Kept: Blocks(conv, plan)}

	if LandsUnder(conv, plan, keeps, trigger) {
		t.Errorf("a history of %d tokens reported it lands under the trigger %d", EstTokens(Bytes(conv)), trigger)
	}
	if !LandsUnder(conv, plan, keeps.Forced(), trigger) {
		t.Error("folding the history away reported it does not land under the trigger")
	}
}
