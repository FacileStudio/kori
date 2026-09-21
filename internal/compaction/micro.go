package compaction

import (
	"fmt"
	"strings"

	"github.com/FacileStudio/nacelle"
)

const (
	// MinResult is the smallest tool result worth tombstoning. Under it the
	// placeholder costs nearly what the result did, and a result that small is
	// usually load-bearing: an id, a path, a diff header.
	MinResult = 1024

	// DroppedNotice opens the text a tombstoned result is replaced with. The
	// pass skips a result already opening with it, so a second pass never pays a
	// placeholder twice or counts it as savings.
	DroppedNotice = "[dropped "

	// DroppedThinkingNotice opens the text a tombstoned thinking block is
	// replaced with. It plays the same role as DroppedNotice and uses the same
	// prefix so a reader sees one mechanism.
	DroppedThinkingNotice = "[dropped thinking: "
)

// MicroStats is what a tombstone pass did: how many results and thinking blocks
// it replaced, and the bytes it took out of the conversation.
type MicroStats struct {
	Results  int
	Thinking int
	Bytes    int
}

// EstTokens is the bytes-to-tokens estimate the whole package uses: four bytes
// per token is the rough English rate, and it is only ever compared against
// itself, so its error bars are directionally consistent.
func EstTokens(bytes int) int64 {
	return int64(bytes) / 4
}

// Bytes is the total byte weight of a conversation, the stand-in for "what is
// about to be discarded" when estimating what a pass frees.
func Bytes(conv []nacelle.Message) int {
	total := 0
	for _, msg := range conv {
		total += MsgBytes(msg)
	}
	return total
}

// MsgBytes is the byte weight of one message: the sum across its parts, which
// hold the tool output, the thinking and the spoken text.
func MsgBytes(msg nacelle.Message) int {
	total := 0
	for _, part := range msg.Parts {
		total += PartBytes(part)
	}
	return total
}

// PartBytes is the byte weight of a single part: tool results and reasoning
// dominate a long session, text is the message itself, anything else is zero.
func PartBytes(part nacelle.Part) int {
	switch typed := part.(type) {
	case nacelle.Text:
		return len(typed.Text)
	case nacelle.ToolResult:
		return len(typed.Result)
	case nacelle.Reasoning:
		return len(typed.Text)
	default:
		return 0
	}
}

// Tombstone replaces oversized tool results and thinking blocks in the given
// spans with placeholders, keeping the call/result pairing intact. It is
// deterministic, needs no backend, and is idempotent: a stub already in place is
// never re-stubbed or counted again. It mutates conv in place, which is safe
// because the caller owns the conversation on its own thread.
func Tombstone(conv []nacelle.Message, spans []Span) MicroStats {
	var stats MicroStats
	for _, span := range spans {
		if span.Zone != ZoneHistory {
			continue
		}
		for i := span.Start; i < span.End && i < len(conv); i++ {
			partial := tombstoneMessage(&conv[i])
			stats.Results += partial.Results
			stats.Thinking += partial.Thinking
			stats.Bytes += partial.Bytes
		}
	}
	return stats
}

// tombstoneMessage routes one message to the half that owns its role: only user
// messages carry tool results, only assistant messages carry reasoning.
func tombstoneMessage(msg *nacelle.Message) MicroStats {
	switch msg.Role {
	case nacelle.RoleUser:
		return tombstoneResults(msg)
	case nacelle.RoleAssistant:
		return tombstoneThinking(msg)
	default:
		return MicroStats{}
	}
}

func tombstoneResults(msg *nacelle.Message) MicroStats {
	var stats MicroStats
	for i, part := range msg.Parts {
		result, ok := part.(nacelle.ToolResult)
		if !ok || len(result.Result) < MinResult || strings.HasPrefix(result.Result, DroppedNotice) {
			continue
		}
		stats.Bytes += len(result.Result)
		msg.Parts[i] = nacelle.ToolResult{
			ID:     result.ID,
			Name:   result.Name,
			Failed: result.Failed,
			Result: fmt.Sprintf("%s%d bytes%s", DroppedNotice, len(result.Result), ". Re-run the tool if the detail matters."),
		}
		stats.Results++
	}
	return stats
}

func tombstoneThinking(msg *nacelle.Message) MicroStats {
	var stats MicroStats
	for i, part := range msg.Parts {
		reasoning, ok := part.(nacelle.Reasoning)
		if !ok || reasoning.Text == "" || strings.HasPrefix(reasoning.Text, DroppedThinkingNotice) {
			continue
		}
		stats.Bytes += len(reasoning.Text)
		msg.Parts[i] = nacelle.Reasoning{
			Text: fmt.Sprintf("%s%d bytes%s", DroppedThinkingNotice, len(reasoning.Text), ". See the assistant text above for the conclusion."),
		}
		stats.Thinking++
	}
	return stats
}
