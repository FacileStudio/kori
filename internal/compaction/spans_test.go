package compaction

import (
	"testing"

	"github.com/FacileStudio/nacelle"
)

// A selector is the wrong place to discover that a plan and a conversation
// disagree: a span reaching past the end is trimmed rather than trusted, so a
// stale plan degrades into an empty section instead of an out-of-range panic.
func TestSelectorsTrimASpanPastTheEnd(t *testing.T) {
	conv := []nacelle.Message{nacelle.UserText("task")}
	spans := []Span{{Zone: ZoneHistory, Start: 1, End: 9}, {Zone: ZoneLedger, Start: 4, End: 5}}

	if got := Section(conv, spans, ZoneHistory); len(got) != 0 {
		t.Errorf("Section = %v, want nothing for a span past the end", got)
	}
	if got := LedgerText(conv, spans); got != "" {
		t.Errorf("LedgerText = %q, want no ledger for a span past the end", got)
	}
}
