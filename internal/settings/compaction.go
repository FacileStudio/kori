package settings

import "os"

// Compaction is the ratio-based context-management surface: the tier ladder
// that decides when a session tombstones, prunes or folds history, and the
// opt-in judge that classifies history blocks before any of that happens.
// Every scalar is a pointer, so a layer that mentions one ratio leaves the rest
// of the policy alone instead of resetting it to zero.
type Compaction struct {
	SoftRatio      *float64 `yaml:"soft_ratio"`
	MidRatio       *float64 `yaml:"mid_ratio"`
	HardRatio      *float64 `yaml:"hard_ratio"`
	KeepTurns      *int     `yaml:"keep_turns"`
	AnchorMessages *int     `yaml:"anchor_messages"`
	Judge          Judge    `yaml:"judge"`
}

// Judge is the TypeSafe System One classifier that ranks history blocks
// keep / prune / ledger before the summarizer writes the ledger. It is off by
// default: enabling it sends conversation history to a third party, so it is
// an explicit opt-in and never a shipped default. The key prefers the
// TYPESAFE_API_KEY environment variable over the file, which is the one
// credential this config may carry.
type Judge struct {
	Enabled        *bool    `yaml:"enabled"`
	Model          string   `yaml:"model"`
	BaseURL        string   `yaml:"base_url"`
	APIKey         string   `yaml:"api_key"`
	PruneThreshold *float64 `yaml:"prune_threshold"`
	MaxBlocks      *int     `yaml:"max_blocks_per_call"`
}

// Merge overwrites every compaction setting over actually mentions, so a
// profile that sets one ratio does not reset the ladder around it. It is
// exported because a cron job's own limits.compaction has to reach a resolved
// run the same way the rest of this package's merge chain does.
func (c *Compaction) Merge(over Compaction) {
	if over.SoftRatio != nil {
		c.SoftRatio = over.SoftRatio
	}
	if over.MidRatio != nil {
		c.MidRatio = over.MidRatio
	}
	if over.HardRatio != nil {
		c.HardRatio = over.HardRatio
	}
	if over.KeepTurns != nil {
		c.KeepTurns = over.KeepTurns
	}
	if over.AnchorMessages != nil {
		c.AnchorMessages = over.AnchorMessages
	}
	c.Judge.merge(over.Judge)
}

// merge overwrites every judge field over actually mentions. An empty string
// never clears one a lower layer supplied, so an environment key survives a
// file that only turns the judge on.
func (j *Judge) merge(over Judge) {
	if over.Enabled != nil {
		j.Enabled = over.Enabled
	}
	if over.Model != "" {
		j.Model = over.Model
	}
	if over.BaseURL != "" {
		j.BaseURL = over.BaseURL
	}
	if over.APIKey != "" {
		j.APIKey = over.APIKey
	}
	if over.PruneThreshold != nil {
		j.PruneThreshold = over.PruneThreshold
	}
	if over.MaxBlocks != nil {
		j.MaxBlocks = over.MaxBlocks
	}
}

// Ratios is the tier ladder with any ratio a layer left out filled from the
// shipped defaults, so callers never handle the pointers themselves.
func (c Compaction) Ratios() (soft, mid, hard float64) {
	soft, mid, hard = DefaultSoftRatio, DefaultMidRatio, DefaultHardRatio
	if c.SoftRatio != nil {
		soft = *c.SoftRatio
	}
	if c.MidRatio != nil {
		mid = *c.MidRatio
	}
	if c.HardRatio != nil {
		hard = *c.HardRatio
	}
	return soft, mid, hard
}

// defaultCompaction is the tier ladder and judge a session with no opinion of
// its own runs on. The judge is off: it is the one setting that sends history
// off the machine, so a machine nobody opted in on never makes that call.
func defaultCompaction() Compaction {
	soft, mid, hard := DefaultSoftRatio, DefaultMidRatio, DefaultHardRatio
	keepTurns, anchorMessages, maxBlocks := 3, 1, 64
	pruneThreshold, judgeEnabled := 0.85, false
	return Compaction{
		SoftRatio:      &soft,
		MidRatio:       &mid,
		HardRatio:      &hard,
		KeepTurns:      &keepTurns,
		AnchorMessages: &anchorMessages,
		Judge: Judge{
			Enabled:        &judgeEnabled,
			Model:          "jev-latest",
			BaseURL:        "https://api.typesafe.ai",
			PruneThreshold: &pruneThreshold,
			MaxBlocks:      &maxBlocks,
		},
	}
}

// compactionEnv is the compaction layer the environment supplies. Every scalar
// is optional and a value it cannot read is treated as unmentioned, exactly the
// way the rest of the environment layer behaves.
func compactionEnv() Compaction {
	return Compaction{
		SoftRatio:      envFloat(EnvPrefix + "COMPACTION_SOFT_RATIO"),
		MidRatio:       envFloat(EnvPrefix + "COMPACTION_MID_RATIO"),
		HardRatio:      envFloat(EnvPrefix + "COMPACTION_HARD_RATIO"),
		KeepTurns:      envInt(EnvPrefix + "COMPACTION_KEEP_TURNS"),
		AnchorMessages: envInt(EnvPrefix + "COMPACTION_ANCHOR_MESSAGES"),
		Judge: Judge{
			Enabled:        envBool(EnvPrefix + "COMPACTION_JUDGE"),
			Model:          envGet("COMPACTION_JUDGE_MODEL"),
			BaseURL:        envGet("COMPACTION_JUDGE_BASE_URL"),
			APIKey:         judgeKeyEnv(),
			PruneThreshold: envFloat(EnvPrefix + "COMPACTION_PRUNE_THRESHOLD"),
			MaxBlocks:      envInt(EnvPrefix + "COMPACTION_MAX_BLOCKS"),
		},
	}
}

// judgeKeyEnv reads the judge's key, preferring TYPESAFE_API_KEY — the vendor's
// own name, so a key already exported for TypeSafe needs no second copy — over
// the namespaced setting when both are set.
func judgeKeyEnv() string {
	if key := os.Getenv("TYPESAFE_API_KEY"); key != "" {
		return key
	}
	return envGet("COMPACTION_JUDGE_API_KEY")
}
