package compaction

import (
	"fmt"
	"strings"

	"github.com/FacileStudio/nacelle"
)

// MicroStats is what a tombstone pass did: how many tool results it replaced,
// and the bytes it took out of the conversation.
type MicroStats struct {
	Results int
	Bytes   int
}

// droppable reports whether a tool result is one Tombstone would replace: big
// enough to be worth a placeholder, and not already carrying one.
func droppable(result nacelle.ToolResult) bool {
	return len(result.Result) >= MinResult && !strings.HasPrefix(result.Result, DroppedNotice)
}

// DroppableBytes is what a tombstone over the given spans would free, measured
// without changing anything. It is the pre-flight a caller runs to decide whether
// the pass is worth the cached prefix it invalidates — see MinCleared — since by
// the time Tombstone has returned the cache is already gone.
func DroppableBytes(conv []nacelle.Message, spans []Span) int {
	total := 0
	for _, span := range spans {
		if span.Zone == ZoneHistory {
			total += spanDroppableBytes(conv, span)
		}
	}
	return total
}

// spanDroppableBytes is what the messages of one history span hold that a
// tombstone would replace. The span is trusted no further than its own bounds:
// one measured against a conversation that has since moved on is trimmed rather
// than indexed past its end.
func spanDroppableBytes(conv []nacelle.Message, span Span) int {
	total := 0
	for i := span.Start; i < span.End && i < len(conv); i++ {
		for _, part := range conv[i].Parts {
			if result, ok := part.(nacelle.ToolResult); ok && droppable(result) {
				total += len(result.Result)
			}
		}
	}
	return total
}

// Tombstone replaces oversized tool results in the given spans with
// placeholders, keeping the call/result pairing intact. It is deterministic,
// needs no backend, and is idempotent: a stub already in place is never
// re-stubbed or counted again. It mutates conv in place, which is safe because
// the caller owns the conversation on its own thread.
//
// It applies whatever it finds: the judgement about whether a pass is worth the
// cache it invalidates belongs to the caller, which asks DroppableBytes first.
//
// Tool results are the whole of it, because they are the only history weight
// that reaches the backend. Reasoning is deliberately left alone: it is recorded
// and displayed but never sent back, since every backend drops it when it builds
// a request (nacelle's anthropic blocksOf and the shared oairunner sift both
// switch on Text, ToolCall and ToolResult and nothing else — Anthropic accepts a
// thinking block only with the signature it was issued with, and the stream
// never carries one). A tombstone on it would therefore free no context at all
// while editing a transcript the reader can still scroll back to, and the pass
// would report the savings as if it had bought something.
func Tombstone(conv []nacelle.Message, spans []Span) MicroStats {
	var stats MicroStats
	for _, span := range spans {
		if span.Zone != ZoneHistory {
			continue
		}
		for i := span.Start; i < span.End && i < len(conv); i++ {
			partial := tombstoneResults(&conv[i])
			stats.Results += partial.Results
			stats.Bytes += partial.Bytes
		}
	}
	return stats
}

// tombstoneResults replaces the oversized results in one message. A result that
// already opens with the notice is left alone, which is what makes a second pass
// free. A result is a user turn's part, and scanning every message rather than
// only the user's is deliberate: the pairing is a property of the ids, not of the
// role, so nothing here has to trust the role to find one.
func tombstoneResults(msg *nacelle.Message) MicroStats {
	var stats MicroStats
	for i, part := range msg.Parts {
		result, ok := part.(nacelle.ToolResult)
		if !ok || !droppable(result) {
			continue
		}
		stats.Bytes += len(result.Result)
		msg.Parts[i] = nacelle.ToolResult{
			ID:     result.ID,
			Name:   result.Name,
			Failed: result.Failed,
			Result: fmt.Sprintf("%s%d bytes] Re-run the tool if the detail matters.", DroppedNotice, len(result.Result)),
		}
		stats.Results++
	}
	return stats
}
