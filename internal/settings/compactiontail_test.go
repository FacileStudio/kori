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
