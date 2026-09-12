// Raw passthrough for gate output that does not parse as compiler-style
// findings: capped at the same budgets as the finding renderer, with overflow
// spilled to a file the pointer line names.

package diagnostics

import (
	"fmt"
	"strings"
)

// renderRaw passes a gate's unparsable output through verbatim, capped at the
// same finding-count and byte budgets as the compiler-style renderer, with
// the overflow spilled to a file the pointer line names.
func renderRaw(gate, text string) string {
	if text == "" {
		return fmt.Sprintf("%s: failed with exit code 1", gate)
	}
	lines := strings.Split(text, "\n")
	if len(lines) > maxFindings || joined(lines) > maxTextBytes {
		head := fit(lines[:min(len(lines), maxFindings-1)], 0)
		name, err := spillFile(text)
		if err != nil {
			return strings.Join(head, "\n")
		}
		pointer := fmt.Sprintf("%s: %d lines of output; full text: %s", gate, len(lines), name)
		return strings.Join(append(head, pointer), "\n")
	}
	return strings.Join(lines, "\n")
}
