package compaction

import (
	"fmt"
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

// A session that keeps reading large files is folded down whenever it crosses
// the soft ratio, and forty turns later it is still under the hard ratio with
// its anchor intact and its roles alternating. This is the end-to-end statement
// the zone model exists to make, with no model call in the loop.
func TestLoadSettlesUnderTheHardRatio(t *testing.T) {
	window := int64(200_000)
	policy := Policy{
		Ratios:         Ratios{Soft: 0.65, Mid: 0.80, Hard: 0.90},
		Window:         window,
		KeepTurns:      3,
		AnchorMessages: 1,
	}
	conv := []nacelle.Message{nacelle.UserText("the task")}
	anchor := conv[0]

	for turn := range 40 {
		conv = appendTurn(conv, fmt.Sprintf("c%d", turn))
		if policy.Tier(EstTokens(Bytes(conv))) >= Soft {
			conv, _ = Apply(conv, policy, Plan(conv, policy), "Decisions:\n- kept going", nil)
		}
	}

	if got := EstTokens(Bytes(conv)); got > int64(policy.Ratios.Hard*float64(window)) {
		t.Errorf("final estimate = %d tokens, want it under the hard ratio", got)
	}
	if len(conv) == 0 || conv[0].Role != anchor.Role {
		t.Errorf("conversation starts with %v, want the anchor first", conv)
	}
	assertLoadShape(t, conv)
}

// appendTurn adds one read-and-continue turn, opening with a user turn when the
// last message was the assistant's so the transcript stays alternating.
func appendTurn(conv []nacelle.Message, id string) []nacelle.Message {
	if len(conv) > 0 && conv[len(conv)-1].Role == nacelle.RoleAssistant {
		conv = append(conv, nacelle.UserText("continue"))
	}
	big := strings.Repeat("x", 40_000)
	return append(conv,
		callMessage(id, "read"),
		resultMessage(id, "read", big),
		nacelle.AssistantText("done"),
		nacelle.UserText("continue"),
	)
}

func assertLoadShape(t *testing.T, conv []nacelle.Message) {
	t.Helper()
	for i := 1; i < len(conv); i++ {
		if conv[i].Role == conv[i-1].Role {
			t.Fatalf("messages %d and %d share role %q after the load", i-1, i, conv[i].Role)
		}
		if opensWithToolResult(conv[i]) && !opensToolPair(conv[i-1], conv[i]) {
			t.Fatalf("message %d is an orphan ToolResult after the load", i)
		}
	}
}
