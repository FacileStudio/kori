package compaction

import (
	"encoding/json"
	"strings"
	"testing"
)

// A choice's criteria is an object keyed by option, not a list of sentences. The
// endpoint validates the shape and answers 422 to a list, so a stub server — the
// only thing the rest of these tests talk to — accepts the broken request
// silently and the judge is dead against the real API while every test is green.
// That is exactly what happened, and this is the test that would have caught it.
func TestChoiceQuestionCriteriaEncodeAsAnObject(t *testing.T) {
	encoded, err := json.Marshal(choiceQuestion("block-1"))
	if err != nil {
		t.Fatalf("marshal the question: %v", err)
	}

	var wire struct {
		Type     string            `json:"type"`
		Criteria map[string]string `json:"criteria"`
		Options  []string          `json:"options"`
	}
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatalf("criteria did not decode as an object (%v): %s", err, encoded)
	}
	if wire.Type != "choice" {
		t.Errorf("type = %q, want the choice primitive", wire.Type)
	}
	if len(wire.Criteria) != len(wire.Options) {
		t.Errorf("criteria describe %d options but %d were offered: %s", len(wire.Criteria), len(wire.Options), encoded)
	}
	for _, option := range wire.Options {
		if wire.Criteria[option] == "" {
			t.Errorf("criteria have nothing for option %q, want every option described so the model knows its boundary", option)
		}
	}
}

// Every question names the block it is about. All of them are evaluated against
// one shared state and in isolation, so a question that does not say which block
// it means leaves the model to answer about "it" — and since the batch is
// otherwise identical it answers identically for every block, which reads as a
// confident unanimous verdict on a whole history. The key is the only link
// between a question and its block.
func TestEveryQuestionNamesItsOwnBlock(t *testing.T) {
	blocks := []Block{{Key: "block-a", Text: "a"}, {Key: "block-b", Text: "b"}}

	questions := questions(blocks)

	for _, block := range blocks {
		question, ok := questions[block.Key]
		if !ok {
			t.Fatalf("no question asked about %q", block.Key)
		}
		instructions, _ := question.Instructions.(string)
		if !strings.Contains(instructions, "["+block.Key+"]") {
			t.Errorf("instructions for %q = %q, want the block named so the model knows which one to judge", block.Key, instructions)
		}
	}
	first, _ := questions["block-a"].Instructions.(string)
	second, _ := questions["block-b"].Instructions.(string)
	if first == second {
		t.Error("two blocks were asked about with identical instructions, want each question pointed at its own block")
	}
}

// Every option the judge may answer must be one the criteria describe, and the
// destructive one has to be among them: an option with no criteria is an answer
// the model has to guess at, and a missing prune is a tier that can never drop
// anything.
func TestChoiceQuestionOffersEveryDecision(t *testing.T) {
	question := choiceQuestion("block-1")

	offered := map[string]bool{}
	for _, option := range question.Options {
		offered[option] = true
	}
	for _, want := range []string{optionKeep, optionPrune, optionLedger} {
		if !offered[want] {
			t.Errorf("options = %v, want %q offered", question.Options, want)
		}
	}
}
