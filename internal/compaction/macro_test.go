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
	out, _ := Apply(conv, plan, "Decisions:\n- read the file", fold.Survives)

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
