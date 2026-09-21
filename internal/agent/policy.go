package agent

import (
	"github.com/FacileStudio/kori/internal/compaction"
	"github.com/FacileStudio/kori/internal/settings"
	"github.com/FacileStudio/kori/internal/tui"
)

// Policy folds the resolved budget and the two end sizes into the one compaction
// policy an interactive session reads. It keeps the ratios even when compact_at
// overrode the ceiling, so a session that pins its trigger still tiers its
// passes against the window.
func Policy(b Budget, keepTurns, anchorMessages int) compaction.Policy {
	return compaction.Policy{
		Ratios:         compaction.Ratios{Soft: b.TierRatio.Soft, Mid: b.TierRatio.Mid, Hard: b.TierRatio.Hard},
		Window:         b.Window,
		Ceiling:        b.Ceiling,
		KeepTurns:      keepTurns,
		AnchorMessages: anchorMessages,
	}
}

// CompactionConfig folds the resolved budget, the two end sizes and the opt-in
// judge into the one struct a session reads, so runflags stays one line per
// field.
func CompactionConfig(b Budget, c settings.Compaction) tui.CompactionConfig {
	return tui.CompactionConfig{
		Policy: Policy(b, settings.DerefInt(c.KeepTurns), settings.DerefInt(c.AnchorMessages)),
		Judge:  Judge(c),
	}
}

// Judge builds the opt-in System One classifier from the compaction settings, or
// nil while the judge is off — the default, because enabling it sends
// conversation history to a third party. The key prefers TYPESAFE_API_KEY, which
// the settings layer has already resolved into the config.
func Judge(c settings.Compaction) compaction.Judge {
	judge := c.Judge
	if !settings.DerefBool(judge.Enabled) {
		return nil
	}
	return compaction.NewJevJudge(compaction.JudgeConfig{
		Enabled:        true,
		Model:          judge.Model,
		BaseURL:        judge.BaseURL,
		APIKey:         judge.APIKey,
		PruneThreshold: settings.DerefFloat(judge.PruneThreshold),
		MaxBlocks:      settings.DerefInt(judge.MaxBlocks),
	})
}
