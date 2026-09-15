package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestPromptWidthDoesNotOverflowTerminal(t *testing.T) {
	m := sized()
	m.resize(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.prompt.SetValue(strings.Repeat("a", 150))

	rows := strings.Split(m.prompt.View(), "\n")
	if len(rows) < 2 {
		t.Fatalf("prompt rows = %d, expected at least 2 wrapped rows", len(rows))
	}
	for i, row := range rows {
		if w := lipgloss.Width(row); w > 80 {
			t.Errorf("row %d width = %d, want <= 80", i, w)
		}
	}
}

func TestPromptCursorPositionAfterWrap(t *testing.T) {
	m := sized()
	m.resize(tea.WindowSizeMsg{Width: 80, Height: 24})

	for i := range 120 {
		m.promptRoute(tea.KeyPressMsg{Code: 'a', Text: "a"})
		view := m.View()
		if view.Cursor == nil {
			t.Fatalf("at length %d: view.Cursor is nil", i+1)
		}
		if view.Cursor.X < 2 {
			t.Errorf("at length %d: cursor X = %d, expected >= 2 (after gutter)", i+1, view.Cursor.X)
		}
	}
}

func TestTUIModeCursorPositionAfterWrap(t *testing.T) {
	m := tuiModel()
	m.resize(tea.WindowSizeMsg{Width: 80, Height: 24})

	for i := range 120 {
		m.promptRoute(tea.KeyPressMsg{Code: 'a', Text: "a"})
		view := m.View()
		if view.Cursor == nil {
			t.Fatalf("at length %d: view.Cursor is nil", i+1)
		}
		if view.Cursor.X < 2 {
			t.Errorf("at length %d: cursor X = %d, expected >= 2 (after gutter)", i+1, view.Cursor.X)
		}
	}
}
