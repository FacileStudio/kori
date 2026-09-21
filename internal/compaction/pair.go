package compaction

import "github.com/FacileStudio/nacelle"

// AlignedCut returns a cut no larger than want that never separates an assistant
// ToolCall message from the user ToolResult message answering it. A cut landing
// between the pair leaves the kept tail opening with a tool result whose call id
// was evicted, which Anthropic rejects with a 400. Pulling the cut back one
// message keeps the pair intact, so alignment never evicts more than asked for.
func AlignedCut(conv []nacelle.Message, want int) int {
	if want <= 0 || want >= len(conv) {
		return want
	}
	trIDs := toolResultIDs(conv[want])
	if len(trIDs) == 0 {
		return want
	}
	prev := conv[want-1]
	if prev.Role != nacelle.RoleAssistant {
		return want
	}
	calls := toolCallIDs(prev)
	for _, id := range trIDs {
		if calls[id] {
			return want - 1
		}
	}
	return want
}

// Covers reports whether a plan still describes a conversation: whether its
// spans tile the whole slice, contiguously and without overlapping, so every
// index a pass reasons about is one the conversation actually has.
//
// It lives beside AlignedCut because it is the same kind of arithmetic — both
// reason about which index is safe to touch — and it is the check a pass makes
// before it reassembles. A plan is measured against one conversation and applied
// to whatever is in the field when the pass lands, and a conversation that moved
// under a pass (a /clear or a /resume between the plan and the install) would
// otherwise be indexed past its own end or, worse, silently replaced by nothing.
func Covers(conv []nacelle.Message, spans []Span) bool {
	at := 0
	for _, span := range spans {
		if span.Start != at || span.End < span.Start {
			return false
		}
		at = span.End
	}
	return at == len(conv)
}

// toolResultIDs collects the ToolCall ids a message answers, empty for any user
// turn that is prose.
func toolResultIDs(msg nacelle.Message) []string {
	var ids []string
	for _, part := range msg.Parts {
		if tr, ok := part.(nacelle.ToolResult); ok {
			ids = append(ids, tr.ID)
		}
	}
	return ids
}

// toolCallIDs collects the ToolCall ids an assistant message asked for.
func toolCallIDs(msg nacelle.Message) map[string]bool {
	ids := map[string]bool{}
	for _, part := range msg.Parts {
		if tc, ok := part.(nacelle.ToolCall); ok {
			ids[tc.ID] = true
		}
	}
	return ids
}
