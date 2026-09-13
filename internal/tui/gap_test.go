package tui

import (
	"strings"
	"testing"
)

func TestGapBetweenPanesCarriesSpine(t *testing.T) {
	m := sized()
	pane := strings.Trim(m.answerBlock("text", m.width), "\n")
	if g := m.gap(pane, pane); !strings.Contains(g, "▌") {
		t.Errorf("gap between two panes = %q, want a spine row", g)
	}
	if g := m.gap(pane, "plain"); g != "\n\n" {
		t.Errorf("gap between pane and plain = %q, want blank line", g)
	}
}
