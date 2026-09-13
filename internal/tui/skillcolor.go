package tui

import "strings"

// skillYellow is the raw escape that recolours a /skill token. Foreground
// only: the prompt and the question pane paint their own backgrounds, and a
// full reset would wipe them mid-row. The closing sequence is the token's
// saved SGR, re-emitted by highlightSkills, which puts whatever style was in
// effect back for the rest of the row.
const skillYellow = "\x1b[33m"

// skillTokens lists every /skill:<name> token the loaded skills answer to.
func (m *Model) skillTokens() []string {
	tokens := make([]string, 0, len(m.skills))
	for name := range m.skills {
		tokens = append(tokens, "/skill:"+name)
	}
	return tokens
}

// highlightSkills recolours every /skill token in a rendered row yellow.
//
// The textarea and lipgloss offer no span styling, so the hint is painted
// after rendering: the scanner walks the row, steps over escape sequences
// untouched, and wraps each token matching a loaded skill — exactly, or as
// the half-typed prefix the autocomplete would still offer — in yellow, then
// re-emits the SGR sequence that was in effect before the token so the
// cursor-line bar and pane backgrounds survive. Tokens never contain the
// escapes' characters, and a slash never appears inside an escape sequence,
// so byte scanning is safe. Multiple tokens per row highlight independently.
func highlightSkills(row string, tokens []string) string {
	var out strings.Builder
	paint := skillPainter{tokens: tokens}
	i := 0
	boundary := true
	for i < len(row) {
		if row[i] == 0x1b {
			seq := ansiSequence(row[i:])
			out.WriteString(seq)
			paint.sgrAfter(seq)
			i += len(seq)
			continue
		}
		if token, end := skillAt(row, i, boundary, paint.tokens); token != "" {
			out.WriteString(paint.wrap(token))
			i = end
			boundary = false
			continue
		}
		boundary = wordGap(row[i])
		out.WriteByte(row[i])
		i++
	}
	return out.String()
}

// skillPainter carries the highlight state across a row: the tokens to
// highlight, and the SGR sequence last seen, which re-emitting after a token
// restores the row's own styling.
type skillPainter struct {
	tokens []string
	sgr    string
}

// sgrAfter records seq as the style in effect when it is an SGR sequence.
func (p *skillPainter) sgrAfter(seq string) {
	if strings.HasSuffix(seq, "m") {
		p.sgr = seq
	}
}

// wrap paints one token yellow, then hands the row its style back.
func (p skillPainter) wrap(token string) string {
	return skillYellow + token + "\x1b[39m" + p.sgr
}

// ansiSequence returns the escape sequence at the front of row: ESC plus
// everything through the CSI final byte, or one more byte when the input is
// not a well-formed sequence.
func ansiSequence(row string) string {
	i := 2
	for i < len(row) && (row[i] < 0x40 || row[i] > 0x7e) {
		i++
	}
	if i < len(row) {
		i++
	}
	return row[:i]
}

// skillAt returns the /skill token starting at i when it sits on a word
// boundary, together with its end offset; empty when there is none.
func skillAt(row string, i int, boundary bool, tokens []string) (string, int) {
	if row[i] != '/' || !boundary {
		return "", i
	}
	end := i
	for end < len(row) && !wordGap(row[end]) && row[end] != 0x1b {
		end++
	}
	if highlightsSkill(row[i:end], tokens) {
		return row[i:end], end
	}
	return "", i
}

// highlightsSkill reports whether token names a loaded skill exactly, or is
// the prefix of one still being typed.
func highlightsSkill(token string, tokens []string) bool {
	for _, name := range tokens {
		if token == name || (strings.HasPrefix(name, token) && len(token) < len(name)) {
			return true
		}
	}
	return false
}
