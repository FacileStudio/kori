package compaction

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

// fakeJudge is a Judge with a scripted answer, so every macro path can be driven
// without a network.
type fakeJudge struct {
	verdicts []Verdict
	err      error
	goal     string
	blocks   []Block
}

func (f *fakeJudge) Classify(_ context.Context, goal string, blocks []Block) ([]Verdict, error) {
	f.goal, f.blocks = goal, blocks
	if f.err != nil {
		return nil, f.err
	}
	return f.verdicts, nil
}

// judgeSample is a conversation whose history is one atomic tool pair and one
// standalone turn, the two shapes a block can be.
func judgeSample() []nacelle.Message {
	return []nacelle.Message{
		nacelle.UserText("the task"),
		callMessage("c1", "read"),
		resultMessage("c1", "read", strings.Repeat("x", 40_000)),
		nacelle.AssistantText("answer one"),
		nacelle.UserText("follow up"),
		nacelle.AssistantText("answer two"),
	}
}

func judgePlan(conv []nacelle.Message) []Span {
	return Plan(conv, Policy{AnchorMessages: 1, KeepTurns: 2})
}

// With the judge off nothing is classified, nothing is pruned and the whole
// history is folded — byte-for-byte the pre-judge pass.
func TestClassifyWithNoJudgeFoldsEverything(t *testing.T) {
	conv := judgeSample()
	plan := judgePlan(conv)

	fold, err := Classify(t.Context(), conv, plan, JudgeRequest{}, nil)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if fold.LedgerSize() != len(HistoryMessages(conv, plan)) {
		t.Errorf("ledger = %d messages, want the whole history", fold.LedgerSize())
	}
	if fold.PrunedSize() != 0 {
		t.Errorf("pruned = %d messages, want none without a judge", fold.PrunedSize())
	}
}

// A judge error degrades to all-keep: nothing is pruned and the error is
// returned for the caller to fall back on.
func TestClassifyOnErrorKeepsEveryBlock(t *testing.T) {
	conv := judgeSample()
	plan := judgePlan(conv)
	judge := &fakeJudge{err: errors.New("429 too many requests")}

	fold, err := Classify(t.Context(), conv, plan, JudgeRequest{Goal: "the task"}, judge)
	if err == nil {
		t.Fatal("err = nil, want the judge failure carried back")
	}
	for _, block := range Blocks(conv, plan) {
		if !fold.Survives(block.Start) {
			t.Errorf("block %+v was pruned after a judge failure", block)
		}
	}
}

// Verdicts decide where each block goes, and a forced pass upgrades keeps to
// ledger folds — the hard tier's force-summarize.
func TestClassifySortsBlocksAndForcesKeeps(t *testing.T) {
	conv := judgeSample()
	plan := judgePlan(conv)
	scripted := &fakeJudge{verdicts: []Verdict{{Decision: Prune}, {Decision: Ledger}}}

	fold, err := Classify(t.Context(), conv, plan, JudgeRequest{Goal: "the task"}, scripted)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if fold.PrunedSize() != 2 || fold.LedgerSize() != 1 {
		t.Errorf("fold = %d pruned / %d ledger, want the tool pair pruned and the turn folded", fold.PrunedSize(), fold.LedgerSize())
	}
	if scripted.goal != "the task" {
		t.Errorf("goal = %q, want the request's goal passed through", scripted.goal)
	}

	keeps := &fakeJudge{verdicts: keepAll(2)}
	forced, err := Classify(t.Context(), conv, plan, JudgeRequest{Goal: "the task", Force: true}, keeps)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if forced.LedgerSize() != 3 || forced.PrunedSize() != 0 {
		t.Errorf("forced fold = %d ledger / %d pruned, want every keep folded", forced.LedgerSize(), forced.PrunedSize())
	}
}

// A pruned block is a whole atomic span: the prune takes the call and the result
// together, so the rebuild can never leave an orphan ToolResult.
func TestPruneDropsWholeAtomicBlocks(t *testing.T) {
	conv := judgeSample()
	plan := judgePlan(conv)
	judge := &fakeJudge{verdicts: []Verdict{{Decision: Prune}, {Decision: Ledger}}}

	fold, err := Classify(t.Context(), conv, plan, JudgeRequest{Goal: "the task"}, judge)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	out, _ := Apply(conv, plan, "Decisions:\n- read the file", fold.Survives, false)

	for i, msg := range out {
		if opensWithToolResult(msg) {
			t.Fatalf("message %d of the rebuild opens on a ToolResult: %v", i, out)
		}
		for _, part := range msg.Parts {
			if result, ok := part.(nacelle.ToolResult); ok {
				t.Errorf("a pruned result survived the rebuild: %+v", result)
			}
		}
	}
}

func TestGoalTextIsThePinnedTask(t *testing.T) {
	conv := judgeSample()

	if got := GoalText(conv, judgePlan(conv)); got != "the task" {
		t.Errorf("GoalText = %q, want the anchor's own text", got)
	}
}

// ledgerSample is a conversation already carrying a ledger, with a tool pair and
// a standalone turn left in its history for a pass to classify.
func ledgerSample() []nacelle.Message {
	return []nacelle.Message{
		nacelle.UserText("the task"),
		BuildLedger("", "Decisions:\n- keep `internal/compaction/ledger.go`"),
		callMessage("c1", "read"),
		resultMessage("c1", "read", strings.Repeat("x", 40_000)),
		nacelle.AssistantText("answer one"),
		nacelle.UserText("follow up"),
		nacelle.AssistantText("answer two"),
	}
}

// ledgerSpan is the span a plan gave the ledger, or the zero Span when it has
// none.
func ledgerSpan(plan []Span) Span {
	var ledger Span
	for _, span := range plan {
		if span.Zone == ZoneLedger {
			ledger = span
		}
	}
	return ledger
}

// I3a, the classification half: the plan gives the ledger its own zone, so it is
// never among the blocks the judge is asked about and its body is never rendered
// into one. A ledger handed to the judge would come back either pruned — a fact
// the session can no longer recover — or summarized into itself.
func TestClassifyIsNeverOfferedTheLedger(t *testing.T) {
	conv := ledgerSample()
	plan := judgePlan(conv)

	ledger := ledgerSpan(plan)
	if ledger.Start != 1 || ledger.End != 2 {
		t.Fatalf("ledger span = %+v, want the ledger alone at [1,2)", ledger)
	}
	body := Body(conv[ledger.Start])
	if body == "" {
		t.Fatal("the sample carries no ledger body, so nothing here would notice the judge seeing it")
	}

	blocks := Blocks(conv, plan)
	if len(blocks) == 0 {
		t.Fatal("no history blocks to classify")
	}
	for _, block := range blocks {
		if block.Start < ledger.End && ledger.Start < block.End {
			t.Errorf("block %+v covers the ledger span %+v, want it never a block", block, ledger)
		}
		if strings.Contains(block.Text, body) {
			t.Errorf("block %+v carries the ledger's own body, want the judge never shown it", block)
		}
	}
}

// I3a, the prune half: a judge that prunes every block it is shown still cannot
// reach the ledger. The pass folds what survives into the ledger's own body, so
// the facts it already held come back through the merge rather than being dropped
// with the history.
func TestPruningEverythingStillCannotReachTheLedger(t *testing.T) {
	conv := ledgerSample()
	plan := judgePlan(conv)
	blocks := Blocks(conv, plan)

	pruneEverything := make([]Verdict, len(blocks))
	for i := range pruneEverything {
		pruneEverything[i] = Verdict{Decision: Prune}
	}
	fold, err := Classify(t.Context(), conv, plan, JudgeRequest{Goal: "the task"}, &fakeJudge{verdicts: pruneEverything})
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}

	out, stats := Apply(conv, plan, "", fold.Survives, false)
	if stats.Refused || stats.Stale {
		t.Fatalf("stats = %+v, want the pass to land", stats)
	}
	ledgers := 0
	for _, msg := range out {
		if !IsLedger(msg) {
			continue
		}
		ledgers++
		if got := Body(msg); !strings.Contains(got, "internal/compaction/ledger.go") {
			t.Errorf("ledger body = %q, want what it already held kept through a prune-everything pass", got)
		}
	}
	if ledgers != 1 {
		t.Errorf("ledgers in the rebuild = %d, want exactly the one", ledgers)
	}
}
