package compaction

import (
	"sort"

	"github.com/FacileStudio/nacelle"
)

// This file is the forcing decision: whether a pass folds the whole history
// rather than the blocks the judge left in place. It lives apart from macro.go
// and apply.go because the file cap is real and the question is one thing —
// what a fold costs once it is built — rather than a second half of either
// classifying or assembling.

// Forced is a fold with every block that would have survived in place folded
// into the ledger instead. It is the lever a pass pulls when the gentle fold does
// not land the conversation under its trigger.
//
// It is the same fold with one list moved, not a second classification: forcing
// cannot change what the judge pruned, so the one destructive verdict stays
// behind the probability and confidence gates that granted it.
//
// What forcing costs is bounded but not nothing, and the honest statement of it
// matters because it is what licenses acting on an estimate. It never *drops* a
// block — nothing leaves the conversation that a prune did not already remove.
// But Kept is precisely the set the judge decided must survive verbatim, and
// forcing hands those blocks to a summarizer whose prompt says "be as short as
// correctness allows". A fact in a kept block can therefore be lost to
// summarization. That is why the lever is pulled on an upper bound of the rebuilt
// size rather than on a guess: the estimate errs toward pulling it, so a fold is
// forced whenever it might not land, and never on the strength of a figure that
// flattered it. Forcing is a trade, not a free move.
//
// The merged list is re-sorted by Start so the summarizer is fed the history in
// the order it happened, which is how the unforced fold reads it too.
func (f Fold) Forced() Fold {
	if len(f.Kept) == 0 {
		return f
	}
	ledger := append(append(make([]Block, 0, len(f.Ledger)+len(f.Kept)), f.Ledger...), f.Kept...)
	sort.Slice(ledger, func(i, j int) bool { return ledger[i].Start < ledger[j].Start })
	return Fold{Ledger: ledger, Pruned: f.Pruned}
}

// LandsUnder reports whether folding conv along plan, keeping only the blocks
// fold survives, would leave the conversation under trigger.
//
// It exists because the forced fold used to be scheduled by a second ratio
// rather than asked for. A ratio over one number is a proxy for this question,
// and a coarse one: two conversations at 0.81 and 0.89 of the usable window ran
// the same pass, though only the second needed forcing. Asking directly costs one
// estimate and removes a rung from the ladder.
//
// The estimate is an upper bound on what Apply would assemble: every term is
// measured exactly except the ledger, which is bounded at the most it can grow to
// (see ledgerBound). So answering "it lands" only when the bound is under the
// trigger is a guarantee and not a guess — anything the bound does not clear
// forces instead, which is the direction that costs verbatim fidelity rather than
// a session left over its trigger for another pass to fail at again.
//
// The comparison is strict. The guards act when size > trigger, so a rebuild
// landing exactly on the trigger would not re-fire; the margin is bought against
// the estimate's own roughness rather than against the guard. Four bytes to the
// token is a rough English rate, and a pass that lands flush against the
// threshold is one whose next turn is decided by that roughness.
func LandsUnder(conv []nacelle.Message, plan []Span, fold Fold, trigger int64) bool {
	if trigger <= 0 {
		return true
	}
	return EstTokens(estimateRebuilt(conv, plan, fold)) < trigger
}

// estimateRebuilt is the byte weight of what Apply would assemble from conv along
// plan under fold, with the ledger's own size bounded rather than measured. It is
// an upper bound: every term below is exact except the ledger, and that one is
// bounded at its maximum.
//
// Every term assemble emits has to be counted here or the estimate stops being
// the bound LandsUnder relies on. The two that are easy to miss are the ledger's
// carry — the parts a same-role neighbour folded into the ledger message, and the
// reply messages answering the calls the ledger itself carries — because they sit
// in the ledger zone rather than in history, so no fold and no prune can reach
// them and they ride along verbatim on every pass.
func estimateRebuilt(conv []nacelle.Message, plan []Span, fold Fold) int {
	total := Bytes(Section(conv, plan, ZoneAnchor)) + Bytes(Section(conv, plan, ZoneActive))
	for _, span := range plan {
		if span.Zone != ZoneHistory {
			continue
		}
		for i := span.Start; i < span.End && i < len(conv); i++ {
			if fold.Survives(i) {
				total += MsgBytes(conv[i])
			}
		}
	}
	carryParts, carryMsgs := ledgerCarry(conv, plan)
	for _, part := range carryParts {
		total += PartBytes(part)
	}
	return total + Bytes(carryMsgs) + ledgerBound(conv, plan)
}

// ledgerBound is the most the rebuilt ledger message can weigh. It is not the
// body already in place: the default fold *merges*, and MergeLedger is monotone,
// so the new body holds every line the old one did plus whatever the summary
// adds. Bounding it by the old body alone would under-count by up to one
// summary, and under-counting is the unsafe direction — it would report a fold
// landing when the merge had pushed it back over.
//
// So the bound is the body in place plus a full summary's headroom, floored at
// one summary for a conversation that has no ledger yet. A consolidating pass
// replaces rather than merges and so lands under this bound too, its rewrite
// being capped at the same MaxLedgerTokens the summarizer writes under.
//
// The headroom is MaxLedgerTokens converted at bytesPerToken, which is the same
// rate the finished estimate is read back at, so the bound and the comparison it
// feeds are in one unit rather than two.
func ledgerBound(conv []nacelle.Message, plan []Span) int {
	summary := MaxLedgerTokens*bytesPerToken + len(Sentinel) + 2
	previous := MsgBytes(newLedgerMessage(LedgerText(conv, plan)))
	return max(previous+MaxLedgerTokens*bytesPerToken, summary)
}
