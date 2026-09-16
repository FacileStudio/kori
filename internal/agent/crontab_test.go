package agent

import (
	"strings"
	"testing"
)

func TestApplyCrontabBlockToEmpty(t *testing.T) {
	block := "# BEGIN KORI JOB test\n0 0 * * * /bin/kori cron run test >> /logs/test.log 2>&1\n# END KORI JOB test"
	got := applyCrontabBlock("", "test", block)
	want := block + "\n"
	if got != want {
		t.Errorf("applyCrontabBlock(\"\") = %q, want %q", got, want)
	}
}

func TestApplyCrontabBlockReplaceExisting(t *testing.T) {
	initial := "# BEGIN KORI JOB test\n0 0 * * * /bin/kori cron run test >> /logs/test.log 2>&1\n# END KORI JOB test\n"
	newBlock := "# BEGIN KORI JOB test\n0 9 * * * /bin/kori cron run test >> /logs/test.log 2>&1\n# END KORI JOB test"
	got := applyCrontabBlock(initial, "test", newBlock)
	want := newBlock + "\n"
	if got != want {
		t.Errorf("applyCrontabBlock replace = %q, want %q", got, want)
	}
}

func TestApplyCrontabBlockPreservesOtherEntries(t *testing.T) {
	initial := "0 * * * * /bin/other\n\n# BEGIN KORI JOB other\n0 0 * * * /bin/kori cron run other\n# END KORI JOB other\n"
	newBlock := "# BEGIN KORI JOB test\n0 9 * * * /bin/kori cron run test >> /logs/test.log 2>&1\n# END KORI JOB test"
	got := applyCrontabBlock(initial, "test", newBlock)
	if !strings.Contains(got, "0 * * * * /bin/other") {
		t.Errorf("expected preserved other entry in %q", got)
	}
	if !strings.Contains(got, "# BEGIN KORI JOB other") {
		t.Errorf("expected preserved other job block in %q", got)
	}
	if !strings.Contains(got, "# BEGIN KORI JOB test") {
		t.Errorf("expected new test job block in %q", got)
	}
}

func TestStripCrontabBlock(t *testing.T) {
	initial := "# BEGIN KORI JOB test\n0 0 * * * /bin/kori cron run test\n# END KORI JOB test\n0 * * * * /bin/other\n"
	got, removed := stripCrontabBlock(initial, "test")
	if !removed {
		t.Errorf("expected removed = true")
	}
	if strings.Contains(got, "BEGIN KORI JOB test") {
		t.Errorf("expected block removed, got %q", got)
	}
	if !strings.Contains(got, "0 * * * * /bin/other") {
		t.Errorf("expected other entry preserved, got %q", got)
	}
}

func TestStripLegacyUntaggedLine(t *testing.T) {
	initial := "0 0 * * * /home/user/.local/bin/kori cron run news\n0 1 * * * /bin/other\n"
	got, removed := stripCrontabBlock(initial, "news")
	if !removed {
		t.Errorf("expected removed = true")
	}
	if strings.Contains(got, "cron run news") {
		t.Errorf("expected legacy line removed, got %q", got)
	}
	if !strings.Contains(got, "0 1 * * * /bin/other") {
		t.Errorf("expected other entry preserved, got %q", got)
	}
}
