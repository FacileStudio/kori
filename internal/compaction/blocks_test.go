package compaction

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

func TestBlocksPairCallsWithTheirResults(t *testing.T) {
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		callMessage("c1", "read"),
		resultMessage("c1", "read", "contents"),
		nacelle.AssistantText("answer"),
		nacelle.UserText("more"),
	}
	spans := Plan(conv, Policy{AnchorMessages: 1, KeepTurns: 1})

	blocks := Blocks(conv, spans)

	if len(blocks) != 2 {
		t.Fatalf("blocks = %d, want the tool pair and the standalone turn", len(blocks))
	}
	if blocks[0].Start != 1 || blocks[0].End != 3 {
		t.Errorf("block 0 = %+v, want the call and its result together", blocks[0])
	}
	if blocks[1].Start != 3 || blocks[1].End != 4 {
		t.Errorf("block 1 = %+v, want the standalone assistant turn", blocks[1])
	}
}

func TestBlocksCoverTheHistoryContiguously(t *testing.T) {
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		callMessage("c1", "read"),
		resultMessage("c1", "read", "contents"),
		nacelle.AssistantText("answer"),
		nacelle.UserText("turn"),
		nacelle.AssistantText("last"),
	}
	spans := Plan(conv, Policy{AnchorMessages: 1, KeepTurns: 1})

	blocks := Blocks(conv, spans)
	start, end, ok := HistoryRange(spans)
	if !ok {
		t.Fatal("plan has no history to block")
	}
	if len(blocks) == 0 || blocks[0].Start != start || blocks[len(blocks)-1].End != end {
		t.Fatalf("blocks = %v, want them to cover [%d,%d)", blocks, start, end)
	}
	for i := 1; i < len(blocks); i++ {
		if blocks[i].Start != blocks[i-1].End {
			t.Errorf("blocks %d and %d are not contiguous: %v", i-1, i, blocks)
		}
	}
}

// A block shows the judge what a call did, not only what it was called: two
// reads of different files are not the same block, and a judge told "tool call
// read" twice has nothing to decide between them on. The arguments are
// abbreviated, so one call cannot dominate a request carrying dozens of blocks.
func TestBlockTextCarriesTheToolCallArguments(t *testing.T) {
	call := callMessage("c1", "read")
	call.Parts[0] = nacelle.ToolCall{ID: "c1", Name: "read", Input: json.RawMessage(`{"path":"internal/tui/compact.go"}`), Finished: true}
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		call,
		resultMessage("c1", "read", "contents"),
		nacelle.AssistantText("answer"),
		nacelle.UserText("more"),
	}
	spans := Plan(conv, Policy{AnchorMessages: 1, KeepTurns: 1})

	blocks := Blocks(conv, spans)
	if len(blocks) == 0 {
		t.Fatal("no blocks to read the text of")
	}
	if text := blocks[0].Text; !strings.Contains(text, "internal/tui/compact.go") {
		t.Errorf("block text = %q, want the call's own arguments in it", text)
	}
	if text := blocks[0].Text; !strings.Contains(text, "tool call read") {
		t.Errorf("block text = %q, want the call still named", text)
	}
}

func TestBlocksIgnoreNonHistorySpans(t *testing.T) {
	conv := []nacelle.Message{nacelle.UserText("the task")}
	spans := []Span{{Zone: ZoneAnchor, Start: 0, End: 1}}

	if blocks := Blocks(conv, spans); len(blocks) != 0 {
		t.Errorf("blocks = %v, want nothing outside history", blocks)
	}
}

// I1: no block ever opens on a ToolResult, over randomly shaped well-formed
// conversations — so a prune of whole blocks can never leave an orphan result.
func TestBlocksNeverOpenOnAToolResult(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for range 300 {
		conv := randomConversation(rng)
		spans := Plan(conv, Policy{AnchorMessages: 1, KeepTurns: 2})
		for _, block := range Blocks(conv, spans) {
			if opensWithToolResult(conv[block.Start]) {
				t.Fatalf("block %+v opens on a ToolResult in %v", block, conv)
			}
		}
	}
}

// randomConversation builds an alternating conversation out of standalone turns
// and paired tool calls, the two shapes a real transcript holds.
func randomConversation(rng *rand.Rand) []nacelle.Message {
	conv := []nacelle.Message{nacelle.UserText("the task")}
	for i := 0; i < 1+rng.Intn(5); i++ {
		if rng.Intn(2) == 0 {
			id := fmt.Sprintf("c%d", i)
			conv = append(conv, callMessage(id, "read"), resultMessage(id, "read", "contents"))
			continue
		}
		conv = append(conv, nacelle.AssistantText("step"), nacelle.UserText("reply"))
	}
	return conv
}
