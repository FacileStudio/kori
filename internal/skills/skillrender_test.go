package skills

import (
	"errors"
	"strings"
	"testing"
)

func TestRenderSkillsIsEmptyWithNothingFound(t *testing.T) {
	if got := renderSkills(nil); got != "" {
		t.Errorf("rendered = %q, want empty with no skills", got)
	}
}

func TestRenderSkillsListsNameDescriptionAndPath(t *testing.T) {
	got := renderSkills([]skill{{Name: "pdf-tools", Description: "Extracts text.", Path: "/x/SKILL.md"}})

	for _, want := range []string{"pdf-tools", "Extracts text.", "/x/SKILL.md"} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered = %q, want it to mention %q", got, want)
		}
	}
}

// All three halves of skillNotice are independent facts and any can be true
// alone — a save failure is not the same problem as an unreviewed
// directory or a rejected manifest, and conflating them into one condition
// would hide whichever one did not happen to be checked first.
func TestSkillNoticeReportsASaveFailureSeparatelyFromSkippedSkills(t *testing.T) {
	notice := skillNotice(nil, errors.New("disk full"), nil)

	if !strings.Contains(notice, "disk full") {
		t.Errorf("notice = %q, want the save error mentioned", notice)
	}
	if strings.Contains(notice, "not trusted") {
		t.Errorf("notice = %q, want no skipped-skills text when nothing was skipped", notice)
	}
}

func TestSkillNoticeIsEmptyWithNothingToReport(t *testing.T) {
	if got := skillNotice(nil, nil, nil); got != "" {
		t.Errorf("notice = %q, want empty with nothing skipped, failed, or rejected", got)
	}
}

// A manifest that exists but was rejected used to vanish without a word —
// an installed skill that never showed up was indistinguishable from one
// that was never installed. The notice has to name the file and say why.
func TestSkillNoticeNamesRejectedManifests(t *testing.T) {
	notice := skillNotice(nil, nil, []string{"/skills/broken/SKILL.md: frontmatter does not parse as YAML: oops"})

	if !strings.Contains(notice, "/skills/broken/SKILL.md") {
		t.Errorf("notice = %q, want the rejected file named", notice)
	}
	if !strings.Contains(notice, "frontmatter does not parse as YAML") {
		t.Errorf("notice = %q, want the rejection reason included", notice)
	}
}
