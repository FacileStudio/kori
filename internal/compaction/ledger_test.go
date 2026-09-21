package compaction

import (
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

func TestIsLedgerRecognizesTheSentinel(t *testing.T) {
	ledger := BuildLedger("", "Decisions:\n- ship it")
	if !IsLedger(ledger) {
		t.Fatalf("IsLedger(BuildLedger(...)) = false, want true")
	}
	if IsLedger(nacelle.UserText("[compacted context] the old marker")) {
		t.Error("IsLedger matched the retired compacted-context marker")
	}
	if IsLedger(nacelle.AssistantText("an ordinary turn")) {
		t.Error("IsLedger matched an ordinary turn")
	}
}

// A ledger built from an earlier one always contains it: a later pass can fold
// new facts in but can never quietly drop an older one.
func TestBuildLedgerFoldsMonotonically(t *testing.T) {
	first := Body(BuildLedger("", "decided A"))
	second := Body(BuildLedger(first, "learned B"))
	if !strings.Contains(second, "decided A") {
		t.Errorf("ledger = %q, want the earlier fact preserved", second)
	}
	if !strings.Contains(second, "learned B") {
		t.Errorf("ledger = %q, want the new fact folded in", second)
	}
}

// Folding a ledger into itself is a no-op, so a pass with nothing new cannot
// duplicate the content it already carries.
func TestBuildLedgerIsIdempotent(t *testing.T) {
	once := Body(BuildLedger("", "decided A\nlearned B"))
	if twice := Body(BuildLedger(once, once)); twice != once {
		t.Errorf("folding a ledger into itself changed it:\n%q\n->\n%q", once, twice)
	}
	if summary := Body(BuildLedger(once, "learned B")); summary != once {
		t.Errorf("refolding a subset changed the ledger:\n%q\n->\n%q", once, summary)
	}
}

func TestBuildLedgerCarriesTheSentinelAndBody(t *testing.T) {
	block := BuildLedger("older", "newer")
	text, ok := block.Parts[0].(nacelle.Text)
	if !ok || !strings.HasPrefix(text.Text, Sentinel) {
		t.Fatalf("ledger = %q, want the sentinel prefix", text.Text)
	}
	if !strings.Contains(text.Text, "newer") || !strings.Contains(text.Text, "older") {
		t.Errorf("ledger = %q, want both halves folded in", text.Text)
	}
}
