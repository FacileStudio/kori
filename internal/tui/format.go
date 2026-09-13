package tui

import (
	"fmt"
	"strings"
)

// question draws the reader's question with the left half-block spine (▌) on
// every row, wrapped or not: the body renders at two columns short of the
// width so each line the style breaks has room for the prefix, and the prefix
// is added after rendering so wrapped rows carry it too.
func (m *Model) question(text string, width int) string {
	body := m.theme.Question.Width(max(width-2, 1)).Render(text)
	spines := m.theme.Question.Render("▌ ")
	tokens := m.skillTokens()
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		lines[i] = spines + highlightSkills(line, tokens)
	}
	return strings.Join(lines, "\n")
}

// countedNoun formats a count with its English plural, singular for exactly
// one.
func countedNoun(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
