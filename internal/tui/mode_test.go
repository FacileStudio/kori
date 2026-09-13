package tui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/charmbracelet/x/ansi"
)

// tuiModel is a model switched into alternate-screen mode with a window, the
// arrangement the mode setting produces in Launch.
func tuiModel() *Model {
	m := NewModel(nil, "banner", nil, SessionConfig{CompactAt: 100_000, PromptPlaceholder: "placeholder"})
	m.mode = modeTUI
	m.resize(tea.WindowSizeMsg{Width: 80, Height: 24})
	return m
}

// renderMode is the one seam the mode string crosses: the config names it,
// this names the renderer. Anything that is not "tui" is inline by design.
func TestRenderModeParsesTheString(t *testing.T) {
	if renderMode("tui") != modeTUI {
		t.Error("renderMode(\"tui\") != modeTUI")
	}
	if renderMode("inline") != modeInline {
		t.Error("renderMode(\"inline\") != modeInline")
	}
	if renderMode("bogus") != modeInline {
		t.Error("an unknown mode should fall back to inline")
	}
}

// Inside the alternate screen there is no terminal scrollback for Println to
// reach, so a finished line is held in memory and drawn back into the view, and
// nothing is handed to tea as a Cmd. The same say-path must produce a printed
// command in the inline mode it shares the queue with.
func TestTUIModeHoldsPrintedLinesInsteadOfHandingThemToTheTerminal(t *testing.T) {
	m := tuiModel()
	m.say(fromReader, "a question")

	if cmd := m.prints(); cmd != nil {
		t.Error("tui mode handed a Cmd to the terminal, want it held in memory")
	}
	held := visible(strings.Join(m.hold, "\n"))
	if !strings.Contains(held, "a question") {
		t.Errorf("held = %q, want the said line", held)
	}
	if len(m.unprinted) != 0 {
		t.Errorf("unprinted = %v, want it drained into hold", m.unprinted)
	}
}

// The hold is a bounded ring of painted rows: past holdRowsCap the oldest rows
// drop, so a long session cannot charge every frame for rows no frame can
// show, and the newest rows — the ones a frame can show — always survive.
func TestTheHoldDropsItsOldestRowsOnceCapped(t *testing.T) {
	m := tuiModel()
	for i := range 2 * holdRowsCap {
		m.printed(fmt.Sprintf("row %d", i))
	}
	if len(m.hold) != holdRowsCap {
		t.Fatalf("hold = %d rows, want it capped at %d", len(m.hold), holdRowsCap)
	}
	if want := fmt.Sprintf("row %d", holdRowsCap); m.hold[0] != want {
		t.Errorf("first held row = %q, want the first row past the cap %q", m.hold[0], want)
	}
	if want := fmt.Sprintf("row %d", 2*holdRowsCap-1); m.hold[len(m.hold)-1] != want {
		t.Errorf("last held row = %q, want the newest row %q", m.hold[len(m.hold)-1], want)
	}
}

// The whole point of the mode: the prompt is pinned to the bottom of an
// alternate screen. The prompt's text sits on the very last row — nothing is
// drawn beneath it — with a single blank row between it and the transcript
// above, like a vim status bar, instead of riding the live region up and down.
func TestTUIModePinsThePromptToTheBottom(t *testing.T) {
	m := tuiModel()
	m.say(fromReader, "a line of transcript")
	m.say(fromModel, "and another line")

	lines := strings.Split(ansi.Strip(m.View().Content), "\n")
	first, last := -1, -1
	for i, line := range lines {
		if strings.Contains(line, "placeholder") {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		t.Fatalf("no prompt line in\n%s", strings.Join(lines, "\n"))
	}
	if last != len(lines)-1 {
		t.Errorf("prompt text ends at row %d, want it on the very last row %d", last, len(lines)-1)
	}
	if first != last {
		t.Errorf("prompt spans rows %d..%d, want a single row", first, last)
	}
	if strings.TrimSpace(lines[first-1]) != "" {
		t.Errorf("row above prompt = %q, want blank", lines[first-1])
	}
}

// In the alternate screen the whole frame is ours; the AltScreen flag is what
// tells Bubble Tea to draw it there rather than over scrollback.
func TestTUIModeRendersOnTheAlternateScreen(t *testing.T) {
	m := tuiModel()
	if !m.View().AltScreen {
		t.Error("tui mode view is not marked for the alternate screen")
	}
}

// The cursor must land on the prompt's own text row. The status line is two
// rows inside one string, so the upper-region row count used to undercount by
// one and the cursor sat a row above the typing line. The cursor's y is set by
// an absolute offset into the whole frame, so it should match the row the text
// renders on.
func TestTUIModeCursorSitsOnThePromptTextRow(t *testing.T) {
	m := tuiModel()
	m.say(fromReader, "a line of transcript")
	m.say(fromModel, "and another line")

	view := m.View()
	if view.Cursor == nil {
		t.Fatal("tui mode view has no cursor")
	}
	lines := strings.Split(ansi.Strip(view.Content), "\n")
	if view.Cursor.Y < 0 || view.Cursor.Y >= len(lines) {
		t.Fatalf("cursor y = %d, outside the %d-row frame", view.Cursor.Y, len(lines))
	}
	if !strings.Contains(lines[view.Cursor.Y], "placeholder") {
		t.Errorf("cursor at row %d = %q, want the prompt's text row", view.Cursor.Y, lines[view.Cursor.Y])
	}
}

// containsString reports whether needle appears in any entry of hay.
func containsString(hay []string, needle string) bool {
	for _, line := range hay {
		if strings.Contains(line, needle) {
			return true
		}
	}
	return false
}

// Inline mode must keep living in the terminal's own scrollback: the same
// say-then-print path yields a Cmd here, never a held buffer.
func TestInlineModeHandsTheTranscriptToTheTerminal(t *testing.T) {
	m := sized()
	m.say(fromReader, "a question")

	if cmd := m.prints(); cmd == nil {
		t.Error("inline mode produced no Cmd, want the line handed to the terminal")
	}
	if len(m.hold) != 0 {
		t.Errorf("hold = %v, want it empty in inline mode", m.hold)
	}
}

// window keeps the newest-avail tail of a scrolled slice, moved back by scroll
// rows, and never lets the read go past the top or below the newest row.
func TestWindowSlicesToTheNewestTailAndScrolls(t *testing.T) {
	content := []string{"a", "b", "c", "d", "e"}
	if got := window(content, 3, 0); !slices.Equal(got, []string{"c", "d", "e"}) {
		t.Errorf("window(content,3,0) = %v, want c d e", got)
	}
	if got := window(content, 3, 2); !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Errorf("window(content,3,2) = %v, want a b c", got)
	}
	if got := window(content, 3, 99); !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Errorf("over-scroll window = %v, want it clamped to the top", got)
	}
	if got := window([]string{"x"}, 3, 1); !slices.Equal(got, []string{"x"}) {
		t.Errorf("window with content shorter than avail = %v, want x unchanged", got)
	}
}

// The scroll wheel moves the tui-mode transcript window back through the held
// lines, and scrolling back down re-anchors it to the newest row — the whole
// point of the prompt being pinned to the bottom.
func TestTUIModeWheelScrollsTheHeldTranscript(t *testing.T) {
	m := tuiModel()
	for i := range 60 {
		m.say(fromClient, fmt.Sprintf("row %02d", i))
	}
	m.prints()
	bottom := strings.Split(ansi.Strip(m.View().Content), "\n")
	if !containsString(bottom, "row 59") {
		t.Fatalf("at the bottom the newest row should be visible in\n%s", strings.Join(bottom, "\n"))
	}
	if containsString(bottom, "row 00") {
		t.Fatalf("at the bottom the oldest row should not yet be visible in\n%s", strings.Join(bottom, "\n"))
	}
	for range 100 {
		m.route(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	}
	if m.scrollTop <= 0 {
		t.Fatalf("scrollTop = %d after wheel up, want > 0", m.scrollTop)
	}
	if !containsString(strings.Split(ansi.Strip(m.View().Content), "\n"), "row 00") {
		t.Error("after scrolling up the oldest row should surface")
	}
	for range 200 {
		m.route(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	}
	if m.scrollTop != 0 {
		t.Errorf("scrollTop = %d after scrolling down, want 0 (re-anchored)", m.scrollTop)
	}
	if !containsString(strings.Split(ansi.Strip(m.View().Content), "\n"), "row 59") {
		t.Error("after scrolling down the newest row should return")
	}
}

// A wheel must never reach the prompt, where it could be read as up/down and
// drag the input's recall history into view. Routing it through scrollWheel
// always claims it, so the prompt text is untouched by a wheel message.
func TestTUIModeWheelNeverNavigatesPromptHistory(t *testing.T) {
	m := tuiModel()
	m.prompt.SetValue("a draft")
	m.route(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	if got := m.prompt.Value(); got != "a draft" {
		t.Errorf("prompt value = %q after a wheel, want the untouched draft", got)
	}
}

// The prompt reads as a single line rather than a padded bar: a single line of
// input renders exactly one row, with the text on that row — no blank padding
// rows inside the field. It grows only when the input actually wraps.
func TestThePromptHasNoPadding(t *testing.T) {
	m := sized()
	lines := strings.Split(ansi.Strip(m.prompt.View()), "\n")
	if len(lines) != minHeightRows {
		t.Errorf("prompt rendered %d rows for one line of input, want %d", len(lines), minHeightRows)
	}
	if strings.TrimSpace(lines[0]) == "" {
		t.Errorf("text row %d is empty in\n%s", 0, strings.Join(lines, "\n"))
	}
}
