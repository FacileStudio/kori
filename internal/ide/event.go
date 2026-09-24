package ide

import (
	"encoding/json"
	"maps"
)

// Event is one object a session sends an editor: the protocol version, a type
// naming what it carries, and that type's fields.
//
// The fields are a map rather than a struct because this protocol grows by
// adding a field to an existing type, and a receiver ignores the ones it does
// not know. A struct would make every addition a change to both sides at once.
type Event struct {
	V      int
	T      string
	Fields map[string]any
}

// MarshalJSON renders the event as the flat object the protocol describes,
// with the version and the type alongside the fields they belong to.
func (e Event) MarshalJSON() ([]byte, error) {
	out := make(map[string]any, len(e.Fields)+2)
	maps.Copy(out, e.Fields)
	out["v"] = e.V
	out["t"] = e.T
	return json.Marshal(out)
}

// The words a run's end is reported with.
const (
	EndTurn       = "end_turn"
	MaxIterations = "max_iterations"
	Cancelled     = "cancelled"
	RunError      = "error"
)

// Hello is the first line a session sends a client that has just connected.
func Hello(pid int, root, session, model, version string) Event {
	return Event{V: Protocol, T: "hello", Fields: map[string]any{
		"pid": pid, "root": root, "session": session, "model": model, "version": version,
	}}
}

// Turn reports that a model turn started. n counts from 1.
func Turn(n int) Event {
	return Event{V: Protocol, T: "turn", Fields: map[string]any{"n": n}}
}

// Done reports that the run finished, and why.
func Done(reason string, cost float64) Event {
	return Event{V: Protocol, T: "done", Fields: map[string]any{"reason": reason, "cost": cost}}
}

// Err reports a protocol-level problem. The connection closes after it.
func Err(reason string) Event {
	return Event{V: Protocol, T: "error", Fields: map[string]any{"reason": reason}}
}
