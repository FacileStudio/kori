package compaction

import "context"

// Decision is what the judge wants done with one history block.
type Decision uint8

const (
	// Keep leaves the block in the conversation verbatim.
	Keep Decision = iota
	// Prune drops the whole atomic block. It is the only destructive verdict,
	// so it is the only one the confidence gate guards.
	Prune
	// Ledger folds the block's facts into the state ledger, compressed by the
	// summarizer, and drops the block itself.
	Ledger
)

// String names a decision the way a report or a test reads it.
func (d Decision) String() string {
	switch d {
	case Prune:
		return "prune"
	case Ledger:
		return "ledger"
	default:
		return "keep"
	}
}

// Verdict is one block's classification, carrying the numbers behind it so a
// caller can log or re-threshold without asking again.
type Verdict struct {
	Decision   Decision
	PruneProb  float64
	Confidence float64
}

// Judge classifies history blocks against the task the session started from.
// It decides keep, prune or ledger per block and writes no prose: the ledger is
// the summarizer's job, fed only the blocks this tags.
type Judge interface {
	Classify(ctx context.Context, goal string, blocks []Block) ([]Verdict, error)
}

// JudgeConfig is the opt-in classifier's settings. It is off by default:
// enabling it sends conversation history — source code, possibly secrets — to a
// third party, so it is never a shipped default.
type JudgeConfig struct {
	Enabled        bool
	Model          string
	BaseURL        string
	APIKey         string
	PruneThreshold float64
	MaxBlocks      int
}

// ConfidenceFloor is how sure the judge must be before a verdict may change the
// conversation. It is deliberately a floor rather than a second ratio: the
// confidence is calibrated, so a low one means the distribution is flat and the
// safe answer is to keep everything.
const ConfidenceFloor = 0.6

const (
	optionKeep   = "keep"
	optionPrune  = "prune"
	optionLedger = "ledger"
)

// decide turns one block's answer into a verdict. The rule is asymmetric on
// purpose: a prune needs both a probability over the threshold and a confidence
// over the floor, while anything ambiguous, missing or empty is kept. The
// failure mode of a bad prune is a conversation that lost a fact; the failure
// mode of a bad keep is a little more context.
func decide(probabilities map[string]float64, choice string, confidence, threshold float64) Verdict {
	pruneProb, hasPrune := probabilities[optionPrune]
	verdict := Verdict{Decision: Keep, PruneProb: pruneProb, Confidence: confidence}
	if confidence < ConfidenceFloor {
		return verdict
	}
	if hasPrune && pruneProb >= threshold {
		verdict.Decision = Prune
		return verdict
	}
	if choice == optionLedger {
		verdict.Decision = Ledger
	}
	return verdict
}

// keepAll is the safe answer: everything stays where it is.
func keepAll(n int) []Verdict {
	out := make([]Verdict, n)
	for i := range out {
		out[i] = Verdict{Decision: Keep}
	}
	return out
}
