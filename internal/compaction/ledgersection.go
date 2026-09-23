// Package compaction — the structural half of folding one ledger body into
// another. It lives apart from ledgerbody.go because the file cap is real and
// the two questions are different: this file decides what a body is made of,
// the other decides what the next body should be.
package compaction

import "strings"

// section is one headed block of a ledger body: the header line that opened it
// — empty for a body written with no headers at all — and the lines under it.
// The summarizer is asked for a fixed list of headers (Decisions, Constraints,
// Plan, State, Artifacts, Ruled out, Open questions), so parsing on headers is
// reading the schema rather than guessing at prose.
type section struct {
	head string
	body []string
}

// sections splits a ledger body into headed blocks, in the order they appear.
// A header opens a block; every other non-empty line belongs to the block above
// it, and a body that opens with prose gets an unnamed block to hold it. Blank
// lines are layout, not content, so they are dropped — render puts them back.
func sections(body string) []section {
	var out []section
	for line := range strings.SplitSeq(strings.TrimSpace(body), "\n") {
		line = strings.TrimRight(line, " \t")
		switch {
		case line == "":
			continue
		case isHead(line):
			out = append(out, section{head: line})
		default:
			if len(out) == 0 {
				out = append(out, section{})
			}
			out[len(out)-1].body = append(out[len(out)-1].body, line)
		}
	}
	return out
}

// sectionNames is the schema the summarizer writes against. A line naming one of
// these opens a section whether or not it carries the colon, because the model is
// told to emit exactly this list and its formatting of the labels varies — and a
// header read as content is a header duplicated on every pass.
var sectionNames = map[string]bool{
	"decisions": true, "constraints": true, "plan": true, "state": true,
	"artifacts": true, "ruled out": true, "open questions": true,
}

// isHead reports whether a line opens a section: a label from the schema, a short
// label ending in a colon, or one carrying markdown emphasis or a heading marker.
// Anything else is content, including a bullet, because a bullet is a fact rather
// than a name for a group of them.
func isHead(line string) bool {
	trimmed := strings.TrimSpace(line)
	switch {
	case trimmed == "":
		return false
	case strings.HasPrefix(trimmed, "#"), strings.HasPrefix(trimmed, "**"):
		return true
	case strings.HasPrefix(trimmed, "- "), strings.HasPrefix(trimmed, "* "):
		return false
	default:
		return sectionNames[normalize(trimmed)] || (strings.HasSuffix(trimmed, ":") && len(trimmed) <= 60)
	}
}

// mergeSections folds summary into previous, section by section and in
// previous's order, then appends the sections only summary has. Every line
// previous carried is kept as it was written, and a line summary repeats is
// dropped: that is what makes the fold monotone without repeating a fact, which
// is the whole point of merging rather than concatenating.
func mergeSections(previous, summary []section) []section {
	out := make([]section, 0, len(previous)+len(summary))
	index := make(map[string]int, len(previous))
	for _, sec := range previous {
		index[normalize(sec.head)] = len(out)
		out = append(out, sec)
	}
	for _, sec := range summary {
		at, ok := index[normalize(sec.head)]
		if !ok {
			index[normalize(sec.head)] = len(out)
			out = append(out, sec)
			continue
		}
		out[at].body = appendUnique(out[at].body, sec.body)
	}
	return out
}

// appendUnique adds the lines of add that lines does not already carry, compared
// on the normalized text so a restatement under different whitespace or
// punctuation is still recognised as the same fact.
func appendUnique(lines, add []string) []string {
	seen := make(map[string]bool, len(lines)+len(add))
	for _, line := range lines {
		seen[normalize(line)] = true
	}
	for _, line := range add {
		if key := normalize(line); !seen[key] {
			seen[key] = true
			lines = append(lines, line)
		}
	}
	return lines
}

// render writes sections back out as a body: each head, a blank line between
// blocks, and every line it holds. It is the inverse of sections, so a body
// already in this shape merges into itself unchanged.
func render(secs []section) string {
	var lines []string
	for _, sec := range secs {
		if sec.head != "" {
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, sec.head)
		}
		lines = append(lines, sec.body...)
	}
	return strings.Join(lines, "\n")
}

// normalize is the key two lines are compared on: whitespace collapsed, case
// folded, a bullet's own marker dropped, and trailing punctuation ignored. It
// collapses what a model restates without altering and leaves genuinely
// different facts apart.
func normalize(line string) string {
	trimmed := strings.TrimSpace(line)
	for _, marker := range []string{"- ", "* ", "• "} {
		trimmed = strings.TrimPrefix(trimmed, marker)
	}
	collapsed := strings.Join(strings.Fields(trimmed), " ")
	return strings.ToLower(strings.TrimRight(collapsed, " .,;:!?"))
}
