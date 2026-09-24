package compaction

import (
	"testing"

	"github.com/FacileStudio/nacelle"
)

// The ladder is measured against the window, and each boundary is inclusive:
// 0.65 is soft, 0.649 is not. The thresholds are computed from a rounded product
// so 0.65 × 200000 lands exactly on 130000 rather than a float hair above it.
func TestTierBoundaries(t *testing.T) {
	p := Policy{Ratios: Ratios{Soft: 0.65, Smart: 0.80}, Window: 200_000}
	tests := []struct {
		name string
		size int64
		want Tier
	}{
		{"under soft", 129_999, Below},
		{"at soft", 130_000, Soft},
		{"under smart", 159_999, Soft},
		{"at smart", 160_000, Smart},
		{"over smart", 199_999, Smart},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := p.Tier(tc.size); got != tc.want {
				t.Errorf("Tier(%d) = %v, want %v", tc.size, got, tc.want)
			}
		})
	}
}

// A backend that reports no window has no ratio to measure against, so the
// absolute ceiling is the only trigger and it buys the full pass: there is no
// soft ratio to tombstone against for free.
func TestTierFallsBackToTheCeilingWithoutAWindow(t *testing.T) {
	p := Policy{Ceiling: 100_000}
	if got := p.Tier(99_999); got != Below {
		t.Errorf("Tier under the ceiling = %v, want Below", got)
	}
	if got := p.Tier(100_000); got != Smart {
		t.Errorf("Tier at the ceiling = %v, want the full smart pass", got)
	}
}

// A compact_at pinned below the soft ratio still floors the tier at soft, so a
// session that asks to compact early does not silently do nothing.
func TestCeilingFloorsTheTierAtSoft(t *testing.T) {
	p := Policy{Ratios: Ratios{Soft: 0.65, Smart: 0.80}, Window: 200_000, Ceiling: 100_000}
	if got := p.Tier(100_000); got != Soft {
		t.Errorf("Tier at a low ceiling = %v, want Soft", got)
	}
	if got := p.Tier(99_999); got != Below {
		t.Errorf("Tier under the ceiling = %v, want Below", got)
	}
}

func TestTriggerPrefersTheCeiling(t *testing.T) {
	pinned := Policy{Ratios: Ratios{Soft: 0.65}, Window: 200_000, Ceiling: 100_000}
	if got := pinned.Trigger(); got != 100_000 {
		t.Errorf("Trigger with a ceiling = %d, want 100000", got)
	}
	derived := Policy{Ratios: Ratios{Soft: 0.65}, Window: 200_000}
	if got := derived.Trigger(); got != 130_000 {
		t.Errorf("Trigger without a ceiling = %d, want the soft ratio 130000", got)
	}
	if got := (Policy{}).Trigger(); got != 0 {
		t.Errorf("Trigger with nothing set = %d, want 0", got)
	}
}

// Plan covers the whole conversation in order, with the anchor pinned at the
// head, the active window at the tail and everything between as history.
func TestPlanPartitionsTheConversation(t *testing.T) {
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		nacelle.AssistantText("working"),
		nacelle.UserText("more"),
		nacelle.AssistantText("working"),
		nacelle.UserText("newest"),
		nacelle.AssistantText("done"),
	}
	p := Policy{KeepTurns: 3, AnchorMessages: 1}

	spans := Plan(conv, p)

	want := []Span{
		{Zone: ZoneAnchor, Start: 0, End: 1},
		{Zone: ZoneHistory, Start: 1, End: 3},
		{Zone: ZoneActive, Start: 3, End: 6},
	}
	if len(spans) != len(want) {
		t.Fatalf("Plan = %v, want %v", spans, want)
	}
	for i := range want {
		if spans[i] != want[i] {
			t.Errorf("span %d = %+v, want %+v", i, spans[i], want[i])
		}
	}
}

// ledgerWithCall is a state ledger carrying a tool call, the shape assembly
// leaves behind once it has absorbed a kept tool block.
func ledgerWithCall(id string) nacelle.Message {
	ledger := BuildLedger("", "state")
	ledger.Parts = append(ledger.Parts, nacelle.ToolCall{ID: id, Name: "read", Finished: true})
	return ledger
}

// A ledger claims the replies answering the calls it carries, so the result never
// stands in history as an orphan a later prune could drop while the call stayed
// behind in the ledger.
func TestPlanExtendsTheLedgerOverItsReplies(t *testing.T) {
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		ledgerWithCall("c1"),
		resultMessage("c1", "read", "contents"),
		nacelle.AssistantText("done"),
		nacelle.UserText("next"),
		nacelle.AssistantText("last"),
	}

	spans := Plan(conv, Policy{AnchorMessages: 1, KeepTurns: 2})

	ledger := Span{}
	for _, span := range spans {
		if span.Zone == ZoneLedger {
			ledger = span
		}
	}
	if ledger.Start != 1 || ledger.End != 3 {
		t.Fatalf("ledger span = %+v, want the call and its reply as [1,3)", ledger)
	}
	for _, block := range Blocks(conv, spans) {
		if opensWithToolResult(conv[block.Start]) {
			t.Errorf("block %+v opens on a ToolResult the ledger already claimed", block)
		}
	}
}

// The active boundary is not pulled back onto the ledger: the ledger is not
// dropped with the cut, it carries the call the result answers, so the result
// may open the active window rather than swallowing the ledger into it.
func TestPlanDoesNotSwallowTheLedgerIntoTheActiveWindow(t *testing.T) {
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		ledgerWithCall("c1"),
		resultMessage("c1", "read", "contents"),
		nacelle.UserText("next"),
		nacelle.AssistantText("last"),
	}

	spans := Plan(conv, Policy{AnchorMessages: 1, KeepTurns: 3})

	active := Section(conv, spans, ZoneActive)
	if len(active) == 0 || active[0].Role != nacelle.RoleUser {
		t.Errorf("active window = %v, want it to open on the result, not the ledger", active)
	}
	if LedgerText(conv, spans) == "" {
		t.Error("the ledger was swallowed into the active window")
	}
}

// The plan never opens the active window on a ToolResult whose call the cut
// dropped, and never bites into the anchor to do it.
func TestPlanAlignsTheActiveBoundary(t *testing.T) {
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		callMessage("call_1", "read"),
		resultMessage("call_1", "read", "contents"),
		nacelle.AssistantText("done"),
	}
	p := Policy{KeepTurns: 1, AnchorMessages: 1}

	spans := Plan(conv, p)

	active := Section(conv, spans, ZoneActive)
	if len(active) == 0 || opensWithToolResult(active[0]) {
		t.Errorf("active window = %v, want a boundary clear of the tool result", active)
	}
}

// A conversation whose head is an assistant ToolCall must not leave the matching
// ToolResult as the first history block: that orphan block could be folded or
// pruned away while its call stayed pinned in the anchor, and the provider
// rejects a call with no answer. The head is extended over its own replies, the
// same way a ledger claims its own.
func TestPlanNeverStartsHistoryOnAnOrphanResult(t *testing.T) {
	conv := []nacelle.Message{
		{Role: nacelle.RoleAssistant, Parts: []nacelle.Part{nacelle.ToolCall{ID: "c1", Name: "read", Finished: true}}},
		resultMessage("c1", "read", "contents"),
		nacelle.AssistantText("done"),
		nacelle.UserText("next"),
		nacelle.AssistantText("last"),
	}
	policy := Policy{AnchorMessages: 1, KeepTurns: 2}

	spans := Plan(conv, policy)

	if anchor := Section(conv, spans, ZoneAnchor); len(anchor) != 2 {
		t.Fatalf("anchor = %v, want the call and its reply pinned together", anchor)
	}
	for _, block := range Blocks(conv, spans) {
		if opensWithToolResult(conv[block.Start]) {
			t.Errorf("block %+v opens on a ToolResult the head already carries", block)
		}
	}

	out, _ := Apply(conv, spans, "", nil, false)
	for i, msg := range out {
		if opensWithToolResult(msg) && (i == 0 || !opensToolPair(out[i-1], msg)) {
			t.Errorf("message %d is an orphan ToolResult after the fold: %v", i, out)
		}
	}
}
