package compaction

import (
	"testing"

	"github.com/FacileStudio/nacelle"
)

// AlignedCut never splits an assistant ToolCall message from the user ToolResult
// answering it: the kept tail must not open with a result whose call was cut.
func TestAlignedCutNeverSplitsAToolPair(t *testing.T) {
	conv := []nacelle.Message{
		nacelle.UserText("setup"),
		nacelle.AssistantText("intro"),
		nacelle.UserText("run the tool"),
		callMessage("call_1", "read"),
		resultMessage("call_1", "read", "file contents"),
		nacelle.UserText("keep going"),
		nacelle.AssistantText("done"),
	}

	tests := []struct {
		cut, want int
	}{
		{cut: 4, want: 3},
		{cut: 3, want: 3},
		{cut: 5, want: 5},
		{cut: 2, want: 2},
	}
	for _, tc := range tests {
		if got := AlignedCut(conv, tc.cut); got != tc.want {
			t.Errorf("AlignedCut(_, %d) = %d, want %d", tc.cut, got, tc.want)
		}
	}
}

// Covers is the precondition Apply checks before it assembles, and it is exact:
// spans that tile the conversation pass, anything else — a gap, an overlap, a
// span past the end — does not.
func TestCoversAcceptsOnlyAPlanThatTilesTheConversation(t *testing.T) {
	conv := []nacelle.Message{nacelle.UserText("task"), nacelle.AssistantText("one"), nacelle.UserText("two")}
	tiled := []Span{{Zone: ZoneAnchor, Start: 0, End: 1}, {Zone: ZoneActive, Start: 1, End: 3}}

	tests := []struct {
		name  string
		spans []Span
		want  bool
	}{
		{"a tiled plan covers", tiled, true},
		{"a span past the end does not", []Span{{Zone: ZoneAnchor, Start: 0, End: 9}}, false},
		{"a gap does not", []Span{{Zone: ZoneAnchor, Start: 0, End: 1}, {Zone: ZoneActive, Start: 2, End: 3}}, false},
		{"an empty plan covers nothing", nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Covers(conv, tc.spans); got != tc.want {
				t.Errorf("Covers(%v) = %v, want %v", tc.spans, got, tc.want)
			}
		})
	}
	if !Covers(conv, Plan(conv, Policy{AnchorMessages: 1, KeepTurns: 1})) {
		t.Error("Plan did not produce a covering plan, so every pass would be refused")
	}
}

func TestAlignedCutLeavesOutOfRangeCutsAlone(t *testing.T) {
	conv := []nacelle.Message{resultMessage("c", "read", "x")}
	if got := AlignedCut(conv, 0); got != 0 {
		t.Errorf("AlignedCut(_, 0) = %d, want 0", got)
	}
	if got := AlignedCut(conv, -1); got != -1 {
		t.Errorf("AlignedCut(_, -1) = %d, want -1", got)
	}
	if got := AlignedCut(conv, 5); got != 5 {
		t.Errorf("AlignedCut(_, 5) past the end = %d, want 5", got)
	}
}
