package compaction

import "strings"

// Identifiers pulls the load-bearing tokens out of a ledger body: the spans it
// marked as code, and the bare tokens that read as a path, a flag or a filename.
// They are what a rewrite may not drop, and the rule is deliberately lexical
// rather than semantic — a deterministic check can refuse a rewrite, and nothing
// here needs to understand what any one identifier means.
func Identifiers(text string) []string {
	var out []string
	inside := false
	for span := range strings.SplitSeq(text, "`") {
		if inside {
			out = append(out, strings.TrimSpace(span))
		}
		inside = !inside
	}
	for field := range strings.FieldsSeq(text) {
		if isPathLike(field) {
			out = append(out, identTrim(field))
		}
	}
	return dedupe(out)
}

// MissingIdentifiers is the identifiers a body no longer carries, empty when the
// rewrite kept every one of them. It is the deterministic half of the guard on a
// rewriting pass: a body that dropped what an earlier one recorded is refused,
// and the merge stands instead — growth, which the next pass can attack again,
// rather than a fact nobody can recover.
func MissingIdentifiers(previous, body string) []string {
	var missing []string
	for _, id := range Identifiers(previous) {
		if !strings.Contains(body, id) {
			missing = append(missing, id)
		}
	}
	return missing
}

// isPathLike reports whether a whitespace-delimited token reads as something a
// model would have to reproduce exactly: a path, a command with a slash in it, a
// flag, or a filename with an extension. Short tokens are skipped, because a
// bare ".go" or "a/b" carries no state worth refusing a rewrite over, and a
// trailing-period abbreviation like "e.g." is too short to qualify.
func isPathLike(tok string) bool {
	tok = identTrim(tok)
	if len(tok) < 4 {
		return false
	}
	if strings.Contains(tok, "/") || strings.HasPrefix(tok, "-") {
		return true
	}
	dot := strings.LastIndex(tok, ".")
	return dot > 0 && len(tok)-dot <= 6
}

// identTrim strips the punctuation a token can be wrapped in without being part
// of it — quotes, brackets, a comma it ended a clause with — so the same path
// spelled inside backticks and bare is recognised as one identifier rather than
// two.
func identTrim(tok string) string {
	return strings.Trim(tok, "`'\"()[]{}.,;:")
}

// dedupe drops repeats, keeping the order they were first seen in, and drops
// empties: a pair of backticks with nothing between them names no identifier.
func dedupe(items []string) []string {
	seen := make(map[string]bool, len(items))
	out := items[:0]
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}
