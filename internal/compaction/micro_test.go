package compaction

import (
	"encoding/json"
	"reflect"
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

// The placeholder is what the model reads in place of the result, so it has to
// read as one balanced sentence that says what happened and what to do about it,
// not as a truncated result. It keeps the call's id and name, which is what keeps
// the pairing readable.
func TestTombstonePlaceholderSaysWhatItReplaced(t *testing.T) {
	conv := historySample()

	Tombstone(conv, splitSpans())

	result := conv[2].Parts[0].(nacelle.ToolResult)
	want := "[dropped 40000 bytes] Re-run the tool if the detail matters."
	if result.Result != want {
		t.Errorf("placeholder = %q, want %q", result.Result, want)
	}
	if result.ID != "c1" || result.Name != "read" {
		t.Errorf("placeholder = %+v, want the call's id and name kept", result)
	}
}

// DroppableBytes is the pre-flight a caller runs before it pays for a cache
// invalidation: it reports exactly what Tombstone would take out, and takes
// nothing out itself.
func TestDroppableBytesMeasuresWithoutChanging(t *testing.T) {
	conv := historySample()
	before := historySample()

	if got := DroppableBytes(conv, splitSpans()); got != 40_000 {
		t.Errorf("DroppableBytes = %d, want the one oversized history result", got)
	}
	if !reflect.DeepEqual(conv, before) {
		t.Error("DroppableBytes changed the conversation, want a measurement only")
	}

	stats := Tombstone(conv, splitSpans())
	if stats.Bytes != 40_000 {
		t.Errorf("Tombstone freed %d bytes, want the 40000 DroppableBytes promised", stats.Bytes)
	}
	if got := DroppableBytes(conv, splitSpans()); got != 0 {
		t.Errorf("DroppableBytes after the pass = %d, want nothing left to drop", got)
	}
}

// MinCleared is the floor under a whole pass and MinResult the floor under one
// result, so the two are different questions and the pass floor has to be the
// larger one or it could never gate anything.
func TestMinClearedIsAboveASingleResult(t *testing.T) {
	if MinCleared <= MinResult {
		t.Errorf("MinCleared = %d, want it above the %d one result needs", MinCleared, MinResult)
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

// The byte weight is the weight a request carries, so a part counts when it is
// sent and not when it is not. Reasoning never leaves the client, so counting it
// would report savings no pass can make; a tool call's own arguments always do,
// and leaving them out understated every edit and write turn — which is where a
// long session's bytes mostly are.
func TestPartBytesCountsWhatARequestCarries(t *testing.T) {
	call := nacelle.ToolCall{ID: "c1", Name: "edit_file", Input: json.RawMessage(`{"path":"internal/tui/run.go"}`)}
	tests := []struct {
		name string
		part nacelle.Part
		want int
	}{
		{"spoken text is sent", nacelle.Text{Text: "hello"}, 5},
		{"a tool result is sent", nacelle.ToolResult{Result: "contents"}, 8},
		{"a tool call's arguments are sent", call, len(call.Input)},
		{"reasoning is never sent", nacelle.Reasoning{Text: strings.Repeat("t", 5_000)}, 0},
		{"a finish marker carries nothing", nacelle.Finish{}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := PartBytes(tc.part); got != tc.want {
				t.Errorf("PartBytes = %d, want %d", got, tc.want)
			}
		})
	}
}
