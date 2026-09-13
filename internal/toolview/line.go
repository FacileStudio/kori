// Package toolview formats and renders tool calls, icons, and status lines.
package toolview

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/FacileStudio/nacelle"
	"github.com/charmbracelet/x/ansi"
)

var primaryKeys = []string{"path", "file_path", "file", "command", "pattern", "query", "url", "name"}

// DurationRoom is the number of cells reserved for the elapsed call duration.
const DurationRoom = 10

func truncate(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	return ansi.Truncate(s, limit, "…")
}

// ToolLine formats a tool call as a single line with name and primary argument.
func ToolLine(name, input string, width int) string {
	return toolLine(name, input, width, "")
}

// ToolLineSource formats a tool call as a single line with name and primary argument,
// choosing the glyph from the tool's nacelle source.
func ToolLineSource(name, input string, width int, source nacelle.Source) string {
	return toolLine(name, input, width, source)
}

func toolLine(name, input string, width int, source nacelle.Source) string {
	room := width - lipgloss.Width(name) - DurationRoom - len("⏺ ()")
	glyph := ToolSourceGlyph(name, source)
	return glyph + " " + name + "(" + truncate(ansi.Strip(PrimaryArg(input)), room) + ")"
}

// PrimaryArg extracts the most identifying argument from tool input.
func PrimaryArg(input string) string {
	fields, err := StrictObject([]byte(input))
	if errors.Is(err, ErrDuplicateKey) {
		return "input has a duplicate key"
	}
	if err != nil || len(fields) == 0 {
		return ""
	}

	for _, key := range primaryKeys {
		if raw, ok := fields[key]; ok {
			return oneLine(raw)
		}
	}
	if len(fields) == 1 {
		for _, raw := range fields {
			return oneLine(raw)
		}
	}
	return ""
}

func oneLine(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.Join(strings.Fields(text), " ")
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return ""
	}
	return strings.Join(strings.Fields(compact.String()), " ")
}
