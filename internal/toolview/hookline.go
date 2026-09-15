package toolview

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// HookLine formats a hook event header line with point, tool, and command.
func HookLine(event, tool, command string, width int) string {
	name := "hook:" + event
	if tool != "" {
		name += "[" + tool + "]"
	}
	room := width - lipgloss.Width(name) - DurationRoom - len("⟡ ()")
	return "⟡ " + name + "(" + truncate(ansi.Strip(command), room) + ")"
}

// HookOutputPreview formats an indented preview of hook output.
func HookOutputPreview(output string, maxLines, width int) string {
	clean := strings.TrimSpace(output)
	if clean == "" {
		return ""
	}
	lines := strings.Split(clean, "\n")
	limit := maxLines
	if limit <= 0 {
		limit = 6
	}
	truncated := len(lines) > limit
	if truncated {
		lines = lines[:limit]
	}
	indentOutputLines(lines, width)
	res := strings.Join(lines, "\n")
	if truncated {
		res += "\n  … more"
	}
	return res
}

func indentOutputLines(lines []string, width int) {
	for i, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if width > 6 {
			trimmed = truncate(trimmed, width-6)
		}
		if i > 0 {
			lines[i] = "  " + trimmed
		} else {
			lines[i] = trimmed
		}
	}
}
