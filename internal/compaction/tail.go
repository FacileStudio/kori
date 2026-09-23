package compaction

import "github.com/FacileStudio/nacelle"

// This file is where the verbatim window ends: the budget that sizes it, and the
// two guards that keep the cut off anything it may not split. It lives beside
// policy.go rather than in it because the file cap is real and the ladder and the
// tail are two different questions — one decides how hard a pass is, the other
// what a pass may not touch.

// activeStart is where the verbatim window begins: the newest turns, sized by the
// token budget rather than by a count, held to a floor of KeepTurns messages,
// pulled back off a ToolResult boundary and never into the anchor. The pull-back
// is skipped when it would land on the ledger: the ledger is not dropped with the
// cut, it carries the very call the result answers, so the pair stays valid with
// the result opening the active window instead of the ledger being swallowed into
// it.
func activeStart(conv []nacelle.Message, n, anchor int, p Policy) int {
	want := walkBack(conv, clamp(n-max(p.KeepTurns, 1), anchor, n), anchor, p.keepBudget())
	if want <= anchor {
		return want
	}
	cut := max(AlignedCut(conv, want), anchor)
	if cut < want && IsLedger(conv[cut]) {
		return want
	}
	return cut
}

// walkBack widens a cut over whole messages while the budget still has room for
// them, newest first, and never past the anchor. It stops reaching as soon as one
// more message would not fit, rather than skipping it and taking an older smaller
// one: the tail is the part of the conversation a model reads in order, and a hole
// in the middle of it is worse than a shorter window.
func walkBack(conv []nacelle.Message, cut, anchor, budget int) int {
	if budget <= 0 {
		return cut
	}
	spent := 0
	for cut > anchor {
		size := MsgBytes(conv[cut-1])
		if spent+size > budget {
			break
		}
		spent += size
		cut--
	}
	return cut
}

// keepBudget is the byte budget the newest turns are widened to on top of the
// message floor, capped so the token-driven part of the verbatim window can never
// take more than half of what a pass may fill. The cap is load-bearing rather than
// tidy: a tail that size leaves nothing for the ladder to fold, so the pass it
// triggers could not land under its own trigger whatever it summarized, and the
// session would sit in the thrash guard instead.
//
// The cap bounds the budget and not the floor. activeStart lays the KeepTurns
// floor down before this budget widens anything, and the active window is never
// rewritten, summarized or pruned by any tier, so a generous floor keeps the tail
// over half however small the budget is: the message floor is the one bound no
// pass can move. It is deliberately the smaller number — the shipped floor is the
// live turn alone — so the budget does the sizing and this cap has room to bite.
// Zero means the session asked for no budget and the message floor is the whole
// of it.
//
// What it is half of is the smaller of the usable window and the ceiling, since
// both bound the same thing from different sides. The ceiling is the term that
// matters on a backend reporting no window at all: there the ladder has nothing to
// measure, the ceiling is the only trigger, and a flat tail budget larger than
// half of it would leave every pass unable to land.
func (p Policy) keepBudget() int {
	if p.KeepTokens <= 0 {
		return 0
	}
	tokens := p.KeepTokens
	if limit := p.tailLimit(); limit > 0 && tokens > limit/2 {
		tokens = limit / 2
	}
	return int(tokens) * bytesPerToken
}

// tailLimit is the smallest budget the verbatim tail may take half of, zero when
// neither bound is known.
func (p Policy) tailLimit() int64 {
	limit := p.Usable()
	if p.Ceiling > 0 && (limit <= 0 || p.Ceiling < limit) {
		return p.Ceiling
	}
	return limit
}
