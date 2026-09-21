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
