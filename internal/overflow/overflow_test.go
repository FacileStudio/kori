package overflow

import (
	"errors"
	"testing"
)

// The messages are the real shapes, not paraphrases: each one is what a provider
// printed when a request outgrew the window, and a classifier that only
// recognises Anthropic's would leave the OpenAI-compatible runner silently
// unrecovered.
func TestDetectRecognisesTheProvidersOwnWording(t *testing.T) {
	refusals := []string{
		"prompt is too long: 213000 tokens > 200000 maximum",
		"This model's maximum context length is 128000 tokens. However, your messages resulted in 145000 tokens.",
		"maximum context length is 131072 tokens, you requested 140000",
		"invalid_request_error: context_length_exceeded",
		"The input token count (1050000) exceeds the maximum number of tokens allowed (1000000)",
		"Prompt contains 40000 tokens, exceeding the maximum context length",
		"too many tokens in the request",
	}

	for _, message := range refusals {
		if !Detect(errors.New(message)) {
			t.Errorf("Detect(%q) = false, want a context-length rejection", message)
		}
	}
}

// Everything else is left alone: a recovered rate limit is still a rate limit,
// and compacting for one would spend a pass on nothing.
func TestDetectLeavesOtherFailuresAlone(t *testing.T) {
	others := []string{
		"429 Too Many Requests: rate limit exceeded",
		"401 invalid api key",
		"context deadline exceeded",
		"the tool returned no output",
	}

	for _, message := range others {
		if Detect(errors.New(message)) {
			t.Errorf("Detect(%q) = true, want it left alone", message)
		}
	}
	if Detect(nil) {
		t.Error("Detect(nil) = true, want false")
	}
}
