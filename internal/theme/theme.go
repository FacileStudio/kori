// Package theme defines styles and color palettes for the terminal UI.
package theme

import (
	"image/color"
	"regexp"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
)

var osc8 = regexp.MustCompile(`\x1b\]8;[^\x1b\x07]*(?:\x07|\x1b\\)`)

// PlainTool returns the fallback style for tools without a custom glyph.
func PlainTool() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
}

// PlainToolANSI is the ANSI colour code for the fallback tool style.
const PlainToolANSI = "34"

// TranscriptStyles contains styles used when formatting transcript rows.
type TranscriptStyles struct {
	Question lipgloss.Style
	Thinking lipgloss.Style
	Tool     lipgloss.Style
	Result   lipgloss.Style
	Failure  lipgloss.Style
	Client   lipgloss.Style
	Command  lipgloss.Style
}

// UIStyles contains styles used for UI elements outside the transcript.
type UIStyles struct {
	Menu       lipgloss.Style
	Plain      lipgloss.Style
	Waiting    lipgloss.Style
	Ready      lipgloss.Style
	Muted      lipgloss.Style
	Border     lipgloss.Style
	Compacting lipgloss.Style
}

// Palette groups all styles and settings needed to render the interface.
type Palette struct {
	TranscriptStyles
	UIStyles
	Markdown string
}

func transcriptStylesFor(quiet color.Color) TranscriptStyles {
	question := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("15")).Bold(true)
	return TranscriptStyles{
		Question: question,
		Thinking: lipgloss.NewStyle().Foreground(quiet).Italic(true),
		Tool:     PlainTool(),
		Result:   lipgloss.NewStyle().Foreground(quiet),
		Failure:  lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		Client:   lipgloss.NewStyle().Foreground(quiet),
		Command:  lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	}
}

func uiStylesFor(pick func(a, b color.Color) color.Color, quiet color.Color) UIStyles {
	return UIStyles{
		Menu: lipgloss.NewStyle().
			Background(pick(lipgloss.Color("7"), lipgloss.Color("8"))).
			Foreground(lipgloss.Color("15")),
		Plain:      lipgloss.NewStyle(),
		Waiting:    lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		Ready:      lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		Muted:      lipgloss.NewStyle().Foreground(quiet),
		Border:     lipgloss.NewStyle().Foreground(pick(lipgloss.Color("0"), lipgloss.Color("15"))),
		Compacting: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
	}
}

// Themed builds the palette for a light or dark terminal.
func Themed(dark bool) Palette {
	pick := lipgloss.LightDark(dark)
	quiet := pick(lipgloss.Color("242"), lipgloss.Color("244"))

	style := "light"
	if dark {
		style = "dark"
	}

	return Palette{
		TranscriptStyles: transcriptStylesFor(quiet),
		UIStyles:         uiStylesFor(pick, quiet),
		Markdown:         style,
	}
}

// Prettier builds the glamour markdown renderer.
//
// The palette is glamour's dark or light layout, kept whole: headings,
// tables, code and inline emphasis all render in glamour's own colours, the
// way glow does. Width is the word-wrap width.
func Prettier(style string, width int) *glamour.TermRenderer {
	base := styles.LightStyleConfig
	if style == "dark" {
		base = styles.DarkStyleConfig
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(base),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil
	}
	return renderer
}

// RenderMarkdown formats markdown text for terminal display.
//
// Glamour frames every render with a top-margin newline and trailing fill to
// the wrap width. Answers stream into the terminal as committed fragments (see
// commitParagraphs), each rendered on its own and joined back together in
// prints, so that framing would otherwise leave a blank line between every
// fragment. Trim the framing here and keep the paragraph's own indent.
func RenderMarkdown(r *glamour.TermRenderer, text string) string {
	if r == nil {
		return text
	}
	rendered, err := r.Render(text)
	if err != nil {
		return text
	}
	rendered = dropHyperlinks(rendered)
	return strings.TrimLeft(strings.TrimRight(rendered, " \n"), "\n")
}

// dropHyperlinks removes OSC 8 hyperlink sequences.
//
// tmux does not speak OSC 8 and forwards the sequences mangled, which leaves
// kitty — the terminal behind tmux over ssh — with a hyperlink open past its
// reset: everything after renders underlined. Plain text keeps the link label
// and reads the same.
func dropHyperlinks(rendered string) string {
	return osc8.ReplaceAllString(rendered, "")
}
