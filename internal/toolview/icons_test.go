package toolview

import (
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

func TestToolKind(t *testing.T) {
	cases := []struct {
		name   string
		source nacelle.Source
		want   string
	}{
		{"read_file", nacelle.ToolSourceLocal, "read"},
		{"search_files", nacelle.ToolSourceLocal, "read"},
		{"run_command", nacelle.ToolSourceLocal, "write"},
		{"edit_file", nacelle.ToolSourceLocal, "write"},
		{"write_file", nacelle.ToolSourceLocal, "write"},
		{"web_fetch", nacelle.ToolSourceLocal, "network"},
		{"parallel_agents", nacelle.ToolSourceLocal, "delegate"},
		{"unknown_custom_tool", nacelle.ToolSourceLocal, "other"},
		{"some_mcp_tool", nacelle.ToolSourceMCP, "mcp"},
		{"read_file", nacelle.ToolSourceMCP, "mcp"},
	}

	for _, tc := range cases {
		if got := ToolKind(tc.name, tc.source); got != tc.want {
			t.Errorf("ToolKind(%q, %q) = %q, want %q", tc.name, tc.source, got, tc.want)
		}
	}
}

func TestToolKindGlyph(t *testing.T) {
	cases := []struct {
		kind string
		want string
	}{
		{"read", "⏺"},
		{"write", "⏺"},
		{"network", "⏺"},
		{"delegate", "⏺"},
		{"mcp", "✻"},
		{"other", "⏺"},
	}

	for _, tc := range cases {
		if got := ToolKindGlyph(tc.kind); got != tc.want {
			t.Errorf("ToolKindGlyph(%q) = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestToolGlyph(t *testing.T) {
	if got := ToolGlyph("run_command"); got != "$" {
		t.Errorf("ToolGlyph(run_command) = %q, want $", got)
	}
	if got := ToolGlyph("unknown"); got != "⏺" {
		t.Errorf("ToolGlyph(unknown) = %q, want ⏺", got)
	}
}

func TestToolRestore(t *testing.T) {
	if got := ToolRestore("run_command"); got != "35" {
		t.Errorf("ToolRestore(run_command) = %q, want 35", got)
	}
	if got := ToolRestore("unknown"); got != "34" {
		t.Errorf("ToolRestore(unknown) = %q, want 34", got)
	}
}

func TestColorGlyph(t *testing.T) {
	line := "$ run_command"
	colored := ColorGlyph(line, "32", "35")
	if !strings.Contains(colored, "\x1b[32m$\x1b[35m") {
		t.Errorf("ColorGlyph output %q missing expected escape codes", colored)
	}
}

func TestToolLinePainted(t *testing.T) {
	painted := ToolLinePainted("$ ls")
	if !strings.Contains(painted, "$ ls") {
		t.Errorf("ToolLinePainted = %q, want text included", painted)
	}
}
