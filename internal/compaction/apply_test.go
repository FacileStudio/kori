package compaction

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

// applySample is a conversation with a large tool result in its history, so a
// pass has something real to fold away.
func applySample() []nacelle.Message {
	return []nacelle.Message{
		nacelle.UserText("the task"),
		callMessage("c1", "read"),
		resultMessage("c1", "read", strings.Repeat("x", 40_000)),
		nacelle.AssistantText("answer one"),
		nacelle.UserText("follow up"),
		nacelle.AssistantText("answer two"),
	}
}

func applyPolicy() Policy {
	return Policy{Ratios: Ratios{Soft: 0.65, Mid: 0.80, Hard: 0.90}, Window: 200_000, AnchorMessages: 1, KeepTurns: 2}
}

// I2: the anchor is byte-identical after any number of passes.
func TestApplyKeepsTheAnchorByteForByte(t *testing.T) {
	policy := applyPolicy()
	conv := applySample()
	anchor := conv[0]

	for pass := range 5 {
		next, _ := Apply(conv, policy, Plan(conv, policy), fmt.Sprintf("fold %d", pass), nil)
		conv = next
		if !reflect.DeepEqual(conv[0], anchor) {
			t.Fatalf("pass %d rewrote the anchor: %+v", pass, conv[0])
		}
	}
}

// A pass installs one ledger in place of the history and keeps both ends.
func TestApplyInstallsALedgerAndKeepsTheEnds(t *testing.T) {
	policy := applyPolicy()
	conv := applySample()

	out, stats := Apply(conv, policy, Plan(conv, policy), "Decisions:\n- done", nil)

	if len(out) != len(conv)-2 {
		t.Fatalf("conversation = %d messages, want the three history turns replaced by one ledger", len(out))
	}
	if !IsLedger(out[1]) {
		t.Errorf("message 1 = %+v, want the ledger", out[1])
	}
	if !reflect.DeepEqual(out[0], conv[0]) {
		t.Errorf("anchor = %+v, want it verbatim", out[0])
	}
	if stats.Summarized != 3 {
		t.Errorf("summarized = %d, want the 3 history turns", stats.Summarized)
	}
}

// I4: the assembled conversation never has two consecutive messages of one role.
func TestApplyKeepsRolesAlternating(t *testing.T) {
	policy := Policy{AnchorMessages: 1, KeepTurns: 3}
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		nacelle.AssistantText("one"),
		nacelle.UserText("two"),
		nacelle.AssistantText("three"),
		nacelle.UserText("four"),
		nacelle.AssistantText("five"),
	}

	out, _ := Apply(conv, policy, Plan(conv, policy), "Decisions:\n- merged", nil)

	for i := 1; i < len(out); i++ {
		if out[i].Role == out[i-1].Role {
			t.Fatalf("messages %d and %d share role %q: %v", i-1, i, out[i].Role, out)
		}
	}
}

// I5: a pass never grows the estimate the report reads.
func TestApplyNeverGrowsTheConversation(t *testing.T) {
	conv := applySample()

	_, stats := Apply(conv, applyPolicy(), Plan(conv, applyPolicy()), "a short ledger", nil)

	if stats.After > stats.Before {
		t.Errorf("after = %d, want no larger than before = %d", stats.After, stats.Before)
	}
}

// absorbedPairSample is a conversation whose first history block — the one the
// ledger sits immediately before — is an atomic tool pair, the shape a judge can
// vote Keep: absorbing the call must not orphan the result.
func absorbedPairSample() []nacelle.Message {
	return []nacelle.Message{
		nacelle.UserText("the task"),
		callMessage("c1", "read"),
		resultMessage("c1", "read", strings.Repeat("x", 40_000)),
		nacelle.AssistantText("a1"),
		nacelle.UserText("u2"),
		nacelle.AssistantText("a2"),
		nacelle.UserText("u3"),
		nacelle.AssistantText("a3"),
		nacelle.UserText("u4"),
		nacelle.AssistantText("a4"),
	}
}

// A kept tool pair absorbed by the ledger stays whole: no later pass, and no
// prune of the history that remains, can separate the call from its result. The
// second pass folds everything it can, so what survives is exactly what the
// ledger absorbed — and that must still carry both halves of the pair. This is
// I1 across two passes, not just one.
func TestApplyKeepsAnAbsorbedToolPairWhole(t *testing.T) {
	policy := Policy{AnchorMessages: 1, KeepTurns: 3}
	conv := absorbedPairSample()
	plan := Plan(conv, policy)
	fold := foldVerdicts(Blocks(conv, plan), []Verdict{
		{Decision: Keep}, {Decision: Ledger}, {Decision: Ledger}, {Decision: Ledger}, {Decision: Ledger},
	}, false)

	out, _ := Apply(conv, policy, plan, "folded", fold.Survives)
	assertPairsWhole(t, out, policy)

	next, _ := Apply(out, policy, Plan(out, policy), "folded again", nil)
	assertPairsWhole(t, next, policy)
	if !carriesToolPair(next) {
		t.Errorf("the absorbed tool pair was dropped on the second pass: %v", next)
	}
}

// A turn the assembly folded into the ledger is not lost when the ledger is
// rebuilt from its text alone: whatever sits past the sentinel is carried
// forward explicitly, so re-passing the same conversation never sheds a fact.
func TestApplyCarriesWhatTheLedgerAbsorbed(t *testing.T) {
	ledger := BuildLedger("", "the state")
	ledger.Parts = append(ledger.Parts, nacelle.Text{Text: "absorbed fact"})
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		ledger,
		nacelle.UserText("u2"),
		nacelle.AssistantText("a3"),
		nacelle.UserText("u4"),
		nacelle.AssistantText("a5"),
	}
	policy := Policy{AnchorMessages: 1, KeepTurns: 2}

	out, _ := Apply(conv, policy, Plan(conv, policy), "a new summary", nil)

	if !containsText(out, "absorbed fact") {
		t.Errorf("the ledger's absorbed part was dropped: %v", out)
	}
	assertAlternating(t, out)
}

// assertPairsWhole checks I1: no block opens on a ToolResult, and every tool
// result that survives has its call in the message before it.
func assertPairsWhole(t *testing.T, conv []nacelle.Message, policy Policy) {
	t.Helper()
	for i, msg := range conv {
		if !opensWithToolResult(msg) {
			continue
		}
		if i == 0 || !opensToolPair(conv[i-1], msg) {
			t.Errorf("message %d opens on a ToolResult with no call before it: %v", i, conv)
		}
	}
	for _, block := range Blocks(conv, Plan(conv, policy)) {
		if opensWithToolResult(conv[block.Start]) {
			t.Errorf("block %+v opens on a ToolResult", block)
		}
	}
}

func carriesToolPair(conv []nacelle.Message) bool {
	calls, results := false, false
	for _, msg := range conv {
		for _, part := range msg.Parts {
			switch part.(type) {
			case nacelle.ToolCall:
				calls = true
			case nacelle.ToolResult:
				results = true
			}
		}
	}
	return calls && results
}

func containsText(conv []nacelle.Message, want string) bool {
	for _, msg := range conv {
		for _, part := range msg.Parts {
			if text, ok := part.(nacelle.Text); ok && strings.Contains(text.Text, want) {
				return true
			}
		}
	}
	return false
}

func assertAlternating(t *testing.T, conv []nacelle.Message) {
	t.Helper()
	for i := 1; i < len(conv); i++ {
		if conv[i].Role == conv[i-1].Role {
			t.Errorf("messages %d and %d share role %q: %v", i-1, i, conv[i].Role, conv)
		}
	}
}

// An empty ledger with no previous one just drops the history; it never installs
// an empty message.
func TestApplyWithNoLedgerDropsTheHistoryOnly(t *testing.T) {
	policy := applyPolicy()
	conv := applySample()

	out, _ := Apply(conv, policy, Plan(conv, policy), "", nil)

	if len(out) != 3 {
		t.Fatalf("conversation = %v, want the anchor and the active window only", out)
	}
	for _, msg := range out {
		if IsLedger(msg) {
			t.Errorf("installed a ledger with nothing to carry: %+v", msg)
		}
	}
}
