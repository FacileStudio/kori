package agent

import (
	"testing"

	"github.com/FacileStudio/nacelle"

	"github.com/FacileStudio/kori/internal/settings"
)

// fixedWindow is the package's answeringStub with a context window, so the
// resolver's ratio branch can be exercised without a real backend.
type fixedWindow struct {
	answeringStub
	window int64
}

func (w fixedWindow) Capabilities() nacelle.Capabilities {
	return nacelle.Capabilities{ContextWindow: w.window}
}

// The ceiling has three sources and this pins the order: a value someone set
// wins, and 0 in it still turns compaction off; an unset value derives from the
// soft ratio and the window; a windowless backend falls back to the constant.
func TestResolveBudgetCeilingPrecedence(t *testing.T) {
	tests := []struct {
		name      string
		compactAt *int64
		window    int64
		ceiling   int64
	}{
		{"unset derives from the window", nil, 200_000, 130_000},
		{"an explicit value wins", new(int64(50_000)), 200_000, 50_000},
		{"an explicit 0 still disables", new(int64(0)), 200_000, 0},
		{"a windowless backend falls back", nil, 0, settings.DefaultCompactAt},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveBudget(tc.compactAt, settings.Compaction{}, &fixedWindow{window: tc.window})
			if got.Ceiling != tc.ceiling {
				t.Errorf("ceiling = %d, want %d", got.Ceiling, tc.ceiling)
			}
			if got.Window != tc.window {
				t.Errorf("window = %d, want %d", got.Window, tc.window)
			}
		})
	}
}

// The ladder is carried whether or not compact_at overrode the ceiling, so a
// session that pins the trigger still tiers its passes.
func TestResolveBudgetCarriesTheRatios(t *testing.T) {
	soft, mid, hard := 0.5, 0.7, 0.95
	configured := settings.Compaction{SoftRatio: &soft, MidRatio: &mid, HardRatio: &hard}
	got := ResolveBudget(new(int64(80_000)), configured, &fixedWindow{window: 100_000})
	if got.TierRatio.Soft != 0.5 || got.TierRatio.Mid != 0.7 || got.TierRatio.Hard != 0.95 {
		t.Errorf("ratios = %+v, want the configured ladder", got.TierRatio)
	}
	if got.Ceiling != 80_000 {
		t.Errorf("ceiling = %d, want the pinned 80000", got.Ceiling)
	}
}
