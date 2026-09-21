package jev

import (
	"errors"
	"fmt"
	"net/http"
)

// UnauthorizedError is a 401: the key is missing, wrong, or not enabled for
// System One. It is not retryable, and it is the one failure an operator must
// see named rather than folded into a generic HTTP error.
type UnauthorizedError struct{ Detail string }

func (e *UnauthorizedError) Error() string {
	return "typesafe: unauthorized (401): " + e.Detail
}

// InvalidRequestError is a 422: the request itself is malformed — a bad option
// count, an unknown primitive. Retrying it unchanged cannot help.
type InvalidRequestError struct{ Detail string }

func (e *InvalidRequestError) Error() string {
	return "typesafe: invalid request (422): " + e.Detail
}

// HTTPError is any other non-2xx answer. 429 and 529 are retried by the client
// before this ever surfaces; a 5xx that survives every attempt comes back here.
type HTTPError struct {
	Status int
	Detail string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("typesafe: http %d: %s", e.Status, e.Detail)
}

// statusError maps a status to its typed error, so a caller can tell an auth
// problem from a malformed request without parsing a string.
func statusError(status int, detail string) error {
	switch status {
	case http.StatusUnauthorized:
		return &UnauthorizedError{Detail: detail}
	case http.StatusUnprocessableEntity:
		return &InvalidRequestError{Detail: detail}
	default:
		return &HTTPError{Status: status, Detail: detail}
	}
}

// retryable reports whether an error is the transient kind the client backs off
// and retries. Two kinds are: a rate limit or an overloaded backend, and a
// transport that never produced an answer at all — a dropped connection or a
// per-attempt timeout, which is exactly the blip a second attempt clears. Two are
// not: a typed 401 or 422 cannot succeed unchanged, and a 4xx or 5xx with a body
// is the endpoint speaking, not the network failing.
func retryable(err error) bool {
	var unauthorized *UnauthorizedError
	var invalid *InvalidRequestError
	if errors.As(err, &unauthorized) || errors.As(err, &invalid) {
		return false
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Status == http.StatusTooManyRequests || httpErr.Status == 529
	}
	return true
}
