package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// assembleInline is the scrollback render: everything already said lives in the
// terminal's own history, and the view draws only the live region, the prompt,
// and the menu above the prompt. The prompt is separated from the content above and
// below by one blank row each, so it reads as its own band. aboveContent
// already ends in a blank row when the menu is closed, so the separator is only
// added when the content does not already breathe.
func (m *Model) assembleInline() tea.View {
	above := m.aboveContent()
	aboveHeight := lipgloss.Height(strings.Join(above, "\n"))
	parts := above
	if last := len(parts) - 1; last < 0 || parts[last] != "" {
		parts = append(parts, "")
	}
	if menu := m.viewMenu(); menu != "" {
		parts = append(parts, menu)
	}
	parts = append(parts, m.prompt.View())
	if below := m.belowContent(); below != "" {
		parts = append(parts, "", below)
	}
	body := strings.Join(parts, "\n")
	m.frameRows = lipgloss.Height(body)

	view := tea.NewView(body)
	menuRows := 0
	if m.viewMenu() != "" {
		menuRows = m.menu.Height()
	}
	if position := m.prompt.Cursor(); position != nil {
		position.Y += aboveHeight + 1 + menuRows
		view.Cursor = position
	}
	return view
}
