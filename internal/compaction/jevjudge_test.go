package compaction

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// answerServer is a stub systemone endpoint: it records the questions it was
// asked and answers with the scripted body.
func answerServer(t *testing.T, body string, asked *map[string]json.RawMessage, requests *atomic.Int32) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		var request struct {
			Questions map[string]json.RawMessage `json:"questions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		*asked = request.Questions
		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// N blocks become N questions in one batched call, and each answer maps back to
// the block it was asked about.
func TestJevJudgeAsksOneBatchedQuestionPerBlock(t *testing.T) {
	var asked map[string]json.RawMessage
	var requests atomic.Int32
	server := answerServer(t, `{"answers":{
		"block-1":{"choice":"prune","confidence":0.95,"probabilities":{"prune":0.9,"keep":0.1}},
		"block-3":{"choice":"ledger","confidence":0.9,"probabilities":{"ledger":0.8}}}}`, &asked, &requests)

	judge := NewJevJudge(JudgeConfig{Enabled: true, BaseURL: server.URL, APIKey: "k", PruneThreshold: 0.85, MaxBlocks: 64})
	conv := judgeSample()

	verdicts, err := judge.Classify(t.Context(), "the task", Blocks(conv, judgePlan(conv)))
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}

	if got := requests.Load(); got != 1 {
		t.Errorf("requests = %d, want the questions batched into one", got)
	}
	if len(asked) != 2 {
		t.Errorf("questions = %d, want one per block", len(asked))
	}
	if len(verdicts) != 2 || verdicts[0].Decision != Prune || verdicts[1].Decision != Ledger {
		t.Errorf("verdicts = %+v, want the answers mapped back to their blocks", verdicts)
	}
}

// A call that fails degrades to all-keep and reports the error, so a 429 costs
// a fallback rather than a prune nobody can undo.
func TestJevJudgeDegradesOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}))
	defer server.Close()

	judge := NewJevJudge(JudgeConfig{Enabled: true, BaseURL: server.URL, PruneThreshold: 0.85})
	blocks := Blocks(judgeSample(), judgePlan(judgeSample()))

	verdicts, err := judge.Classify(t.Context(), "the task", blocks)
	if err == nil {
		t.Fatal("err = nil, want the rate limit reported")
	}
	for i, verdict := range verdicts {
		if verdict.Decision != Keep {
			t.Errorf("verdict %d = %v after a failed call, want keep", i, verdict.Decision)
		}
	}
}

// Only the newest max_blocks_per_call blocks are asked about; the rest are
// folded, which the summarizer can undo by never losing the content.
func TestJevJudgeFoldsWhatItCannotBatch(t *testing.T) {
	var asked map[string]json.RawMessage
	var requests atomic.Int32
	server := answerServer(t, `{"answers":{"block-3":{"choice":"prune","confidence":0.95,"probabilities":{"prune":0.9}}}}`, &asked, &requests)

	judge := NewJevJudge(JudgeConfig{Enabled: true, BaseURL: server.URL, MaxBlocks: 1, PruneThreshold: 0.85})
	blocks := Blocks(judgeSample(), judgePlan(judgeSample()))

	verdicts, err := judge.Classify(t.Context(), "the task", blocks)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if len(asked) != 1 {
		t.Errorf("questions = %d, want only the newest block asked", len(asked))
	}
	if verdicts[0].Decision != Ledger {
		t.Errorf("overflow verdict = %v, want the unasked block folded", verdicts[0].Decision)
	}
	if verdicts[1].Decision != Prune {
		t.Errorf("asked verdict = %v, want the newest block's answer", verdicts[1].Decision)
	}
}

// A judge built without a prune threshold falls back to the shipped one rather
// than to "prune on any probability at all": a half-filled config must not
// invert the one destructive verdict. The answer here is a 0.20 prune
// probability, which clears a threshold of zero and nothing else.
func TestJevJudgeFallsBackToTheDefaultThreshold(t *testing.T) {
	var asked map[string]json.RawMessage
	var requests atomic.Int32
	server := answerServer(t, `{"answers":{"block-1":{"choice":"keep","confidence":0.95,"probabilities":{"prune":0.20,"keep":0.80}}}}`, &asked, &requests)

	judge := NewJevJudge(JudgeConfig{Enabled: true, BaseURL: server.URL, MaxBlocks: 64})
	conv := judgeSample()

	verdicts, err := judge.Classify(t.Context(), "the task", Blocks(conv, judgePlan(conv)))
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if verdicts[0].Decision != Keep {
		t.Errorf("verdict = %v, want keep — a 0.20 prune probability must not clear the shipped %v threshold",
			verdicts[0].Decision, DefaultPruneThreshold)
	}
}

// A disabled judge builds nothing, so an off setting cannot accidentally reach
// the network.
func TestNewJevJudgeIsNilWhenDisabled(t *testing.T) {
	if judge := NewJevJudge(JudgeConfig{}); judge != nil {
		t.Errorf("judge = %v, want nil while the setting is off", judge)
	}
}
