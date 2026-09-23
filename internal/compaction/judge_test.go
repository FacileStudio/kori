package compaction

import (
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
)

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

// The threshold is the number that decides a deletion, so an unusable one refuses
// to prune instead of treating every probability as over it. It is the shape a
// config arrives in: a key nobody mentioned is a zero, and a zero would be
// cleared by any answer the judge happened to give.
func TestDecideNeverPrunesBelowAnUnusableThreshold(t *testing.T) {
	probs := map[string]float64{optionPrune: 0.9, optionKeep: 0.1}

	for _, threshold := range []float64{0, -0.5} {
		if got := decide(probs, optionPrune, 0.95, threshold); got.Decision != Keep {
			t.Errorf("threshold %v pruned at %v prune probability, want keep", threshold, probs[optionPrune])
		}
	}
	if got := decide(probs, optionPrune, 0.95, DefaultPruneThreshold); got.Decision != Prune {
		t.Errorf("threshold %v = %v, want a usable threshold to still prune", DefaultPruneThreshold, got.Decision)
	}
}

func TestDecideCarriesTheNumbers(t *testing.T) {
	verdict := decide(map[string]float64{optionPrune: 0.9}, optionPrune, 0.95, 0.85)
	if verdict.PruneProb != 0.9 || verdict.Confidence != 0.95 {
		t.Errorf("verdict = %+v, want the probability and confidence carried", verdict)
	}
}

// A garbage body prunes nothing. An endpoint that answers 200 with something
// that is not an answer — a truncated one, an answer with no confidence, a choice
// nobody defined — must not turn into a deletion: the malformed case comes back as
// an error with all-keep verdicts, so a caller that ignores the error still drops
// nothing, and the shaped-but-empty cases fall to the confidence floor.
func TestJevJudgePrunesNothingOnAMalformedAnswer(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"a truncated body", `{"answers":{"block-1":{"choice":"prune",`, true},
		{"an answer that is not JSON", "not an answer at all", true},
		{"an answer with no confidence", `{"answers":{"block-1":{"choice":"prune","probabilities":{"prune":0.99}}}}`, false},
		{"a choice nobody defined", `{"answers":{"block-1":{"choice":"delete","confidence":0.99,"probabilities":{"delete":0.99}}}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertNoPrune(t, tc.body, tc.wantErr)
		})
	}
}

// assertNoPrune drives a real judge against a stub endpoint answering with the
// given body and checks that every block the judge was asked about came back
// keep, whatever the body did to the call.
func assertNoPrune(t *testing.T, body string, wantErr bool) {
	t.Helper()
	var asked map[string]json.RawMessage
	var requests atomic.Int32
	server := answerServer(t, body, &asked, &requests)

	judge := NewJevJudge(JudgeConfig{Enabled: true, BaseURL: server.URL, PruneThreshold: 0.85})
	blocks := Blocks(judgeSample(), judgePlan(judgeSample()))

	verdicts, err := judge.Classify(t.Context(), "the task", blocks)
	if gotErr := err != nil; gotErr != wantErr {
		t.Fatalf("Classify error = %v, want an error: %v", err, wantErr)
	}
	if len(verdicts) != len(blocks) {
		t.Fatalf("verdicts = %d, want one per block so a caller can index them", len(verdicts))
	}
	for i, verdict := range verdicts {
		if verdict.Decision != Keep {
			t.Errorf("verdict %d = %v for a garbage answer, want keep", i, verdict.Decision)
		}
	}
}

// One block cannot dominate the request it travels in. A block holding a whole
// tool result is cut to maxBlockText and the cut is marked, so a batch of dozens
// of blocks still reaches the judge whole in count and in shape — and the cut is
// per block, so the small turns beside it are not touched.
func TestBlockTextIsClampedToOneCap(t *testing.T) {
	conv := ledgerSample()
	plan := judgePlan(conv)

	blocks := Blocks(conv, plan)
	if len(blocks) != 2 {
		t.Fatalf("blocks = %d, want the tool pair and the standalone turn", len(blocks))
	}

	text := blocks[0].Text
	if want := maxBlockText + len("…"); len(text) != want {
		t.Errorf("block text = %d bytes for a %d-byte result, want the %d-byte cap plus its marker",
			len(text), 40_000, want)
	}
	if !strings.HasSuffix(text, "…") {
		t.Errorf("block text ends %q, want the cut marked", text[len(text)-4:])
	}
	if !strings.HasPrefix(text, "tool call read") {
		t.Errorf("block text = %q, want the head of the block kept and only its tail cut", text[:40])
	}
	if small := blocks[1].Text; strings.HasSuffix(small, "…") {
		t.Errorf("small block text = %q, want the cap applied per block", small)
	}
}

func TestDecisionNamesTheVerdicts(t *testing.T) {
	for decision, want := range map[Decision]string{Keep: "keep", Prune: "prune", Ledger: "ledger"} {
		if got := decision.String(); got != want {
			t.Errorf("Decision(%d).String() = %q, want %q", decision, got, want)
		}
	}
}
