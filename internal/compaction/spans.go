package compaction

import "github.com/FacileStudio/nacelle"

// Section returns the messages covered by every span of one zone, in order.
func Section(conv []nacelle.Message, spans []Span, zone Zone) []nacelle.Message {
	var out []nacelle.Message
	for _, span := range spans {
		if span.Zone == zone {
			out = append(out, conv[span.Start:span.End]...)
		}
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
// none yet.
func LedgerText(conv []nacelle.Message, spans []Span) string {
	for _, span := range spans {
		if span.Zone == ZoneLedger {
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
func LedgerEnd(conv []nacelle.Message, ledger, active int) int {
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

func clamp(n, low, high int) int {
	return min(max(n, low), high)
}
