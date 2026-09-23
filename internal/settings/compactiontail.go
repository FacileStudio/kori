package settings

import "fmt"

// validateTail rejects a tail bound or a window figure that cannot work. It runs
// on the resolved settings, so only a value a layer actually wrote down is
// judged — a job file that mentions one ratio and nothing else keeps the shipped
// tail rather than failing on the fields it never named.
//
// Each rejected value is a number a zero would quietly disable rather than a
// preference. `keep_tokens: 0` leaves the tail sized by keep_turns alone,
// `window_tokens: 0` reads as "ask the backend" — which is what leaving it out
// already means — and a reserve that leaves under half of a declared window
// usable shrinks the ladder to a single rung: usable collapses toward one token,
// every tier trips at once, and the session sits at hard while the config still
// reads as enabled. That is the same failure `soft_ratio: 0` used to have, and
// failing at load is what keeps it from being found out later.
//
// The reserve and window are only compared with each other when both were
// written down here. The window usually comes from the backend, which this layer
// cannot see, so ResolveBudget cuts a reserve that would swallow it instead.
func validateTail(c Compaction) error {
	if c.KeepTurns != nil && *c.KeepTurns < 1 {
		return &ParseError{Path: "limits.compaction.keep_turns", Err: fmt.Errorf(
			"want a count of at least 1, got %d", *c.KeepTurns)}
	}
	if c.AnchorMessages != nil && *c.AnchorMessages < 1 {
		return &ParseError{Path: "limits.compaction.anchor_messages", Err: fmt.Errorf(
			"want a count of at least 1, got %d", *c.AnchorMessages)}
	}
	for _, bound := range []struct {
		key   string
		value *int64
	}{{"keep_tokens", c.KeepTokens}, {"reserve_tokens", c.ReserveTokens}, {"window_tokens", c.WindowTokens}} {
		if bound.value != nil && *bound.value < 1 {
			return &ParseError{Path: "limits.compaction." + bound.key, Err: fmt.Errorf(
				"want a positive token count, got %d — leave it out and kori derives it", *bound.value)}
		}
	}
	if c.ReserveTokens != nil && c.WindowTokens != nil && *c.ReserveTokens > *c.WindowTokens/2 {
		return &ParseError{Path: "limits.compaction.reserve_tokens", Err: fmt.Errorf(
			"want a reserve that leaves at least half of window_tokens usable, got %d of %d", *c.ReserveTokens, *c.WindowTokens)}
	}
	return nil
}
