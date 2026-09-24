package ide

import "strings"

// spanOfRewrite reports the first and last line, 1-based and inclusive, that a
// whole-file rewrite touched in the file it produced. A rewrite that only
// removed lines has no changed line of its own in the new file, so it is
// reported at the single line the removal left behind: an editor asked to
// mark an empty span has nowhere to put the mark.
func spanOfRewrite(before, after string) (int, int) {
	old, updated := splitLines(before), splitLines(after)
	head := sharedHead(old, updated)
	tail := sharedTail(old[head:], updated[head:])
	first := head + 1
	last := first
	if end := len(updated) - tail; end > first {
		last = end
	}
	return clampLine(first, len(updated)), clampLine(last, len(updated))
}

// spanOfFragment reports the first and last line a fragment occupies in the
// file it was written into, and says no when the file does not hold it: a
// fragment nobody can find is a span nobody can mark.
func spanOfFragment(text, fragment string) (int, int, bool) {
	at := strings.Index(text, fragment)
	if fragment == "" || at < 0 {
		return 0, 0, false
	}
	first := strings.Count(text[:at], "\n") + 1
	return first, first + strings.Count(strings.TrimSuffix(fragment, "\n"), "\n"), true
}

// splitLines breaks file contents into the lines a mark is counted in,
// dropping the single trailing newline that ends a well-formed text file
// rather than counting it as a line of its own.
func splitLines(text string) []string {
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// sharedHead counts the leading lines two files have in common.
func sharedHead(old, updated []string) int {
	n := 0
	for n < len(old) && n < len(updated) && old[n] == updated[n] {
		n++
	}
	return n
}

// sharedTail counts the trailing lines two files have in common, none of
// which is a line sharedHead already claimed.
func sharedTail(old, updated []string) int {
	n := 0
	for n < len(old) && n < len(updated) && old[len(old)-1-n] == updated[len(updated)-1-n] {
		n++
	}
	return n
}

// clampLine keeps a reported line inside the file it marks, and never below
// the first: line numbers are 1-based, and a mark at line 0 is no mark.
func clampLine(line, total int) int {
	if line < 1 || total < 1 {
		return 1
	}
	return min(line, total)
}
