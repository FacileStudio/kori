package jev

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// A 429 is transient: the client backs off and retries, so one rate limit does
// not cost the caller an answer.
func TestEvaluateRetriesTransientFailures(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		writeBody(t, w, `{"answers":{"a":{"choice":"keep","confidence":0.9,"probabilities":{"keep":1}}}}`)
	}))
	defer server.Close()

	response, err := newTestClient(server.URL).Evaluate(t.Context(), "state", map[string]Question{"a": {Type: "choice"}})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if got := requests.Load(); got != 2 {
		t.Errorf("requests = %d, want a retry after the 429", got)
	}
	if response.Answers["a"].Choice != "keep" {
		t.Errorf("answer = %+v, want the retried answer", response.Answers["a"])
	}
}

// A gateway's 502, 503 or 504 is a transport failure wearing a status code: the
// request never reached the endpoint, so it is retried like a rate limit rather
// than spending the attempt budget on a pass that has to fall back anyway.
func TestEvaluateRetriesAGatewayFailure(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		writeBody(t, w, `{"answers":{"a":{"choice":"keep","confidence":0.9,"probabilities":{"keep":1}}}}`)
	}))
	defer server.Close()

	response, err := newTestClient(server.URL).Evaluate(t.Context(), "state", map[string]Question{"a": {Type: "choice"}})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if got := requests.Load(); got != 2 {
		t.Errorf("requests = %d, want a retry after the gateway failed", got)
	}
	if response.Answers["a"].Choice != "keep" {
		t.Errorf("answer = %+v, want the retried answer", response.Answers["a"])
	}
}

// A transport error that never produced a response is transient too — a dropped
// connection is the blip a second attempt clears — so it is retried rather than
// reported as a dead endpoint. The drop is simulated by hijacking the socket and
// closing it unanswered, so the client sees an EOF instead of a status code.
func TestEvaluateRetriesTransportErrors(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				t.Error("response writer cannot be hijacked, cannot simulate a transport error")
				return
			}
			conn, _, err := hijacker.Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)
				return
			}
			if err := conn.Close(); err != nil {
				t.Errorf("close hijacked connection: %v", err)
			}
			return
		}
		writeBody(t, w, `{"answers":{"a":{"choice":"keep","confidence":0.9,"probabilities":{"keep":1}}}}`)
	}))
	defer server.Close()

	response, err := newTestClient(server.URL).Evaluate(t.Context(), "state", map[string]Question{"a": {Type: "choice"}})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if got := requests.Load(); got < 2 {
		t.Errorf("requests = %d, want the call to retry past the dropped connection", got)
	}
	if response.Answers["a"].Choice != "keep" {
		t.Errorf("answer = %+v, want the retried answer", response.Answers["a"])
	}
}

// A rate limit that never lifts gives up after the attempt budget rather than
// hammering the endpoint forever.
func TestEvaluateStopsAfterTheAttemptBudget(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}))
	defer server.Close()

	_, err := newTestClient(server.URL).Evaluate(t.Context(), "state", map[string]Question{"a": {Type: "choice"}})
	if err == nil {
		t.Fatal("err = nil, want the last transient failure")
	}
	if got := requests.Load(); got != 3 {
		t.Errorf("requests = %d, want the configured 3 attempts and no more", got)
	}
}

// A zero-value Client still makes one attempt. Without the floor an attempt
// budget of zero would leave the loop body unrun and hand back an empty response
// with a nil error, which a caller cannot tell from a judge that classified
// nothing — the difference between "nothing to prune" and "never asked".
func TestEvaluateMakesOneAttemptOnAZeroValueClient(t *testing.T) {
	var client Client

	if _, err := client.Evaluate(t.Context(), "state", map[string]Question{"a": {Type: "choice"}}); err == nil {
		t.Fatal("err = nil, want a transport error rather than an empty success")
	}
}

// The client's own wait honours the caller's context, so a judge can never hold
// a pass open past its deadline.
func TestEvaluateHonoursTheContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		writeBody(t, w, `{"answers":{}}`)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()

	if _, err := newTestClient(server.URL).Evaluate(ctx, "state", map[string]Question{"a": {Type: "choice"}}); err == nil {
		t.Fatal("err = nil, want the deadline read as a failure")
	}
}
