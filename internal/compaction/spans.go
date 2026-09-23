package compaction

import "github.com/FacileStudio/nacelle"

// Section returns the messages covered by every span of one zone, in order. A
// span that reaches past the conversation, or one that ends before it starts, is
// trimmed rather than trusted: the plan is measured against one conversation and
// read against whatever is in the field, and a selector is the wrong place to
// find out the two disagree.
func Section(conv []nacelle.Message, spans []Span, zone Zone) []nacelle.Message {
	var out []nacelle.Message
	for _, span := range spans {
		if span.Zone != zone || span.Start >= span.End || span.Start >= len(conv) {
			continue
		}
		out = append(out, conv[span.Start:min(span.End, len(conv))]...)
	}
	return out
}

// HistoryMessages is every message a pass may touch: the history spans, in
// order, with the ledger and both pinned ends left out.
func HistoryMessages(conv []nacelle.Message, spans []Span) []nacelle.Message {
	return Section(conv, spans, ZoneHistory)
}

// HistoryRange is the first and last index a plan's history spans cover, with ok
// false when the plan has no history at all.
func HistoryRange(spans []Span) (start, end int, ok bool) {
	for _, span := range spans {
		if span.Zone != ZoneHistory {
			continue
		}
		if !ok {
			start = span.Start
		}
		end, ok = span.End, true
	}
	return start, end, ok
}

// LedgerText is the body of the conversation's state ledger, empty when it has
// none yet. A ledger span past the end of the conversation is a plan that no
// longer describes it, so it reads as no ledger rather than an out-of-range
// panic.
func LedgerText(conv []nacelle.Message, spans []Span) string {
	for _, span := range spans {
		if span.Zone == ZoneLedger && span.Start < len(conv) {
			return Body(conv[span.Start])
		}
	}
	return ""
}

// LedgerEnd is where a ledger span ends: one message, plus any replies that
// answer the tool calls the ledger itself carries. A ledger that absorbed a kept
// ToolCall must absorb its ToolResult too, or the result would stand in history
// as an orphan a later prune could drop while the call stayed behind. The scan
// stops at the active window, which is never touched either way.
//
// An index the conversation does not have is not a ledger, so it has no replies
// to claim and the end is the index itself: a plan read against a shorter
// conversation leaves an empty span rather than running off its end.
func LedgerEnd(conv []nacelle.Message, ledger, active int) int {
	if ledger < 0 || ledger >= len(conv) {
		return ledger
	}
	calls := toolCallIDs(conv[ledger])
	end := ledger + 1
	for end < active && end < len(conv) && answersAny(conv[end], calls) {
		end++
	}
	return end
}

// answersAny reports whether a message replies to one of the given tool-call
// ids, so a ledger can claim the result that belongs to a call it carries.
func answersAny(msg nacelle.Message, calls map[string]bool) bool {
	for _, id := range toolResultIDs(msg) {
		if calls[id] {
			return true
		}
	}
	return false
}

// appendHistory adds a history span for a range, skipping an empty one so a plan
// never carries a zero-width span.
func appendHistory(spans []Span, start, end int) []Span {
	if start >= end {
		return spans
	}
	return append(spans, Span{Zone: ZoneHistory, Start: start, End: end})
}

func clamp(n, low, high int) int {
	return min(max(n, low), high)
}
