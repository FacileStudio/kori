package toolview

import (
	"strings"
	"testing"
)

func TestHookLineFormatting(t *testing.T) {
	line := HookLine("after_tool_call", "edit_file", "graft blast --no-refresh", 80)
	if !strings.HasPrefix(line, "⟡ hook:after_tool_call[edit_file](") {
		t.Errorf("got %q, want prefix hook:after_tool_call[edit_file]", line)
	}
	if !strings.Contains(line, "graft blast --no-refresh") {
		t.Errorf("got %q, want command in line", line)
	}

	sessionStart := HookLine("session_start", "", "graft map --no-refresh", 80)
	if !strings.HasPrefix(sessionStart, "⟡ hook:session_start(") {
		t.Errorf("got %q, want prefix hook:session_start(", sessionStart)
	}
}

func TestHookOutputPreviewFormatting(t *testing.T) {
	empty := HookOutputPreview("", 4, 80)
	if empty != "" {
		t.Errorf("got %q, want empty", empty)
	}

	single := HookOutputPreview("single line output", 4, 80)
	if single != "single line output" {
		t.Errorf("got %q, want 'single line output'", single)
	}

	multiline := HookOutputPreview("line1\nline2\nline3\nline4\nline5", 3, 80)
	if !strings.Contains(multiline, "line1\n  line2\n  line3\n  … more") {
		t.Errorf("got %q, want truncation with … more", multiline)
	}
}
