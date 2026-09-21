package compaction

import (
	"strings"

	"github.com/FacileStudio/nacelle"
)

const (
	// MinResult is the smallest tool result worth tombstoning. Under it the
	// placeholder costs nearly what the result did, and a result that small is
	// usually load-bearing: an id, a path, a diff header.
	MinResult = 1024

	// MinCleared is the least a whole tombstone pass must free before it is worth
	// running at all. Clearing a result rewrites the messages after it, which is
	// exactly the prefix the provider had already cached, so a pass that trades
	// that re-write for a few hundred tokens costs more than it saves. MinResult
	// is the floor under one result; this is the floor under the pass, and eight
	// results' worth is the point where the freed context pays for the cache.
	MinCleared = 8 * MinResult

	// DroppedNotice opens the text a tombstoned result is replaced with. The
	// pass skips a result already opening with it, so a second pass never pays a
	// placeholder twice or counts it as savings.
	DroppedNotice = "[dropped "

	// maxBlockInput bounds a tool call's arguments in the text a judge is shown:
	// enough for a path, a command or a pattern, not enough for one call's payload
	// to dominate a request that carries dozens of blocks. callText is the only
	// reader.
	maxBlockInput = 200

	// maxBlockText bounds one history block's rendered text, so a block holding a
	// whole tool result cannot dominate the request it is one of. The block is what
	// the judge decides on, so the head of the result is kept and the tail cut with
	// a marker rather than the block being dropped.
	maxBlockText = 16 * 1024
)

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

// callText is one tool call as the text a judge is shown: the tool's name and
// its own arguments, abbreviated to at most n bytes. The arguments are the point
// — a block judged on "tool call read" alone cannot tell two reads of different
// files apart — and they are cut because a block is one of dozens in a single
// request. A cut through a multi-byte rune is trimmed rather than emitted broken:
// this text lands in a JSON body.
func callText(call nacelle.ToolCall, n int) string {
	args := strings.TrimSpace(string(call.Input))
	if args == "" {
		return call.Name
	}
	return call.Name + " " + clampText(args, n)
}

// clampText cuts text to at most n bytes and marks the cut with an ellipsis, so a
// reader can tell an abbreviated block from a complete one. A cut through a
// multi-byte rune is trimmed rather than emitted broken: this text lands in a
// JSON body, both as a judge's block and as a tool call's arguments.
func clampText(text string, n int) string {
	if n <= 0 || len(text) <= n {
		return text
	}
	return strings.ToValidUTF8(text[:n], "") + "…"
}
