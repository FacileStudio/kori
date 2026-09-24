package compaction

import (
	"testing"
)

// This file covers Fold.Forced: the lever that moves a fold's surviving blocks
// into the ledger. The estimate that decides *when* to pull it lives in
// landsunder_test.go.

// Forcing only ever moves a block out of the conversation: Kept empties, Ledger
// grows, Pruned is untouched. That monotonicity is what makes deciding on an
// estimate safe — forcing always helps the estimate, so a wrong one costs
// verbatim fidelity and never a dropped block, and the one destructive verdict
// stays behind the probability and confidence gates that granted it.
func TestForcedFoldNeverKeepsMoreThanTheGentleOne(t *testing.T) {
	conv := judgeSample()
	plan := judgePlan(conv)
	gentle := foldVerdicts(Blocks(conv, plan), []Verdict{
		{Decision: Prune}, {Decision: Keep}, {Decision: Keep},
	})
	forced := gentle.Forced()

	if forced.PrunedSize() != gentle.PrunedSize() {
		t.Errorf("forced pruned = %d, want the gentle %d: forcing must not touch a prune", forced.PrunedSize(), gentle.PrunedSize())
	}
	if forced.LedgerSize() < gentle.LedgerSize() {
		t.Errorf("forced ledger = %d, want at least the gentle %d", forced.LedgerSize(), gentle.LedgerSize())
	}
	for i := range conv {
		if forced.Survives(i) {
			t.Errorf("index %d survives a forced fold, want nothing kept in place", i)
		}
	}
	if gentleBytes, forcedBytes := estimateRebuilt(conv, plan, gentle), estimateRebuilt(conv, plan, forced); forcedBytes > gentleBytes {
		t.Errorf("forced estimate = %d bytes, want no more than the gentle %d", forcedBytes, gentleBytes)
	}
}

// A fold that already keeps nothing is forced as it stands, so the lever is free
// to be pulled twice — which a consolidating pass does.
func TestForcingAnAlreadyForcedFoldChangesNothing(t *testing.T) {
	conv := judgeSample()
	plan := judgePlan(conv)
	already := Fold{Ledger: Blocks(conv, plan)}

	got := already.Forced()
	if got.LedgerSize() != already.LedgerSize() || got.PrunedSize() != already.PrunedSize() {
		t.Errorf("forced an already-forced fold: got %d ledger / %d pruned, want %d / %d",
			got.LedgerSize(), got.PrunedSize(), already.LedgerSize(), already.PrunedSize())
	}
}

// Forcing merges two lists that were each in order, so the union is not. The
// summarizer is fed the history in the order it happened — the same order the
// unforced fold reads it in — so the merge has to be re-sorted rather than
// concatenated.
func TestForcedFoldFeedsTheSummarizerInOrder(t *testing.T) {
	conv := judgeSample()
	plan := judgePlan(conv)
	blocks := Blocks(conv, plan)
	if len(blocks) < 2 {
		t.Fatalf("the fixture has %d blocks, want at least two to interleave", len(blocks))
	}

	gentle := Fold{Ledger: blocks[1:], Kept: blocks[:1]}
	forced := gentle.Forced()

	if len(forced.Ledger) != len(blocks) {
		t.Fatalf("forced ledger = %d blocks, want all %d", len(forced.Ledger), len(blocks))
	}
	for i := 1; i < len(forced.Ledger); i++ {
		if forced.Ledger[i].Start < forced.Ledger[i-1].Start {
			t.Errorf("ledger block %d starts at %d, before block %d at %d: the merge was not re-sorted",
				i, forced.Ledger[i].Start, i-1, forced.Ledger[i-1].Start)
		}
	}
	if got := forced.LedgerMessages(conv); len(got) != blockSize(forced.Ledger) {
		t.Errorf("LedgerMessages = %d messages, want the %d the blocks cover", len(got), blockSize(forced.Ledger))
	}
}

// The judge's own keeps are what forcing overrides, so the fold it produces must
// still carry every one of them into the summarizer rather than dropping them.
func TestForcedFoldCarriesEveryKeptBlock(t *testing.T) {
	conv := judgeSample()
	plan := judgePlan(conv)
	blocks := Blocks(conv, plan)
	gentle := Fold{Kept: blocks}

	forced := gentle.Forced()

	for _, block := range blocks {
		found := false
		for _, folded := range forced.Ledger {
			if folded.Start == block.Start && folded.End == block.End {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("block %d-%d was in Kept but is not in the forced ledger", block.Start, block.End)
		}
	}
}
