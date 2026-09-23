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

// A reversed span — one whose end sits before its start — is the same kind of
// disagreement: the low bound of the slice would exceed the high one, which is a
// panic rather than the empty read the span describes. A span that does describe
// messages is still read, so the guard skips the reversed one alone.
func TestSectionSkipsAReversedSpan(t *testing.T) {
	conv := []nacelle.Message{nacelle.UserText("task"), nacelle.AssistantText("one")}
	reversed := []Span{{Zone: ZoneHistory, Start: 1, End: 0}}
	both := []Span{{Zone: ZoneHistory, Start: 1, End: 0}, {Zone: ZoneHistory, Start: 0, End: 2}}

	if got := Section(conv, reversed, ZoneHistory); len(got) != 0 {
		t.Errorf("Section = %v, want nothing for a reversed span", got)
	}
	if got := Section(conv, both, ZoneHistory); len(got) != len(conv) {
		t.Errorf("Section = %v, want the one span that really covers messages", got)
	}
}

// The ledger index is the other read with no bound of its own: an index the
// conversation does not have is not a ledger, so it has no replies to claim and
// the end is the index itself, rather than a scan starting off the end of it.
func TestLedgerEndLeavesAnIndexTheConversationDoesNotHaveAlone(t *testing.T) {
	conv := []nacelle.Message{nacelle.UserText("task")}

	if got := LedgerEnd(conv, 4, 9); got != 4 {
		t.Errorf("LedgerEnd(_, 4, 9) = %d, want the index itself", got)
	}
	if got := LedgerEnd(conv, -1, 9); got != -1 {
		t.Errorf("LedgerEnd(_, -1, 9) = %d, want the index itself", got)
	}
	if got := LedgerEnd(conv, 0, 9); got != 1 {
		t.Errorf("LedgerEnd(_, 0, 9) = %d, want the one message it covers", got)
	}
}
