package compaction

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/FacileStudio/nacelle"
)

// historySample is a well-formed conversation with two large tool results: one
// in history, one in the active window, so a pass can be checked for touching
// only the former.
func historySample() []nacelle.Message {
	return []nacelle.Message{
		nacelle.UserText("the task"),
		callMessage("c1", "read"),
		resultMessage("c1", "read", strings.Repeat("x", 40_000)),
		nacelle.AssistantText("answer one"),
		nacelle.UserText("follow up"),
		callMessage("c2", "read"),
		resultMessage("c2", "read", strings.Repeat("y", 30_000)),
		nacelle.AssistantText("answer two"),
	}
}

// splitSpans is the history/active split of historySample: [1,5) is history,
// [5,8) is the verbatim active window.
func splitSpans() []Span {
	return []Span{{Zone: ZoneHistory, Start: 1, End: 5}, {Zone: ZoneActive, Start: 5, End: 8}}
}

func TestTombstoneDropsOnlyHistoryResults(t *testing.T) {
	conv := historySample()

	stats := Tombstone(conv, splitSpans())

	if stats.Results != 1 || stats.Bytes != 40_000 {
		t.Fatalf("stats = %+v, want the one 40000-byte history result", stats)
	}
	if result := conv[2].Parts[0].(nacelle.ToolResult); !strings.HasPrefix(result.Result, DroppedNotice) {
		t.Errorf("history result = %q, want a placeholder", result.Result)
	}
	if result := conv[6].Parts[0].(nacelle.ToolResult); strings.HasPrefix(result.Result, DroppedNotice) {
		t.Error("the active window's result was tombstoned, want it kept verbatim")
	}
}

func TestTombstoneIsIdempotent(t *testing.T) {
	conv := historySample()

	Tombstone(conv, splitSpans())
	second := Tombstone(conv, splitSpans())

	if second != (MicroStats{}) {
		t.Errorf("second pass = %+v, want no stub and no debit", second)
	}
}

func TestTombstoneKeepsThePairingShape(t *testing.T) {
	conv := historySample()
	before := historySample()

	Tombstone(conv, splitSpans())

	for i := range conv {
		if len(conv[i].Parts) != len(before[i].Parts) {
			t.Fatalf("message %d changed shape: %d parts, was %d", i, len(conv[i].Parts), len(before[i].Parts))
		}
		for j := range conv[i].Parts {
			was, ok := before[i].Parts[j].(nacelle.ToolResult)
			now, still := conv[i].Parts[j].(nacelle.ToolResult)
			if ok != still {
				t.Fatalf("message %d part %d changed kind", i, j)
			}
			if ok && now.ID != was.ID {
				t.Errorf("message %d part %d: id %q, was %q — the pairing broke", i, j, now.ID, was.ID)
			}
		}
	}
}

// A call's text is its name and its own arguments, so two reads of different
// files are not the same text. A long argument list is cut and marked, so a
// reader can tell an abbreviated block from a complete one — and a cut through a
// multi-byte rune is trimmed rather than emitted broken, since this text lands in
// a JSON request body.
func TestCallTextCarriesTheArgumentsAndMarksTheCut(t *testing.T) {
	call := func(input string) nacelle.ToolCall {
		return nacelle.ToolCall{ID: "c1", Name: "read", Input: json.RawMessage(input), Finished: true}
	}

	if got := callText(call(`{"path":"main.go"}`), maxBlockInput); got != `read {"path":"main.go"}` {
		t.Errorf("callText = %q, want the name and its arguments", got)
	}
	if got := callText(call(""), maxBlockInput); got != "read" {
		t.Errorf("callText = %q, want the bare name for a call with no arguments", got)
	}

	long := call(`{"path":"` + strings.Repeat("x", maxBlockInput+50) + `"}`)
	if got := callText(long, maxBlockInput); !strings.HasSuffix(got, "…") {
		t.Errorf("callText = %q, want a marked cut at the cap", got)
	}

	runes := call(strings.Repeat("é", maxBlockInput))
	if cut := callText(runes, maxBlockInput-1); !utf8.ValidString(cut) {
		t.Errorf("callText cut a multi-byte rune in half: %q", cut)
	}
}

func TestTombstoneLeavesSmallResultsAlone(t *testing.T) {
	conv := []nacelle.Message{nacelle.UserText("task"), resultMessage("c", "read", strings.Repeat("x", MinResult-1))}

	if stats := Tombstone(conv, []Span{{Zone: ZoneHistory, Start: 1, End: 2}}); stats.Results != 0 {
		t.Errorf("stats = %+v, want a result under the floor left alone", stats)
	}
}

// Reasoning is recorded and displayed but never sent — every backend drops it
// when it builds a request — so the deterministic pass leaves it alone. A stub
// on it would free no context while editing a transcript the reader can still
// scroll back to, and it would count as savings in the pass's own report.
func TestTombstoneLeavesHistoryReasoningAlone(t *testing.T) {
	thought := strings.Repeat("t", 5000)
	conv := []nacelle.Message{
		nacelle.UserText("task"),
		{Role: nacelle.RoleAssistant, Parts: []nacelle.Part{nacelle.Reasoning{Text: thought}, nacelle.Text{Text: "conclusion"}}},
		{Role: nacelle.RoleAssistant, Parts: []nacelle.Part{nacelle.Reasoning{Text: strings.Repeat("u", 50)}, nacelle.Text{Text: "recent"}}},
	}
	spans := []Span{{Zone: ZoneHistory, Start: 1, End: 2}, {Zone: ZoneActive, Start: 2, End: 3}}

	stats := Tombstone(conv, spans)

	if stats != (MicroStats{}) {
		t.Errorf("stats = %+v, want nothing replaced and nothing credited", stats)
	}
	if reasoning := conv[1].Parts[0].(nacelle.Reasoning); reasoning.Text != thought {
		t.Errorf("history reasoning = %q, want it kept verbatim", reasoning.Text)
	}
	if text := conv[1].Parts[1].(nacelle.Text); text.Text != "conclusion" {
		t.Errorf("assistant text = %q, want it preserved", text.Text)
	}
}
