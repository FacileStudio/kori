// Package overflow recognises the one provider failure a harness can answer by
// itself: a request the model's context window was too small to hold.
//
// No backend exposes it as a type a client can match, so this is a phrase check
// over the error text. That makes it a heuristic, and it is used as one — a false
// positive costs a compaction pass and one retry, and the retry is spent once per
// turn, so a wrong guess is bounded work rather than a loop. A false negative
// costs nothing beyond the failure the reader would have seen anyway, which is
// why the vocabulary is the providers' own wording rather than a guess at it.
package overflow

import "strings"

// tooLong is the vocabulary the providers print for a rejected request. Each
// phrase was taken from a real refusal rather than invented:
// Anthropic's "prompt is too long: 213000 tokens > 200000 maximum", the
// OpenAI-compatible "This model's maximum context length is 128000 tokens",
// OpenRouter's per-endpoint "maximum context length", Gemini's "input token count
// ... exceeds the maximum", and Mistral's "exceeding the maximum context length".
// "too many tokens" is the shortest shape the others collapse to.
var tooLong = []string{
	"prompt is too long",
	"maximum context length",
	"context length",
	"context_length",
	"too many tokens",
	"input token count",
	"exceeds the maximum",
	"exceeding the maximum",
}

// Detect reports whether err is a context-length rejection, and so whether
// compacting and retrying the run could succeed where the request just failed.
func Detect(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	for _, phrase := range tooLong {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}
