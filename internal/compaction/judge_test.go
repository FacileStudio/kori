package compaction

import "testing"

// The prune gate is asymmetric: a probability over the threshold only prunes
// when the calibrated confidence also clears the floor. Everything else — under
// the threshold, unsure, or simply missing — keeps the block.
func TestDecidePrunesOnlyAtTheThreshold(t *testing.T) {
	probs := func(prune float64) map[string]float64 {
		return map[string]float64{optionPrune: prune, optionKeep: 1 - prune}
	}
	tests := []struct {
		name       string
		probs      map[string]float64
		choice     string
		confidence float64
		want       Decision
	}{
		{"below the threshold stays", probs(0.70), optionKeep, 0.95, Keep},
		{"at the threshold prunes", probs(0.85), optionPrune, 0.95, Prune},
		{"above the threshold prunes", probs(0.86), optionPrune, 0.95, Prune},
		{"low confidence keeps everything", probs(0.99), optionPrune, 0.40, Keep},
		{"an explicit ledger choice folds", probs(0.10), optionLedger, 0.95, Ledger},
		{"missing probabilities keep", nil, "", 0.95, Keep},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := decide(tc.probs, tc.choice, tc.confidence, 0.85); got.Decision != tc.want {
				t.Errorf("decide = %v, want %v", got.Decision, tc.want)
			}
		})
	}
}

func TestDecideCarriesTheNumbers(t *testing.T) {
	verdict := decide(map[string]float64{optionPrune: 0.9}, optionPrune, 0.95, 0.85)
	if verdict.PruneProb != 0.9 || verdict.Confidence != 0.95 {
		t.Errorf("verdict = %+v, want the probability and confidence carried", verdict)
	}
}

func TestDecisionNamesTheVerdicts(t *testing.T) {
	for decision, want := range map[Decision]string{Keep: "keep", Prune: "prune", Ledger: "ledger"} {
		if got := decision.String(); got != want {
			t.Errorf("Decision(%d).String() = %q, want %q", decision, got, want)
		}
	}
}
