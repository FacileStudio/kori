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

	// maxBlockInput bounds a tool call's arguments in the text a judge is shown:
	// enough for a path, a command or a pattern, not enough for one call's payload
	// to dominate a request that carries dozens of blocks. callText is the only
	// reader.
	maxBlockInput = 200
)

// MicroStats is what a tombstone pass did: how many tool results it replaced,
// and the bytes it took out of the conversation.
type MicroStats struct {
	Results int
	Bytes   int
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

// measure is what one reassembly cost, in the same estimate Bytes reports.
func measure(conv, out []nacelle.Message, summarized int) Stats {
	return Stats{
		Before:     EstTokens(Bytes(conv)),
		After:      EstTokens(Bytes(out)),
		Summarized: summarized,
	}
}

// Tombstone replaces oversized tool results in the given spans with
// placeholders, keeping the call/result pairing intact. It is deterministic,
// needs no backend, and is idempotent: a stub already in place is never
// re-stubbed or counted again. It mutates conv in place, which is safe because
// the caller owns the conversation on its own thread.
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

// callText is one tool call as the text a judge is shown: the tool's name and
// its own arguments, abbreviated to at most n bytes. The arguments are the point
// — a block judged on "tool call read" alone cannot tell two reads of different
// files apart — and they are cut because a block is one of dozens in a single
// request. A cut through a multi-byte rune is trimmed rather than emitted broken:
// this text lands in a JSON body.
func callText(call nacelle.ToolCall, n int) string {
	args := strings.TrimSpace(string(call.Input))
	switch {
	case args == "":
		return call.Name
	case len(args) <= n:
		return call.Name + " " + args
	default:
		return call.Name + " " + strings.ToValidUTF8(args[:n], "") + "…"
	}
}
