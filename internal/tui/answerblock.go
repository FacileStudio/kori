package tui

import "strings"

import "github.com/FacileStudio/nacelle-tui/internal/toolview"

// answerPane paints the answer as a pane: the markdown rendered two columns
// short, each row padded onto the block backdrop (or left on the terminal's
// own background when transparent blocks are on), and the whole thing wrapped
// by the same left-bordered box the tool panes draw, with a blue spine. It is
// shared by the committed block and the streaming region, so the spine is on
// the pane from the first streamed delta, not only once the answer finishes.
// The two borrowed columns are why restyle builds the renderer at width-2.
func (m *Model) answerPane(text string, width int) string {
	inner := max(width-2, 1)
	rows := strings.Split(m.markdown(text), "\n")
	for i, row := range rows {
		rows[i] = toolview.MatchBackground(m.theme.Plain, m.transparent).Width(inner).Render(row)
	}
	return toolview.Box(rows, "4", m.transparent, width)
}

// answerBlock paints one committed answer chunk as a pane of its own. A blank
// row above and below keeps neighbouring blocks and the question pane from
// touching it.
//
// It lives in its own file because it is the only paint arm that reaches
// into the box renderer rather than handing off a string, and the file it
// would otherwise share a home with is already at the line cap for other
// reasons.
func (m *Model) answerBlock(text string, width int) string {
	return "\n" + m.answerPane(text, width) + "\n"
}
