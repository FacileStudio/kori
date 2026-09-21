package compaction

import (
	"context"
	"strings"

	"github.com/FacileStudio/kori/internal/jev"
)

// jevJudge is the System One adapter: one state, one choice question per block,
// one batched call. It is the only place in this package that knows a network
// exists.
type jevJudge struct {
	client    *jev.Client
	threshold float64
	maxBlocks int
}

// NewJevJudge builds the opt-in classifier, or nil when the judge is off — a
// small, tool-free surface the TUI can hold and test without a network.
func NewJevJudge(cfg JudgeConfig) Judge {
	if !cfg.Enabled {
		return nil
	}
	return &jevJudge{
		client:    jev.New(jev.Config{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.Model}),
		threshold: pruneThreshold(cfg.PruneThreshold),
		maxBlocks: cfg.MaxBlocks,
	}
}

// pruneThreshold fills in a threshold an adapter was handed none of. A config
// that never mentions the key carries a zero, and a zero would be cleared by any
// prune probability at all — so the value a half-filled config falls back to is
// the shipped default, not "prune whenever the judge is not sure".
func pruneThreshold(threshold float64) float64 {
	if threshold <= 0 || threshold > 1 {
		return DefaultPruneThreshold
	}
	return threshold
}

// Classify asks every block's question in one call and maps the answers back.
// Blocks past max_blocks_per_call are not asked about and are folded instead:
// the summarizer can compress them, where an unclassified prune could not be
// taken back. A failed call returns all-keep verdicts and the error, so a caller
// that ignores the error still prunes nothing.
func (j *jevJudge) Classify(ctx context.Context, goal string, blocks []Block) ([]Verdict, error) {
	if len(blocks) == 0 {
		return nil, nil
	}
	verdicts := make([]Verdict, len(blocks))
	first := j.overflow(blocks, verdicts)
	asked := blocks[first:]

	response, err := j.client.Evaluate(ctx, state(goal, asked), questions(asked))
	if err != nil {
		return keepAll(len(blocks)), err
	}
	for i, block := range asked {
		answer := response.Answers[block.Key]
		verdicts[first+i] = decide(answer.Probabilities, answer.Choice, answer.Confidence, j.threshold)
	}
	return verdicts, nil
}

// overflow marks the oldest blocks that do not fit the batch as ledger material
// and returns the index the call may start at. It is the answer to "too much
// history for one call": classify the recent end where a prune is most useful,
// and let the summarizer compress the rest.
func (j *jevJudge) overflow(blocks []Block, verdicts []Verdict) int {
	if j.maxBlocks <= 0 || len(blocks) <= j.maxBlocks {
		return 0
	}
	first := len(blocks) - j.maxBlocks
	for i := range first {
		verdicts[i] = Verdict{Decision: Ledger}
	}
	return first
}

// state renders the goal and the blocks into the text every question is asked
// against. The goal is the pinned task, so a block is judged by whether the
// original task still needs it rather than by how recent it looks.
//
// The block text is untrusted — it is tool output and file contents, and a block
// can argue for its own verdict. That is why the only destructive decision is
// gated twice in decide (the prune probability and the calibrated confidence),
// and why anything unclear falls back to Keep. Widening what is sent here, or
// what one verdict is allowed to do, means revisiting that gate rather than this
// function.
func state(goal string, blocks []Block) string {
	var b strings.Builder
	b.WriteString("GOAL:\n")
	b.WriteString(goal)
	b.WriteString("\n\nHISTORY BLOCKS:\n")
	for _, block := range blocks {
		b.WriteString("\n[")
		b.WriteString(block.Key)
		b.WriteString("]\n")
		b.WriteString(block.Text)
	}
	return b.String()
}

// questions is one choice question per block, all answered in one call. Adding
// more questions does not cost a second round trip for JEV.
func questions(blocks []Block) map[string]jev.Question {
	out := make(map[string]jev.Question, len(blocks))
	for _, block := range blocks {
		out[block.Key] = choiceQuestion()
	}
	return out
}

// choiceQuestion spells the three options out as the criteria: a decision, a
// constraint or a dead end. The value of JEV is that it decides rather than
// writes, so the criteria are the whole contract with it.
func choiceQuestion() jev.Question {
	return jev.Question{
		Type:         "choice",
		Instructions: "Decide what this block of an agent's working history is worth to the task above.",
		Criteria: []string{
			optionKeep + ": the block is live state the agent is still working from",
			optionLedger + ": a decision, constraint, artifact or dead end worth keeping in compressed form",
			optionPrune + ": superseded output or a dead end with no lasting value",
		},
		Options: []string{optionKeep, optionPrune, optionLedger},
	}
}
