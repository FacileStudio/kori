package jev

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const batchAnswer = `{"model":"jev-latest","answers":{
	"a":{"choice":"prune","confidence":0.9,"probabilities":{"prune":0.9,"keep":0.1}},
	"b":{"choice":"keep","confidence":0.8,"probabilities":{"keep":0.8}}},
	"usage":{"input_tokens":120}}`

// recorded is what a stub endpoint saw: the requests, the auth header, the path,
// and the last body's state and questions.
type recorded struct {
	requests  atomic.Int32
	auth      string
	path      string
	state     string
	questions map[string]Question
}

// recordingServer answers every request with body and records what it was asked,
// so a test can assert on both sides of the wire.
func recordingServer(t *testing.T, body string, rec *recorded) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.requests.Add(1)
		rec.auth, rec.path = r.Header.Get("Authorization"), r.URL.Path
		var sent struct {
			State     string              `json:"state"`
			Questions map[string]Question `json:"questions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Errorf("decode request: %v", err)
		}
		rec.state, rec.questions = sent.State, sent.Questions
		writeBody(t, w, body)
	}))
	t.Cleanup(server.Close)
	return server
}

func writeBody(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(body)); err != nil {
		t.Errorf("write response: %v", err)
	}
}

func newTestClient(url string) *Client {
	return New(Config{BaseURL: url, APIKey: "test-key", Attempts: 3, Backoff: time.Millisecond})
}

func twoQuestions() map[string]Question {
	return map[string]Question{
		"a": {Type: "choice", Instructions: "decide", Options: []string{"keep", "prune"}},
		"b": {Type: "choice", Instructions: "decide", Options: []string{"keep", "prune"}},
	}
}

// One call carries every question, all evaluated against the same state, and the
// answers come back keyed by the name each was asked under.
func TestEvaluateSendsOneBatchedRequest(t *testing.T) {
	var rec recorded
	server := recordingServer(t, batchAnswer, &rec)

	response, err := newTestClient(server.URL).Evaluate(t.Context(), "state text", twoQuestions())
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if got := rec.requests.Load(); got != 1 {
		t.Errorf("requests = %d, want the two questions batched into one", got)
	}
	if rec.auth != "Bearer test-key" {
		t.Errorf("Authorization = %q, want the bearer key", rec.auth)
	}
	if rec.path != endpoint {
		t.Errorf("path = %q, want %q", rec.path, endpoint)
	}
	if rec.state != "state text" || len(rec.questions) != 2 || rec.questions["a"].Type != "choice" {
		t.Errorf("sent = %q / %v, want the state and both choice questions", rec.state, rec.questions)
	}
	if response.Answers["a"].Choice != "prune" || response.Answers["a"].Probabilities["prune"] != 0.9 {
		t.Errorf("answer a = %+v, want the choice and its probabilities", response.Answers["a"])
	}
	if response.Usage.InputTokens != 120 {
		t.Errorf("usage = %+v, want the billed input", response.Usage)
	}
}

// A 401 and a 422 are typed, so a caller can tell an auth problem from a
// malformed request without reading a string.
func TestEvaluateReturnsTypedErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		check  func(error) bool
	}{
		{"unauthorized", http.StatusUnauthorized, func(err error) bool {
			var target *UnauthorizedError
			return errors.As(err, &target)
		}},
		{"invalid request", http.StatusUnprocessableEntity, func(err error) bool {
			var target *InvalidRequestError
			return errors.As(err, &target)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "nope", tc.status)
			}))
			defer server.Close()

			_, err := newTestClient(server.URL).Evaluate(t.Context(), "state", map[string]Question{"a": {Type: "choice"}})
			if err == nil || !tc.check(err) {
				t.Fatalf("err = %v, want a typed %s error", err, tc.name)
			}
		})
	}
}

// The vendor's own environment key is read when the config carries none, so a
// key already exported for TypeSafe needs no second copy.
func TestClientReadsTheTypesafeEnvironmentKey(t *testing.T) {
	var rec recorded
	server := recordingServer(t, `{"answers":{}}`, &rec)

	t.Setenv("TYPESAFE_API_KEY", "from-typesafe")
	client := New(Config{BaseURL: server.URL, Attempts: 1})
	if _, err := client.Evaluate(t.Context(), "state", map[string]Question{"a": {Type: "choice"}}); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !strings.Contains(rec.auth, "from-typesafe") {
		t.Errorf("Authorization = %q, want the environment key", rec.auth)
	}
}
