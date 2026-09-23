package compaction

import (
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

// absorbedPairSample is a conversation whose first history block — the one the
// ledger sits immediately before — is a small atomic tool pair, the shape a judge
// can vote Keep: absorbing the call must not orphan the result. The turns after
// it are bulky, because a pass that folds nothing worth folding is refused and
// then there is no absorption to test.
func absorbedPairSample() []nacelle.Message {
	return []nacelle.Message{
		nacelle.UserText("the task"),
		callMessage("c1", "read"),
		resultMessage("c1", "read", "contents"),
		callMessage("c2", "read"),
		resultMessage("c2", "read", strings.Repeat("x", 40_000)),
		callMessage("c3", "read"),
		resultMessage("c3", "read", strings.Repeat("y", 40_000)),
		nacelle.AssistantText("a4"),
		nacelle.UserText("u4"),
		callMessage("c4", "read"),
		resultMessage("c4", "read", strings.Repeat("z", 40_000)),
		nacelle.AssistantText("a5"),
	}
}

// A kept tool pair absorbed by the ledger stays whole: no later pass, and no
// prune of the history that remains, can separate the call from its result. The
// second pass rebuilds the ledger from its text alone after the session has kept
// working, so the pair has to come back through the carry rather than surviving
// by luck. This is I1 across two passes, not just one.
func TestApplyKeepsAnAbsorbedToolPairWhole(t *testing.T) {
	policy := Policy{AnchorMessages: 1, KeepTurns: 3}
	conv := absorbedPairSample()
	plan := Plan(conv, policy)
	fold := foldVerdicts(Blocks(conv, plan), []Verdict{
		{Decision: Keep}, {Decision: Ledger}, {Decision: Ledger}, {Decision: Ledger}, {Decision: Ledger},
	}, false)

	out, _ := Apply(conv, plan, "folded", fold.Survives, false)
	assertPairsWhole(t, out, policy)

	worked := appendTurn(out, "c9")
	next, _ := Apply(worked, Plan(worked, policy), "folded again", nil, false)
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
		callMessage("c1", "read"),
		resultMessage("c1", "read", strings.Repeat("x", 40_000)),
		nacelle.UserText("u2"),
		nacelle.AssistantText("a3"),
		nacelle.UserText("u4"),
		nacelle.AssistantText("a5"),
	}
	policy := Policy{AnchorMessages: 1, KeepTurns: 2}

	out, stats := Apply(conv, Plan(conv, policy), "a new summary", nil, false)

	if stats.Refused {
		t.Fatalf("the pass was refused, so the carry is not being exercised: %+v", stats)
	}
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
