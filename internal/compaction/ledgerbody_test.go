package compaction

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

// longLedger is a body past its budget: ten kilobytes of recorded facts, which
// is the size a pass is asked to consolidate rather than add to.
func longLedger() string {
	return "Decisions:\n- keep `internal/compaction/ledger.go`\nState:\n" + strings.Repeat("- recorded fact\n", 600)
}

// The regression this file exists for. The summarizer is handed the earlier
// ledger and told to build on it, so it restates what the ledger already holds —
// and a fold that concatenated on a restatement doubled the body every pass.
// Restating a fact must cost nothing, and the merge is what makes that true even
// when the prompt's own instruction is ignored.
func TestMergeLedgerDoesNotRepeatWhatTheEarlierBodyRecorded(t *testing.T) {
	previous := "Decisions:\n- use the ledger\n- pin the anchor\n\nState:\n- pass one landed"
	summary := "Decisions:\n- Use the ledger.\n- add the judge\n\nState:\n- pass one landed"

	merged := MergeLedger(previous, summary)

	if got := strings.Count(merged, "use the ledger"); got != 1 {
		t.Errorf("merged = %q, want the restated decision recorded once, not %d times", merged, got)
	}
	if strings.Count(merged, "State:") != 1 {
		t.Errorf("merged = %q, want one State section rather than one per pass", merged)
	}
	if !strings.Contains(merged, "add the judge") {
		t.Errorf("merged = %q, want the new decision kept", merged)
	}
	if strings.Index(merged, "Decisions:") > strings.Index(merged, "State:") {
		t.Errorf("merged = %q, want the earlier body's section order kept", merged)
	}
}

// The end-to-end version of the same defect, which is the one that actually grew
// a session: eight passes whose summarizer writes its own section list every time
// — reproducing what it was shown in its own formatting, headers without colons
// and bullets with periods — used to leave eight copies of every fact. The count
// is the assertion, not the size: growth is fine, duplication is not.
func TestRepeatedPassesDoNotDoubleTheLedger(t *testing.T) {
	policy := applyPolicy()
	conv := applySample()

	for pass := range 8 {
		echoed := "Decisions\n* decided A.\nState\n* pass " + fmt.Sprint(pass) + " landed."
		conv = appendTurn(conv, fmt.Sprintf("c%d", pass))
		out, stats := Apply(conv, Plan(conv, policy), echoed, nil, false)
		if stats.Refused {
			t.Fatalf("pass %d was refused, so the fold is not being exercised: %+v", pass, stats)
		}
		conv = out
	}

	body := Body(conv[1])
	if got := strings.Count(body, "decided A"); got != 1 {
		t.Errorf("ledger body = %q, want the decision recorded once, not %d times", body, got)
	}
	for pass := range 8 {
		if got := strings.Count(body, fmt.Sprintf("pass %d landed", pass)); got != 1 {
			t.Errorf("ledger body = %q, want pass %d recorded once, not %d times", body, pass, got)
		}
	}
}

// Monotonicity is the property the old fold promised and this one has to keep:
// nothing an earlier body recorded is dropped by a later merge. Idempotence
// falls out of it — merging a body into itself is the same body.
func TestMergeLedgerKeepsEveryFactItWasGiven(t *testing.T) {
	once := MergeLedger("", "decided A")
	twice := MergeLedger(once, "learned B")

	if !strings.Contains(twice, once) || !strings.Contains(twice, "learned B") {
		t.Errorf("merged = %q, want both %q and the new fact", twice, once)
	}
	if again := MergeLedger(twice, twice); again != twice {
		t.Errorf("MergeLedger(body, body) = %q, want %q", again, twice)
	}
}

// A section the summary introduces and the earlier body never had is kept, not
// discarded: the merge is where the new pass's material lands.
func TestMergeLedgerKeepsASectionOnlyTheSummaryCarries(t *testing.T) {
	merged := MergeLedger("Decisions:\n- a", "Open questions:\n- whether the judge ships")

	if !strings.Contains(merged, "Open questions:") || !strings.Contains(merged, "whether the judge ships") {
		t.Errorf("merged = %q, want the section the summary introduced", merged)
	}
}

// The budget is the one summary's worth, and it is what asks for a rewrite.
func TestLedgerOverBudgetHoldsAtOneSummary(t *testing.T) {
	if LedgerOverBudget("Decisions:\n- short") {
		t.Error("a short body was over budget, want it under")
	}
	if !LedgerOverBudget(longLedger()) {
		t.Error("a ten-kilobyte body was under budget, want it over")
	}
}

// A pass that asked for a consolidated ledger and returned one that still names
// everything the old body named is granted the rewrite, which is the only way a
// ledger that only ever grew gets smaller.
func TestNextLedgerGrantsARewriteThatKeepsTheIdentifiers(t *testing.T) {
	previous := "Decisions:\n- keep `internal/compaction/ledger.go`\nState:\n" + strings.Repeat("- fact\n", 100)
	summary := "Decisions:\n- keep `internal/compaction/ledger.go`, consolidated"

	body, replaced := NextLedger(previous, summary, true)

	if !replaced || body != summary {
		t.Errorf("NextLedger = (%q, %v), want the rewrite granted", body, replaced)
	}
}

// The rewrite is refused the moment it stops carrying an identifier the ledger
// held. Growth is the failure mode that leaves open, and it is the right one:
// the next pass can attack a longer ledger, nobody can attack a forgotten one.
func TestNextLedgerRefusesARewriteThatDropsAnIdentifier(t *testing.T) {
	previous := "Decisions:\n- keep `internal/compaction/ledger.go`\nState:\n- pinned v1.2.3"
	summary := "Decisions:\n- consolidating, see the notes"

	body, replaced := NextLedger(previous, summary, true)

	if replaced {
		t.Fatalf("NextLedger = (%q, true), want the rewrite refused", body)
	}
	if !strings.Contains(body, "internal/compaction/ledger.go") || !strings.Contains(body, "v1.2.3") {
		t.Errorf("body = %q, want the merge to have kept every identifier", body)
	}
}

// A pass that was not asked to consolidate never rewrites the ledger, however
// complete its summary looks: adding to the ledger is the default.
func TestNextLedgerMergesWhenNoRewriteWasAskedFor(t *testing.T) {
	previous := "Decisions:\n- keep `internal/compaction/ledger.go`"
	summary := "Decisions:\n- keep `internal/compaction/ledger.go`\n- and use the judge"

	body, replaced := NextLedger(previous, summary, false)

	if replaced {
		t.Fatal("NextLedger replaced the ledger, want a merge when no rewrite was asked for")
	}
	if !strings.Contains(body, "use the judge") {
		t.Errorf("body = %q, want the new decision merged in", body)
	}
}

// overdueLedgerConversation is a conversation already carrying a ledger past its
// budget, with something left in its history for a pass to fold.
func overdueLedgerConversation() []nacelle.Message {
	return []nacelle.Message{
		nacelle.UserText("the task"),
		BuildLedger("", longLedger()),
		callMessage("c1", "read"),
		resultMessage("c1", "read", strings.Repeat("x", 40_000)),
		nacelle.AssistantText("answer one"),
		nacelle.UserText("follow up"),
		nacelle.AssistantText("answer two"),
	}
}

// The assembler reports that the ledger was rewritten, which is the only way a
// session can say a ledger shrank rather than leaving the reader to infer it.
func TestApplyReportsAReplacedLedger(t *testing.T) {
	in := overdueLedgerConversation()
	summary := "Decisions:\n- keep `internal/compaction/ledger.go`"

	out, stats := Apply(in, Plan(in, applyPolicy()), summary, nil, true)

	if !stats.LedgerReplaced || stats.LedgerKept {
		t.Fatalf("stats = %+v, want the rewrite granted and reported", stats)
	}
	if body := Body(out[1]); body != summary {
		t.Errorf("ledger body = %q, want the consolidated rewrite", body)
	}
	if !reflect.DeepEqual(out[0], in[0]) {
		t.Errorf("anchor = %+v, want it verbatim", out[0])
	}
}

// And it reports the refusal, so a session can say the ledger stayed as it was
// written rather than claiming a consolidation that did not happen.
func TestApplyReportsARefusedLedgerRewrite(t *testing.T) {
	in := overdueLedgerConversation()

	out, stats := Apply(in, Plan(in, applyPolicy()), "Decisions:\n- consolidated", nil, true)

	if stats.LedgerReplaced || !stats.LedgerKept {
		t.Fatalf("stats = %+v, want the rewrite refused and reported", stats)
	}
	if body := Body(out[1]); !strings.Contains(body, "internal/compaction/ledger.go") {
		t.Errorf("ledger body = %q, want the merge to have kept the identifier", body)
	}
}
