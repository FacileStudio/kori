package skills

import (
	"path/filepath"
	"strings"
	"testing"
)

// A folded block scalar (description: >) decodes with folded line breaks and
// a trailing newline, which a menu item renders as a second, blank line.
func TestParseSkillDescriptionIsSingleLine(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "name: my-skill\ndescription: >\n  First line,\n  folded onto one, with a trailing newline.")
	s, problem := parseSkill(filepath.Join(dir, "SKILL.md"))
	if problem != "" {
		t.Fatalf("parseSkill problem: %v", problem)
	}
	if strings.ContainsAny(s.Description, "\n\r") || s.Description != strings.TrimSpace(s.Description) {
		t.Errorf("description = %q, want a single line with no trailing newline", s.Description)
	}
}
