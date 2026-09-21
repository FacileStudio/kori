package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/FacileStudio/kori/internal/compaction"
	"github.com/FacileStudio/nacelle"
)

// stubJudge is a Judge with no answers: the tests that only care whether a judge
// is attached need a non-nil implementation, not a classification.
type stubJudge struct{}

func (stubJudge) Classify(context.Context, string, []compaction.Block) ([]compaction.Verdict, error) {
	return nil, nil
}

// The footer names the context against the window it is measured on, with the
// ratio and the tier the size has reached, so a reader can see the ladder
// coming rather than only learning about it from the compaction report.
func TestTheFooterShowsTheContextRatioAndTier(t *testing.T) {
	m := sized()
	m.policy = windowedPolicy()
	m.size = 120_000

	foot := visible(strings.Join(m.footer(), " "))
	if !strings.Contains(foot, "↕120k/200k · 0.60") {
		t.Errorf("footer = %q, want the size against the window with its ratio", foot)
	}
	if !strings.Contains(foot, "soft") {
		t.Errorf("footer = %q, want the tier the size has reached", foot)
	}
}

// With no window to measure against the ratio is undefined, so the footer keeps
// the plain size and takes the tier from the absolute ceiling instead.
func TestTheFooterKeepsThePlainSizeWithoutAWindow(t *testing.T) {
	m := sized()
	m.policy = compaction.Policy{
		Ratios:  compaction.Ratios{Soft: 0.65, Mid: 0.80, Hard: 0.90},
		Ceiling: 100_000,
	}
	m.size = 150_000

	foot := visible(strings.Join(m.footer(), " "))
	if !strings.Contains(foot, "↕150k") || strings.Contains(foot, "/") {
		t.Errorf("footer = %q, want the plain size and no ratio", foot)
	}
	if !strings.Contains(foot, "mid") {
		t.Errorf("footer = %q, want the tier the ceiling bought", foot)
	}
}

// /status carries the ladder beyond the footer's figure: the accumulated ledger
// with the tier of the last pass that wrote it.
func TestStatusReportsTheLedgerAndTheLastPassTier(t *testing.T) {
	m := sized()
	m.policy = windowedPolicy()
	m.size = 150_000
	m.conversation = []nacelle.Message{
		nacelle.UserText("the original task"),
		compaction.BuildLedger("", "a decision, a constraint"),
		nacelle.AssistantText("older work"),
		nacelle.UserText("older answer"),
		nacelle.AssistantText("active turn"),
		nacelle.UserText("the newest turn"),
	}
	m.last = compacted{tier: compaction.Mid, turns: 2}

	m.statusCmd()
	got := strings.Join(m.unprinted, "\n")
	for _, want := range []string{"↕150k/200k · 0.75", "ledger · ~", "last pass mid"} {
		if !strings.Contains(got, want) {
			t.Errorf("status missing %q in %q", want, got)
		}
	}
}

// A session that has measured nothing and built no ledger says nothing about
// compaction, rather than printing a zero line every time /status is asked.
func TestStatusSaysNothingAboutCompactionBeforeTheFirstPass(t *testing.T) {
	m := sized()
	m.statusCmd()
	got := strings.Join(m.unprinted, "\n")
	for _, want := range []string{"ledger ·", "judge ·"} {
		if strings.Contains(got, want) {
			t.Errorf("status = %q, want no %q line with nothing to report", got, want)
		}
	}
}

// The judge is the one setting that sends the conversation off the machine, and
// once enabled nothing else in the UI says so — a reader who opted in months ago
// has nothing to remind them that every pass ships the history somewhere. /status
// is where they already look for the ladder, so the exposure is stated there.
func TestStatusNamesAnOptedInJudge(t *testing.T) {
	m := sized()
	m.policy = windowedPolicy()
	m.judge = stubJudge{}
	m.size = 150_000

	m.statusCmd()

	if got := strings.Join(m.unprinted, "\n"); !strings.Contains(got, "judge · on") {
		t.Errorf("status = %q, want the judge named while it is on", got)
	}
}
