package tui

// Tests for the ledger's own economy: a body that has outgrown one summary is
// rewritten rather than added to. What a merge does line by line lives in
// package compaction; what is tested here is the session's half — when the pass
// asks for a rewrite, what the summarizer is told, and what the report says
// happened when it did or would not.

import (
	"strings"
	"testing"

	"github.com/FacileStudio/kori/internal/compaction"
	"github.com/FacileStudio/nacelle"
)

// ledgerIdentifier is the one load-bearing token in the fixture ledger, kept in
// both halves of the test so a summary that drops it is easy to write.
const ledgerIdentifier = "`internal/compaction/ledger.go`"

func longLedgerBody() string {
	return "Decisions:\n- keep " + ledgerIdentifier + "\nState:\n" + strings.Repeat("- recorded fact\n", 600)
}

// overdueLedger is a conversation already carrying a ledger past its budget,
// with a heavy history behind it: something worth consolidating and something
// worth folding, which is the shape a rewrite is legitimate for.
func overdueLedger() []nacelle.Message {
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		compaction.BuildLedger("", longLedgerBody()),
	}
	return append(conv, heavyHistory()...)
}

// The pass decides for itself, off the ledger the conversation actually holds:
// over budget means this pass may rewrite, under it means it may only add.
func TestPassDecidesToConsolidateAnOverBudgetLedger(t *testing.T) {
	tests := []struct {
		name string
		conv []nacelle.Message
		want bool
	}{
		{"a ledger past its budget", overdueLedger(), true},
		{"a ledger still under it", func() []nacelle.Message {
			return append([]nacelle.Message{nacelle.UserText("the task"), compaction.BuildLedger("", "Decisions:\n- keep "+ledgerIdentifier)}, heavyHistory()...)
		}(), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := sized()
			m.conversation = tc.conv

			if got := m.pass(m.plan(), compaction.Mid).consolidate; got != tc.want {
				t.Errorf("pass.consolidate = %v, want %v", got, tc.want)
			}
		})
	}
}

// The addendum that carries the earlier ledger says why it is there, because the
// natural reading of "here is what you recorded before" is to restate it — and a
// restatement is what doubled the body on every pass.
func TestCompactPromptTellsTheSummarizerNotToRepeatTheLedger(t *testing.T) {
	m := sized()
	m.conversation = overdueLedger()
	plan := m.plan()
	fold := compaction.Fold{Ledger: compaction.Blocks(m.conversation, plan)}

	adding := compactPrompt(m.conversation, plan, fold, false)
	if !promptText(adding, "do not repeat anything it holds") {
		t.Error("prompt = no no-restatement rule, want the earlier ledger carried with one")
	}
	if !promptText(adding, longLedgerBody()[:40]) {
		t.Error("prompt = no earlier ledger, want it carried in the ask")
	}

	rewriting := compactPrompt(m.conversation, plan, fold, true)
	if !promptText(rewriting, "rewritten rather than added to") {
		t.Error("prompt = no consolidating instruction, want the rewrite asked for explicitly")
	}
	if promptText(rewriting, "do not repeat anything it holds") {
		t.Error("prompt = the add-only rule, want the two instructions to be alternatives")
	}
}

// A consolidating pass installs the rewrite, says so, and the body really is the
// rewrite rather than the old one plus it.
func TestSettleCompactionConsolidatesAnOverBudgetLedger(t *testing.T) {
	m := sized()
	m.conversation = overdueLedger()
	m.size = 130_000
	plan := m.plan()
	summary := "Decisions:\n- keep " + ledgerIdentifier + " — consolidated"

	m.settleCompaction(compactOutcome{
		before:      m.size,
		plan:        plan,
		fold:        compaction.Fold{Ledger: compaction.Blocks(m.conversation, plan)},
		tier:        compaction.Hard,
		summary:     summary,
		consolidate: true,
	})

	if !m.last.replaced {
		t.Fatalf("last = %+v, want the ledger reported as consolidated", m.last)
	}
	if said := strings.Join(spoken(m), "\n"); !strings.Contains(said, "consolidated the ledger") {
		t.Errorf("report = %q, want the consolidation named", said)
	}
	got, ok := ledgerIn(m.conversation)
	if !ok {
		t.Fatal("conversation = no ledger, want one installed")
	}
	if got != summary {
		t.Errorf("ledger body = %q, want the consolidated rewrite", got)
	}
}

// A rewrite that drops what the ledger knew is not installed: the merge stands,
// and the report says the rewrite was refused rather than pretending it landed.
func TestSettleCompactionKeepsTheLedgerWhenARewriteDropsAnIdentifier(t *testing.T) {
	m := sized()
	m.conversation = overdueLedger()
	m.size = 130_000
	plan := m.plan()

	m.settleCompaction(compactOutcome{
		before:      m.size,
		plan:        plan,
		fold:        compaction.Fold{Ledger: compaction.Blocks(m.conversation, plan)},
		tier:        compaction.Hard,
		summary:     "Decisions:\n- consolidated, see the notes",
		consolidate: true,
	})

	if m.last.replaced || !m.last.keptAsIs {
		t.Fatalf("last = %+v, want the rewrite refused and the ledger kept", m.last)
	}
	got, _ := ledgerIn(m.conversation)
	if !strings.Contains(got, ledgerIdentifier) {
		t.Errorf("ledger body = %q, want the identifier the merge kept", got)
	}
	if said := strings.Join(spoken(m), "\n"); !strings.Contains(said, "rewrite dropped an identifier") {
		t.Errorf("report = %q, want the refusal named", said)
	}
}

// ledgerIn is the body of the conversation's ledger message, if it has one.
func ledgerIn(conv []nacelle.Message) (string, bool) {
	for _, msg := range conv {
		if compaction.IsLedger(msg) {
			return compaction.Body(msg), true
		}
	}
	return "", false
}
