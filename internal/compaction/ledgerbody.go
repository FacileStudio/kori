package compaction

import (
	"strings"

	"github.com/FacileStudio/nacelle"
)

// MaxLedgerTokens is the size a ledger body may reach before the next pass is
// asked to rewrite it rather than add to it. One summary is the unit: the
// summarizer call is capped at this many tokens (the TUI aliases its own
// compactMaxTokens to this constant), so a ledger larger than one summary is
// carrying more than a compression is meant to, and a rewrite is the only way
// back down. Without a bound the ledger only ever grows — the fold is monotone
// by design — and it grows into the window it exists to protect.
const MaxLedgerTokens = 2000

// MergeLedger folds a summary into the ledger body it was written from, section
// by section, and never repeats a line the earlier body already carries. It is
// the deterministic half of the fold: the prompt asks the summarizer not to
// restate what the earlier ledger records, and this is what holds when it does
// anyway. Order comes from the earlier body, so a merged ledger keeps the shape
// the schema asked for rather than the order the last reply happened to use.
func MergeLedger(previous, summary string) string {
	previous, summary = strings.TrimSpace(previous), strings.TrimSpace(summary)
	switch {
	case previous == "":
		return summary
	case summary == "":
		return previous
	}
	return render(mergeSections(sections(previous), sections(summary)))
}

// LedgerOverBudget reports whether a ledger body has grown past what one summary
// may take, which is the one condition a pass consolidates under. It measures
// the body alone and never the conversation: the fold's I5 refusal compares
// whole-conversation estimates, so a body accumulating a couple of kilobytes a
// pass is invisible to it until the ledger dominates the window it was built to
// protect.
func LedgerOverBudget(body string) bool {
	return EstTokens(len(strings.TrimSpace(body))) > MaxLedgerTokens
}

// NextLedger is the body the rebuilt ledger carries, and whether that body is a
// rewrite of what came before rather than a merge into it. A merge is the
// default and the safe half: the earlier body survives whole. A replacement is
// granted only to a pass that asked for one and only when the rewrite still
// carries every identifier the earlier body named, so the one path that can lose
// a fact is gated on a check that refuses to lose one.
func NextLedger(previous, summary string, replace bool) (string, bool) {
	previous, summary = strings.TrimSpace(previous), strings.TrimSpace(summary)
	if replace && summary != "" && len(MissingIdentifiers(previous, summary)) == 0 {
		return summary, true
	}
	return MergeLedger(previous, summary), false
}

// newLedgerMessage wraps a finished body as the ledger: the sentinel, then the
// body. Both the fold and a consolidating rebuild end here, because the ledger
// is one message however its text was arrived at.
func newLedgerMessage(body string) nacelle.Message {
	text := Sentinel
	if body = strings.TrimSpace(body); body != "" {
		text += "\n\n" + body
	}
	return nacelle.Message{
		Role:  nacelle.RoleAssistant,
		Parts: []nacelle.Part{nacelle.Text{Text: text}},
	}
}
