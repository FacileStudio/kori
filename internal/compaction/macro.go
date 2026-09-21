package compaction

import (
	"context"
	"sort"
	"strings"

	"github.com/FacileStudio/nacelle"
)

// Fold is the classified shape of one plan's history: the blocks that survive in
// place, the blocks the summarizer must fold into the ledger, and the blocks
// pruned whole. It is what a classified pass applies.
type Fold struct {
	Kept   []Block
	Ledger []Block
	Pruned []Block
}

// Survives reports whether a conversation index is kept in place by this fold.
// It is the predicate Apply is handed, so a prune can only ever drop what the
// judge tagged. Kept comes out of Blocks in ascending order and its blocks never
// overlap, and Apply asks once per history index, so the lookup is a binary
// search over End rather than a walk of every kept block.
func (f Fold) Survives(index int) bool {
	at := sort.Search(len(f.Kept), func(i int) bool { return f.Kept[i].End > index })
	return at < len(f.Kept) && f.Kept[at].Start <= index
}

// LedgerMessages is the raw history the summarizer is fed: the ledger-tagged
// blocks, in order. It is the whole point of the division of labour — the model
// only ever sees the blocks the judge asked it to compress.
func (f Fold) LedgerMessages(conv []nacelle.Message) []nacelle.Message {
	var out []nacelle.Message
	for _, block := range f.Ledger {
		out = append(out, conv[block.Start:block.End]...)
	}
	return out
}

// LedgerSize is how many messages the ledger blocks carry.
func (f Fold) LedgerSize() int {
	return blockSize(f.Ledger)
}

// PrunedSize is how many messages the pruned blocks drop.
func (f Fold) PrunedSize() int {
	return blockSize(f.Pruned)
}

// JudgeRequest is everything one classification needs beyond the conversation:
// the task to judge against and whether the pass must fold the whole history
// (the hard tier, which overrides keep verdicts). The prune threshold lives on
// the judge itself, where the adapters that read it are built.
type JudgeRequest struct {
	Goal  string
	Force bool
}

// Classify runs the judge over a plan's history and sorts its blocks into a
// Fold. A nil judge — the default — keeps nothing and folds everything, so the
// pass is byte-for-byte the pre-judge behaviour. A judge error comes back with an
// all-keep fold, so a caller that ignores the error prunes nothing.
func Classify(ctx context.Context, conv []nacelle.Message, plan []Span, req JudgeRequest, judge Judge) (Fold, error) {
	blocks := Blocks(conv, plan)
	if judge == nil {
		return Fold{Ledger: blocks}, nil
	}
	verdicts, err := judge.Classify(ctx, req.Goal, blocks)
	if err != nil {
		return Fold{Kept: blocks}, err
	}
	return foldVerdicts(blocks, verdicts, req.Force), nil
}

// foldVerdicts sorts each block by its verdict, defaulting a missing or short
// answer to Keep and upgrading a Keep to Ledger when the pass is forced.
func foldVerdicts(blocks []Block, verdicts []Verdict, force bool) Fold {
	var fold Fold
	for i, block := range blocks {
		decision := Keep
		if i < len(verdicts) {
			decision = verdicts[i].Decision
		}
		if force && decision == Keep {
			decision = Ledger
		}
		switch decision {
		case Prune:
			fold.Pruned = append(fold.Pruned, block)
		case Ledger:
			fold.Ledger = append(fold.Ledger, block)
		default:
			fold.Kept = append(fold.Kept, block)
		}
	}
	return fold
}

// GoalText is the pinned task the judge classifies against: the anchor's own
// text. It is the one thing a pass never rewrites, which makes it the right
// yardstick for "does the task still need this".
func GoalText(conv []nacelle.Message, plan []Span) string {
	var b strings.Builder
	for _, msg := range Section(conv, plan, ZoneAnchor) {
		for _, part := range msg.Parts {
			if text, ok := part.(nacelle.Text); ok {
				b.WriteString(text.Text)
				b.WriteString("\n")
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func blockSize(blocks []Block) int {
	total := 0
	for _, block := range blocks {
		total += block.End - block.Start
	}
	return total
}
