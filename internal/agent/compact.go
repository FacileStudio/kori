package agent

import (
	"github.com/FacileStudio/nacelle"

	"github.com/FacileStudio/kori/internal/settings"
)

// Budget is the resolved compaction ceiling for one session: the token count
// the soft tier trips at, the ratio ladder the mid and hard tiers sit on, and
// the backend's total window (0 when the backend does not report one).
type Budget struct {
	Ceiling   int64
	TierRatio struct {
		Soft, Mid, Hard float64
	}
	Window int64
}

// ResolveBudget turns the compact_at setting and the compaction ratios into the
// ceiling a session actually runs on. A value someone set — file, environment
// or flag — wins outright, and 0 in it still turns compaction off. An unset
// value derives the ceiling from soft_ratio × the backend's context window, and
// falls back to settings.DefaultCompactAt when the backend reports no window at
// all. The ratio ladder is carried whether or not compact_at overrode the
// ceiling, so a session that pins compact_at still tiers its passes.
//
// A derived ceiling that comes out zero or less falls back to
// settings.DefaultCompactAt rather than staying at zero, because a ceiling of
// zero is how every gate spells "compaction off": letting a degenerate ratio
// produce one would turn the ladder off while reporting it as enabled. Settings
// rejects such a ratio at load, so this is the floor under a caller that built a
// Compaction itself.
func ResolveBudget(compactAt *int64, c settings.Compaction, backend nacelle.Backend) Budget {
	soft, mid, hard := c.Ratios()
	budget := Budget{Window: backend.Capabilities().ContextWindow}
	budget.TierRatio.Soft, budget.TierRatio.Mid, budget.TierRatio.Hard = soft, mid, hard
	switch {
	case compactAt != nil:
		budget.Ceiling = *compactAt
	case budget.Window > 0:
		budget.Ceiling = int64(soft * float64(budget.Window))
		if budget.Ceiling <= 0 {
			budget.Ceiling = settings.DefaultCompactAt
		}
	default:
		budget.Ceiling = settings.DefaultCompactAt
	}
	return budget
}
