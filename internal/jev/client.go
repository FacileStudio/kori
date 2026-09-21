package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is TypeSafe's own host; BaseURL exists for a proxy or a
	// test server.
	DefaultBaseURL = "https://api.typesafe.ai"

	// DefaultModel is the current System One model name. JEV is early access, so
	// a caller may override it without a code change.
	DefaultModel = "jev-latest"

	// endpoint is the one path this client speaks.
	endpoint = "/v1/systemone"

	defaultTimeout  = 10 * time.Second
	defaultAttempts = 4
	defaultBackoff  = 250 * time.Millisecond

	// maxBody caps what one answer may be read into memory; a JEV response is
	// small, so anything larger is a proxy error page, not an answer.
	maxBody = 1 << 20
)

// Config is one client's settings. Empty fields take the defaults; APIKey falls
// back to TYPESAFE_API_KEY when it is unset, so a key already exported for the
// vendor needs no second copy.
type Config struct {
	BaseURL  string
	APIKey   string
	Model    string
	Timeout  time.Duration
	Attempts int
	Backoff  time.Duration
}

// Client is the System One client. It is safe for concurrent use; the judge
// only ever calls it from one pass goroutine.
type Client struct {
	baseURL  string
	apiKey   string
	model    string
	http     *http.Client
	attempts int
	backoff  time.Duration
}

// New builds a client, filling every unset field from the defaults.
func New(c Config) *Client {
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	if c.Model == "" {
		c.Model = DefaultModel
	}
	if c.APIKey == "" {
		c.APIKey = os.Getenv("TYPESAFE_API_KEY")
	}
	if c.Timeout <= 0 {
		c.Timeout = defaultTimeout
	}
	if c.Attempts <= 0 {
		c.Attempts = defaultAttempts
	}
	if c.Backoff <= 0 {
		c.Backoff = defaultBackoff
	}
	return &Client{
		baseURL:  strings.TrimRight(c.BaseURL, "/"),
		apiKey:   c.APIKey,
		model:    c.Model,
		http:     &http.Client{Timeout: c.Timeout},
		attempts: c.Attempts,
		backoff:  c.Backoff,
	}
}

// Evaluate sends one state and its questions and returns every answer, all of
// them computed against the same state in a single call. A transient failure —
// a 429 or a 529 — is retried with exponential backoff up to Attempts times;
// anything else comes back as its typed error.
func (c *Client) Evaluate(ctx context.Context, state any, questions map[string]Question) (Response, error) {
	payload, err := json.Marshal(request{State: state, Model: c.model, Questions: questions})
	if err != nil {
		return Response{}, err
	}
	var last error
	for attempt := range c.attempts {
		response, err := c.post(ctx, payload)
		if err == nil {
			return response, nil
		}
		last = err
		if !retryable(err) || attempt == c.attempts-1 {
			break
		}
		if err := c.pause(ctx, attempt); err != nil {
			return Response{}, err
		}
	}
	return Response{}, last
}

// post is one attempt: one request, one read, one decoded response or typed
// error. The bearer key is the only credential this package ever touches and is
// never logged.
func (c *Client) post(ctx context.Context, payload []byte) (Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+endpoint, bytes.NewReader(payload))
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(res.Body, maxBody))
	if err != nil {
		return Response{}, err
	}
	if res.StatusCode != http.StatusOK {
		return Response{}, statusError(res.StatusCode, strings.TrimSpace(string(body)))
	}

	var out Response
	if err := json.Unmarshal(body, &out); err != nil {
		return Response{}, &HTTPError{Status: res.StatusCode, Detail: "malformed response: " + err.Error()}
	}
	return out, nil
}

// pause waits out one backoff interval, doubling per attempt, and gives up the
// moment the caller's context does.
func (c *Client) pause(ctx context.Context, attempt int) error {
	timer := time.NewTimer(c.backoff << attempt)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
