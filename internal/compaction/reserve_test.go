package compaction

// The reserve is the runway held back from the backend's window for the turn's
// own answer, and the ladder names a fraction of what is left rather than of the
// whole window. Its tests live here, apart from the ladder's own, because the
// reserve is the thing that decides what the ratios are ratios *of*.

import "testing"

// A reserve moves every rung down together, because the ladder names a fraction
// of the window a turn can actually fill: 0.65/0.80 of 160000 is 104000/128000,
// and what that buys is that the top rung still leaves the 40000 reserve plus a
// fifth of the usable window for the answer instead of a fifth of the raw one.
func TestTierMeasuresTheUsableWindow(t *testing.T) {
	p := Policy{Ratios: Ratios{Soft: 0.65, Smart: 0.80}, Window: 200_000, Reserve: 40_000}
	tests := []struct {
		name string
		size int64
		want Tier
	}{
		{"under the usable soft", 103_999, Below},
		{"at the usable soft", 104_000, Soft},
		{"just under the usable smart", 127_999, Soft},
		{"at the usable smart", 128_000, Smart},
		{"over the usable smart", 199_999, Smart},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := p.Tier(tc.size); got != tc.want {
				t.Errorf("Tier(%d) = %v, want %v", tc.size, got, tc.want)
			}
		})
	}
}

// The trigger follows the reserve too — a session is asked to compact at the
// soft ratio of what it can fill, not of the whole window — and a reserve that
// would swallow the window leaves nothing to measure rather than a zero
// threshold every automatic guard would read as "compaction off".
func TestReserveMovesTheTrigger(t *testing.T) {
	held := Policy{Ratios: Ratios{Soft: 0.65}, Window: 200_000, Reserve: 40_000}
	if got := held.Trigger(); got != 104_000 {
		t.Errorf("Trigger with a 40000 reserve = %d, want 104000", got)
	}

	degenerate := Policy{Ratios: Ratios{Soft: 0.65}, Window: 200_000, Reserve: 200_000}
	if got := degenerate.Usable(); got != 0 {
		t.Errorf("Usable with a degenerate reserve = %d, want 0", got)
	}
	if got := degenerate.Trigger(); got != 0 {
		t.Errorf("Trigger with a degenerate reserve = %d, want no threshold", got)
	}
}
