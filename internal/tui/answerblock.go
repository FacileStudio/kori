package tui

import "strings"

import "github.com/FacileStudio/nacelle-tui/internal/toolview"

// answerBlock paints one committed answer chunk as a pane of its own: the
// markdown rendered two columns short, each row padded onto the block
// backdrop (or left on the terminal's own background when transparent blocks
// are on), and the whole thing wrapped by the same left-bordered box the
// tool panes draw, with a blue spine. A blank row above and below keeps
// neighbouring blocks and the question pane from touching it. The two
// borrowed columns are why restyle builds the renderer at width-2.
//
// It lives in its own file because it is the only paint arm that reaches
// into the box renderer rather than handing off a string, and the file it
// would otherwise share a home with is already at the line cap for other
// reasons.
func (m *Model) answerBlock(text string, width int) string {
	inner := max(width-2, 1)
	rows := strings.Split(m.markdown(text), "\n")
	for i, row := range rows {
		rows[i] = toolview.MatchBackground(m.theme.Plain, m.transparent).Width(inner).Render(row)
	}
	return "\n" + toolview.Box(rows, "4", m.transparent, width) + "\n"
}
