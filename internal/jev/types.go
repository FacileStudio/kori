// Package jev is the TypeSafe System One client: typed questions in, calibrated
// decisions out, and no generated text at all. It is a single integration, the
// same shape internal/diagnostics and internal/usage take, and it carries no SDK
// dependency — the wire format is three fields and a JSON body.
//
// JEV is early access. Model names, availability and rate limits can move, so
// the caller treats any failure as a reason to fall back, never as an error to
// surface as a broken conversation. See internal/compaction's judge.
package jev

// Question is one typed ask. Instructions and Criteria are `any` on purpose:
// the API takes a string or a list for each, and this client does not narrow a
// shape the vendor owns.
type Question struct {
	Type         string   `json:"type"`
	Instructions any      `json:"instructions"`
	Criteria     any      `json:"criteria,omitempty"`
	Options      []string `json:"options,omitempty"`
}

// Answer is one question's result. Answer is a "choice": the picked option, the
// calibrated confidence and the option probabilities. Noul carries the other
// primitive's single number and stays zero for a choice.
type Answer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
	Noul          float64            `json:"noul"`
}

// Usage is what one call billed. Output is free on JEV, but the field is read
// anyway so the client never has to guess which side it was on.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Response is one Evaluate call's answer: the model that served it, every
// question's result keyed by the name it was asked under, and the usage.
type Response struct {
	Model   string            `json:"model"`
	Answers map[string]Answer `json:"answers"`
	Usage   Usage             `json:"usage"`
}

// request is the wire body. Every question is evaluated in parallel against the
// same state in one call, which is the fact the whole judge design leans on.
type request struct {
	State     any                 `json:"state"`
	Model     string              `json:"model"`
	Questions map[string]Question `json:"questions"`
}
