package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/FacileStudio/kori/internal/toolview"
)

// modeInline prints finished lines into the terminal's own scrollback and
// draws the live region and prompt beneath them, Bubble Tea's standard render.
// modeTUI holds the transcript inside its own buffer and draws it, live region,
// and prompt on an alternate screen with the prompt pinned to the bottom, like
// htop or vim. renderMode turns the config's mode string into one of them;
// anything but "tui" is inline.
const (
	modeInline int = 0
	modeTUI    int = 1
)

// wheelStep is how many transcript rows one wheel notch pulls the tui-mode
// window back. Small, so a flick is not a page — the window re-anchors to the
// newest row with the prompt the moment the wheel comes back down.
const wheelStep = 3

func renderMode(name string) int {
	if name == "tui" {
		return modeTUI
	}
	return modeInline
}

// assembleView builds the per-frame view. Inline mode reuses the terminal's own
// scrollback and renders only the live region, prompt, and menu. TUI mode
// assembles the whole screen — held transcript, live region, prompt pinned to
// the bottom — and marks it for the alternate screen.
func (m *Model) assembleView() tea.View {
	if m.mode == modeTUI {
		return m.assembleTUI()
	}
	return m.assembleInline()
}

// assembleTUI is the alternate-screen render. Every frame it builds the whole
// screen: the held transcript and live region tailed to whatever rows the screen
// can spare, and the prompt pinned to the bottom with nothing beneath it and
// one blank row above, so it reads as a fixed input bar the way vim keeps its
// status line. The slash-command menu, when open, draws directly above the
// prompt and claims rows from the transcript's tail — it renders above
// everything else. There is no terminal scrollback to lean on inside an alternate
// screen, so finished lines are held in m.hold and drawn back here; when the
// transcript grows past the screen, the oldest rows scroll off rather than the
// prompt moving. The prompt's own top row is where its text and cursor live, so
// the cursor offset must land exactly there.
func (m *Model) assembleTUI() tea.View {
	prompt := highlightSkills(m.prompt.View(), m.skillTokens())
	menu := m.viewMenu()
	menuRows := 0
	if menu != "" {
		menuRows = lipgloss.Height(menu)
	}
	avail := max(m.windowHeight-1-lipgloss.Height(prompt)-menuRows-m.belowRows(), 1)
	parts := append(m.tuiUpper(avail), "")
	if menu != "" {
		parts = append(parts, menu)
	}
	parts = append(parts, prompt)
	if below := m.belowContent(); below != "" {
		parts = append(parts, "", below)
	}
	body := strings.Join(parts, "\n")
	m.frameRows = lipgloss.Height(body)

	view := tea.NewView(body)
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	if position := m.prompt.Cursor(); position != nil {
		position.Y += avail + 1 + menuRows
		view.Cursor = position
	}
	return view
}

// belowRows is the height the below-prompt block claims, blank separator
// included, so assembleTUI can reserve it from the transcript's share and the
// prompt stays pinned above it instead of the block being clipped off-screen.
func (m *Model) belowRows() int {
	if below := m.belowContent(); below != "" {
		return lipgloss.Height(below) + 1
	}
	return 0
}

// reflowHold re-wraps the alternate-screen transcript for the new width. The
// terminal reflows only lines it wrapped itself; rows this client painted
// earlier were wrapped at the width of their moment, and a shrink would clip
// them. Every row wider than the pane is broken again here, continuations
// carrying the border glyph when the row they continue opened with one, so a
// box's spine reads down its whole wrapped length.
func (m *Model) reflowHold() {
	limit := max(m.width-1, 1)
	reflowed := make([]string, 0, len(m.hold))
	for _, row := range m.hold {
		reflowed = append(reflowed, toolview.WrapRow(row, limit)...)
	}
	m.hold = reflowed
}

// tuiUpper is the scrolling region above the pinned prompt: the held transcript
// and the run's live output, newest at the bottom, padded to exactly avail rows
// so the prompt stays anchored. Each held entry is one pre-wrapped row — the
// split happened at print time — so joining the already-painted rows is enough;
// what the terminal clips horizontally, it clips. The hold is tailed to avail
// before the live rows join, since only the combined tail can reach the screen.
// aboveContent is split on newlines first, because it can hand over a multi-line
// string (the status line is always two rows): an appended entry must be one
// visual row or the len()-based count understates the padded height, pushing
// the prompt a row off the bottom and the cursor a row off the text. Blank
// separator rows are kept, not dropped, so the status line keeps its breathing
// room: without them the spinner row sat flush against the content above no
// matter how the live region was spaced. The trailing blank aboveContent passes
// over when the menu is closed is stripped here so it cannot double with the
// prompt's own separator row and push the pinned cursor a row down.
//
// full runs oldest to newest — the held transcript (top) then the live region
// (bottom) — and window drops scrollTop from the newest row before the tail, so
// the wheel can surface earlier lines above the live region; at scrollTop 0 it
// is exactly the newest-avail tail no scroll can show differently. Rows are
// padded up top to exactly avail so the prompt stays anchored whether the
// window reaches the top of the scrollback or not.
func (m *Model) tuiUpper(avail int) []string {
	live := make([]string, 0, len(m.hold)+avail)
	for _, row := range m.aboveContent() {
		live = append(live, strings.Split(row, "\n")...)
	}
	for len(live) > 0 && live[len(live)-1] == "" {
		live = live[:len(live)-1]
	}
	full := append(append([]string{}, m.hold...), live...)
	rows := window(full, avail, m.scrollTop)
	var padded []string
	for pad := avail - len(rows); pad > 0; pad-- {
		padded = append(padded, "")
	}
	return append(padded, rows...)
}

// window slices content down to its last-avail rows moved back by scroll lines,
// clamped so it never goes past the top of the content or below the newest row.
func window(content []string, avail, scroll int) []string {
	if len(content) <= avail {
		return content
	}
	if scroll < 0 {
		scroll = 0
	}
	if maxScroll := len(content) - avail; scroll > maxScroll {
		scroll = maxScroll
	}
	return content[len(content)-avail-scroll : len(content)-scroll]
}

// scrollWheel moves the tui-mode transcript window in response to the wheel and
// always claims the message, so a wheel has never fallen through to the prompt
// as history. Inline mode never requests the mouse — the terminal scrolls its
// own scrollback there — and returns no Cmd either way.
func (m *Model) scrollWheel(msg tea.MouseWheelMsg) tea.Cmd {
	if m.mode == modeTUI {
		switch msg.Button {
		case tea.MouseWheelUp:
			m.scrollTop += wheelStep
		case tea.MouseWheelDown:
			m.scrollTop -= wheelStep
		}
		if m.scrollTop < 0 {
			m.scrollTop = 0
		}
	}
	return nil
}

// assembleInline is the scrollback render: everything already said lives in the
// terminal's own history, and the view draws only the live region, the prompt,
// and the menu above the prompt. The prompt is separated from the content above and
// below by one blank row each, so it reads as its own band. aboveContent
// already ends in a blank row when the menu is closed, so the separator is only
// added when the content does not already breathe.
