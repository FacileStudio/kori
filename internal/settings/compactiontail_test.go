package settings

import (
	"strings"
	"testing"
)

// The reserve is what a declared window holds back for the turn's own answer, so
// what it leaves behind is the window the ladder actually measures. A reserve one
// token under the window leaves a single token to measure against: every rung
// trips at once and the session is pinned at the hard tier while the config still
// reads as enabled. Half is the floor, because a reserve that takes more than half
// leaves the tiers crowded into a minority of the window; the shipped reserve is a
// fifth, so nothing about the defaults moves. It is checked through the real
// loader rather than the struct, so the file and the environment are refused the
// same way.
func TestValidateTailRefusesAReserveThatStarvesTheWindow(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		valid bool
	}{
		{"a reserve one under the window", "limits:\n  compaction:\n    window_tokens: 200000\n    reserve_tokens: 199999\n", false},
		{"a reserve that fills the window", "limits:\n  compaction:\n    window_tokens: 200000\n    reserve_tokens: 200000\n", false},
		{"a reserve just over half the window", "limits:\n  compaction:\n    window_tokens: 200000\n    reserve_tokens: 100001\n", false},
		{"a reserve at half the window", "limits:\n  compaction:\n    window_tokens: 200000\n    reserve_tokens: 100000\n", true},
		{"the shipped fifth of the window", "limits:\n  compaction:\n    window_tokens: 200000\n    reserve_tokens: 40000\n", true},
		{"a reserve with no window to compare against", "limits:\n  compaction:\n    reserve_tokens: 40000\n", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			written(t, tc.body)
			_, err := settings(Config{})
			if err == nil && !tc.valid {
				t.Fatal("settings accepted a reserve that leaves the ladder nothing to measure")
			}
			if err != nil && tc.valid {
				t.Errorf("settings rejected a usable reserve: %v", err)
			}
			if err != nil && !strings.Contains(err.Error(), "limits.compaction.reserve_tokens") {
				t.Errorf("error %q does not name limits.compaction.reserve_tokens", err)
			}
		})
	}
}

// The message floor is the one bound no pass can move: activeStart lays it down
// before the token budget widens anything, and it is enough to hold the tail over
// the half-cap however small the budget is. That is the wedge the cap exists to
// close, and it is closed here rather than at run time — refusing a floor is a
// check on a config, where dropping the floor or re-expressing a count of turns as
// a byte budget would both mean touching a window no tier may rewrite.
//
// What the floor costs is estimated from the package's own bound for one turn,
// because a count of turns a session has not taken cannot be measured. Only a
// window or a ceiling written down here bounds the cap, so the last two cases are
// the ones the check has to leave alone: a backend's own window is one this layer
// cannot see, and refusing a config over it would be worse than the floor. It is
// checked through the real loader, so the file and the environment are refused the
// same way, and the shipped floor is judged with everything else.
func TestValidateTailRefusesAMessageFloorOverTheHalfCap(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		valid bool
	}{
		{"a floor over what the ceiling leaves", "limits:\n  compaction:\n    window_tokens: 200000\n    keep_turns: 5\n  compact_at: 40000\n", false},
		{"a floor the ceiling alone refuses", "limits:\n  compaction:\n    keep_turns: 3\n  compact_at: 20000\n", false},
		{"a floor that fits the same ceiling", "limits:\n  compaction:\n    window_tokens: 200000\n    keep_turns: 4\n  compact_at: 40000\n", true},
		{"the shipped floor under the same cap", "limits:\n  compaction:\n    window_tokens: 200000\n    keep_turns: 1\n  compact_at: 40000\n", true},
		{"a floor of five the window can afford", "limits:\n  compaction:\n    window_tokens: 200000\n    keep_turns: 5\n", true},
		{"a floor with no window and no ceiling", "limits:\n  compaction:\n    keep_turns: 5\n", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			written(t, tc.body)
			_, err := settings(Config{})
			if err == nil && !tc.valid {
				t.Fatal("settings accepted a message floor over the half of a pass it may fill")
			}
			if err != nil && tc.valid {
				t.Errorf("settings rejected a usable message floor: %v", err)
			}
			if err != nil && !strings.Contains(err.Error(), "limits.compaction.keep_turns") {
				t.Errorf("error %q does not name limits.compaction.keep_turns", err)
			}
		})
	}
}
