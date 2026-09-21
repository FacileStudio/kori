package compaction

import (
	"fmt"
	"strings"

	"github.com/FacileStudio/nacelle"
)

// Block is one atomic unit of history: an assistant turn that asked for tools
// together with the user turn answering them, or one standalone turn. A prune
// removes whole blocks or nothing, which is what keeps every ToolResult paired
// with the ToolCall it answers, and gives the judge one decision per unit.
type Block struct {
	// Key is a stable identifier for the block, used as a judge question name.
	Key string
	// Start and End are the half-open range the block covers.
	Start, End int
	// Text is the block's rendered content, what a judge is shown.
	Text string
}

// Blocks splits the history spans of a plan into atomic blocks. Only history is
// chunked: the anchor, the ledger and the active window are never prune
// candidates. A block never opens on a ToolResult — an orphan reply is folded
// back into the block before it.
func Blocks(conv []nacelle.Message, spans []Span) []Block {
	var blocks []Block
	for _, span := range spans {
		if span.Zone == ZoneHistory && span.Start < span.End {
			blocks = append(blocks, spanBlocks(conv, span.Start, span.End)...)
		}
	}
	return blocks
}

func spanBlocks(conv []nacelle.Message, start, end int) []Block {
	var blocks []Block
	for i := start; i < end; {
		next := blockEnd(conv, i, end)
		blocks = append(blocks, newBlock(conv, i, next))
		i = next
	}
	return foldOrphanResults(conv, blocks)
}

// blockEnd is where the block opening at start ends: a tool call paired with the
// reply that answers it takes two messages, anything else takes one.
func blockEnd(conv []nacelle.Message, start, end int) int {
	if start+1 < end && opensToolPair(conv[start], conv[start+1]) {
		return start + 2
	}
	return start + 1
}

// opensToolPair reports whether reply answers a call made in call.
func opensToolPair(call, reply nacelle.Message) bool {
	if call.Role != nacelle.RoleAssistant {
		return false
	}
	calls := toolCallIDs(call)
	if len(calls) == 0 {
		return false
	}
	for _, id := range toolResultIDs(reply) {
		if calls[id] {
			return true
		}
	}
	return false
}

// foldOrphanResults merges a block that opens on a ToolResult back into the one
// before it, so no block ever starts with an answer whose question it does not
// carry. With no previous block the orphan is left alone: it is a malformed
// boundary the caller's own alignment should have prevented.
func foldOrphanResults(conv []nacelle.Message, blocks []Block) []Block {
	out := make([]Block, 0, len(blocks))
	for _, block := range blocks {
		if len(out) > 0 && opensWithToolResult(conv[block.Start]) {
			last := &out[len(out)-1]
			last.End = block.End
			last.Text = renderBlock(conv, last.Start, last.End)
			continue
		}
		out = append(out, block)
	}
	return out
}

func opensWithToolResult(msg nacelle.Message) bool {
	if len(msg.Parts) == 0 {
		return false
	}
	_, ok := msg.Parts[0].(nacelle.ToolResult)
	return ok
}

func newBlock(conv []nacelle.Message, start, end int) Block {
	return Block{
		Key:   fmt.Sprintf("block-%d", start),
		Start: start,
		End:   end,
		Text:  renderBlock(conv, start, end),
	}
}

// renderBlock is the text a judge is shown for one atomic block: every part of
// every message in it, in order, cut to maxBlockText so one block holding a whole
// tool result cannot dominate a request carrying dozens of them. A tool call
// carries its arguments, abbreviated: the name alone cannot tell two reads of
// different files apart, and a block is judged by what the call actually did.
//
// Reasoning is rendered here even though Tombstone leaves it alone and MsgBytes
// counts it as unsent. The two are not the same question: the tombstone pass
// edits a transcript the reader scrolls back to, where a stub buys no context,
// while the judge is shown the block's own content — and a chain of thought is
// often what says whether the turn after it was a decision or a dead end.
func renderBlock(conv []nacelle.Message, start, end int) string {
	var b strings.Builder
	for i := start; i < end; i++ {
		for _, part := range conv[i].Parts {
			switch typed := part.(type) {
			case nacelle.Text:
				b.WriteString(typed.Text)
			case nacelle.Reasoning:
				b.WriteString(typed.Text)
			case nacelle.ToolCall:
				b.WriteString("tool call ")
				b.WriteString(callText(typed, maxBlockInput))
			case nacelle.ToolResult:
				b.WriteString(typed.Result)
			}
			b.WriteByte('\n')
		}
	}
	return clampText(b.String(), maxBlockText)
}
