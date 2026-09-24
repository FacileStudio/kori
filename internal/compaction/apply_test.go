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
	return Policy{Ratios: Ratios{Soft: 0.65, Smart: 0.80}, Window: 200_000, AnchorMessages: 1, KeepTurns: 2}
}

// I2: the anchor is byte-identical after any number of passes. The session keeps
// working between passes, so each one really rebuilds the conversation around the
// ledger rather than being refused for having nothing worth folding.
func TestApplyKeepsTheAnchorByteForByte(t *testing.T) {
	policy := applyPolicy()
	conv := applySample()
	anchor := conv[0]

	for pass := range 5 {
		conv = appendTurn(conv, fmt.Sprintf("c%d", pass))
		next, _ := Apply(conv, Plan(conv, policy), fmt.Sprintf("fold %d", pass), nil, false)
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

	out, stats := Apply(conv, Plan(conv, policy), "Decisions:\n- done", nil, false)

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

// I4: the assembled conversation never has two consecutive messages of one role,
// and the collision is really repaired — the ledger and the turn after it share a
// role here, so the merge has to fold one into the other rather than leaving them
// adjacent. The history is bulky on purpose: a pass folding nothing worth folding
// is refused, and then nothing is being tested.
func TestApplyKeepsRolesAlternating(t *testing.T) {
	policy := Policy{AnchorMessages: 1, KeepTurns: 3}
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		callMessage("c1", "read"),
		resultMessage("c1", "read", strings.Repeat("x", 40_000)),
		callMessage("c2", "read"),
		resultMessage("c2", "read", strings.Repeat("y", 40_000)),
		nacelle.AssistantText("three"),
		nacelle.UserText("four"),
		nacelle.AssistantText("five"),
	}

	out, stats := Apply(conv, Plan(conv, policy), "Decisions:\n- merged", nil, false)

	if stats.Refused {
		t.Fatalf("the pass was refused, so the merge is not being exercised: %+v", stats)
	}
	assertAlternating(t, out)
}

// smallTurnSample is a conversation whose whole history is one two-word turn: a
// ledger that compresses it costs more than the turn did, which is the judged
// shape I5 has to refuse.
func smallTurnSample() []nacelle.Message {
	return []nacelle.Message{
		nacelle.UserText("the task"),
		nacelle.AssistantText("ok"),
		nacelle.UserText("u2"),
		nacelle.AssistantText("a2"),
	}
}

// I5: a pass never grows the estimate the report reads, whatever the tier. The
// judged shape is the one that can move the wrong way — a two-byte turn folded
// into a verbose ledger adds weight — and that rebuild is refused outright and
// the original handed back, rather than installed with a hopeful number.
func TestApplyNeverGrowsTheConversation(t *testing.T) {
	small := smallTurnSample()
	fold := foldVerdicts(Blocks(small, Plan(small, applyPolicy())), []Verdict{{Decision: Ledger}})

	tests := []struct {
		name    string
		conv    []nacelle.Message
		ledger  string
		keep    func(int) bool
		refused bool
	}{
		{"an unclassified pass folds the whole history", applySample(), "a short ledger", nil, false},
		{"a judged pass folds a two-byte turn into a long ledger", small, strings.Repeat("decision, constraint, dead end. ", 400), fold.Survives, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, stats := Apply(tc.conv, Plan(tc.conv, applyPolicy()), tc.ledger, tc.keep, false)
			if stats.Refused != tc.refused || stats.After > stats.Before {
				t.Fatalf("stats = %+v, want a refused pass of %v over %v", stats, tc.refused, tc.conv)
			}
			if tc.refused && !reflect.DeepEqual(out, tc.conv) {
				t.Errorf("a refused pass rewrote the conversation: %v", out)
			}
		})
	}
}

func assertAlternating(t *testing.T, conv []nacelle.Message) {
	t.Helper()
	for i := 1; i < len(conv); i++ {
		if conv[i].Role == conv[i-1].Role {
			t.Errorf("messages %d and %d share role %q: %v", i-1, i, conv[i].Role, conv)
		}
	}
}

// A plan is measured against one conversation and applied to whatever is in the
// field when the pass lands. A conversation that changed in between — here a
// shorter one, as a /clear or a /resume leaves behind — must be refused rather
// than indexed with the old spans: the assembly would either run off the end of
// it or silently replace it with nothing.
func TestApplyRefusesAPlanThatNoLongerCoversTheConversation(t *testing.T) {
	policy := applyPolicy()
	plan := Plan(applySample(), policy)
	moved := []nacelle.Message{nacelle.UserText("a different conversation")}

	out, stats := Apply(moved, plan, "a ledger for a conversation that is gone", nil, false)

	if !stats.Stale {
		t.Fatalf("stats = %+v, want the plan refused as stale", stats)
	}
	if stats.Before != stats.After || stats.Summarized != 0 {
		t.Errorf("stats = %+v, want no work claimed", stats)
	}
	if !reflect.DeepEqual(out, moved) {
		t.Errorf("conversation = %v, want it handed back untouched", out)
	}
}

// A pass that drops history but has no summary to install still installs the
// ledger buffer. The sentinel is what keeps the pinned head and the turns after
// it role-legal — the head's last turn and the next turn routinely share a role —
// and an empty ledger is cheaper than a boundary the backends refuse.
func TestApplyBuffersTheBoundaryWithoutALedger(t *testing.T) {
	policy := applyPolicy()
	conv := applySample()

	out, _ := Apply(conv, Plan(conv, policy), "", nil, false)

	if len(out) != 4 {
		t.Fatalf("conversation = %v, want the anchor, the buffer and the active window", out)
	}
	if !IsLedger(out[1]) {
		t.Fatalf("message 1 = %+v, want the ledger buffer", out[1])
	}
	if body := Body(out[1]); body != "" {
		t.Errorf("ledger body = %q, want nothing folded into it", body)
	}
	assertAlternating(t, out)
}

// A ledger with an empty body is not the same thing as a conversation with no
// ledger. The assembly installs the empty one on purpose when a pass dropped
// history and had no summary to stand in its place, and that message is the
// buffer keeping the pinned head and the turn after it role-legal — so reading its
// empty body as "no ledger" hands back the anchor and the active window with the
// ledger silently dropped and the two roles left colliding.
func TestApplyKeepsALedgerWithAnEmptyBody(t *testing.T) {
	policy := Policy{AnchorMessages: 1, KeepTurns: 2}
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		BuildLedger("", ""),
		nacelle.UserText("follow up"),
		nacelle.AssistantText("answer"),
	}

	out, stats := Apply(conv, Plan(conv, policy), "", nil, false)

	if len(out) != len(conv) {
		t.Fatalf("conversation = %v, want the ledger kept in place", out)
	}
	if !IsLedger(out[1]) {
		t.Fatalf("message 1 = %+v, want the ledger buffer", out[1])
	}
	if stats.Refused {
		t.Errorf("stats = %+v, want the ledger rebuilt rather than the pass refused", stats)
	}
	assertAlternating(t, out)
}

// A call that changes nothing is handed back untouched rather than rewritten to
// close a boundary no pass created: there is no history between the pinned head
// and the active window, so there is nothing to drop and nothing to buffer.
func TestApplyLeavesAnUnchangedConversationAlone(t *testing.T) {
	policy := Policy{AnchorMessages: 1, KeepTurns: 3}
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		nacelle.AssistantText("answer"),
		nacelle.UserText("newest"),
		nacelle.AssistantText("last"),
	}

	out, stats := Apply(conv, Plan(conv, policy), "", nil, false)

	if !reflect.DeepEqual(out, conv) {
		t.Errorf("conversation = %v, want it handed back untouched", out)
	}
	if stats.Refused {
		t.Error("an unchanged conversation was reported as refused")
	}
}
