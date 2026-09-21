package settings

import "testing"

// The ladder and the judge's own defaults are the shipped policy: the judge is
// off because enabling it is what sends history off the machine.
func TestCompactionDefaults(t *testing.T) {
	c := Defaults("").Compaction
	soft, mid, hard := c.Ratios()
	if soft != DefaultSoftRatio || mid != DefaultMidRatio || hard != DefaultHardRatio {
		t.Errorf("ratios = %v/%v/%v, want %v/%v/%v", soft, mid, hard, DefaultSoftRatio, DefaultMidRatio, DefaultHardRatio)
	}
	if c.KeepTurns == nil || *c.KeepTurns != 3 {
		t.Errorf("keep_turns = %v, want 3", c.KeepTurns)
	}
	if c.AnchorMessages == nil || *c.AnchorMessages != 1 {
		t.Errorf("anchor_messages = %v, want 1", c.AnchorMessages)
	}
	if DerefBool(c.Judge.Enabled) {
		t.Error("judge enabled = true, want the judge off by default")
	}
	if c.Judge.Model != "jev-latest" || c.Judge.BaseURL != "https://api.typesafe.ai" {
		t.Errorf("judge endpoint = %q %q, want the TypeSafe defaults", c.Judge.Model, c.Judge.BaseURL)
	}
	if c.Judge.PruneThreshold == nil || *c.Judge.PruneThreshold != 0.85 {
		t.Errorf("prune_threshold = %v, want 0.85", c.Judge.PruneThreshold)
	}
	if c.Judge.MaxBlocks == nil || *c.Judge.MaxBlocks != 64 {
		t.Errorf("max_blocks_per_call = %v, want 64", c.Judge.MaxBlocks)
	}
}

// A file that mentions one ratio leaves the rest of the ladder standing, the
// way every other pointer setting in this package behaves — and its silence
// about compact_at must not resurrect a non-zero default.
func TestCompactionFileLeavesUnmentionedRatiosAlone(t *testing.T) {
	written(t, "limits:\n  compaction:\n    soft_ratio: 0.5\n    judge:\n      enabled: true\n")
	c, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	soft, mid, hard := c.Compaction.Ratios()
	if soft != 0.5 || mid != DefaultMidRatio || hard != DefaultHardRatio {
		t.Errorf("ratios = %v/%v/%v, want the file's 0.5 then the defaults", soft, mid, hard)
	}
	if !DerefBool(c.Compaction.Judge.Enabled) {
		t.Error("judge enabled = false, want the file's true")
	}
	if c.CompactAt != nil {
		t.Errorf("compact_at = %d, want the file's silence to leave it unset", *c.CompactAt)
	}
}

// The judge's key can live in the file like any other setting: an environment
// that mentions neither vendor variable must not clear it, and the vendor's own
// variable must still win over it when it is set.
func TestJudgeKeyComesFromTheFileUnlessTheEnvironmentOverrides(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "")
	t.Setenv(EnvPrefix+"COMPACTION_JUDGE_API_KEY", "")
	written(t, "limits:\n  compaction:\n    judge:\n      enabled: true\n      api_key: from-file\n")

	c, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if c.Compaction.Judge.APIKey != "from-file" {
		t.Errorf("judge key = %q, want the file's own", c.Compaction.Judge.APIKey)
	}

	t.Setenv("TYPESAFE_API_KEY", "from-env")
	c, err = settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if c.Compaction.Judge.APIKey != "from-env" {
		t.Errorf("judge key = %q, want the environment to win", c.Compaction.Judge.APIKey)
	}
}

// TYPESAFE_API_KEY is the vendor's own name and wins over the namespaced
// setting, so a key already exported for TypeSafe needs no second copy.
func TestJudgeKeyPrefersTheTypesafeVariable(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "from-typesafe")
	t.Setenv(EnvPrefix+"COMPACTION_JUDGE_API_KEY", "from-kori")
	if key := FromEnv().Compaction.Judge.APIKey; key != "from-typesafe" {
		t.Errorf("judge key = %q, want TYPESAFE_API_KEY to win", key)
	}
	t.Setenv("TYPESAFE_API_KEY", "")
	if key := FromEnv().Compaction.Judge.APIKey; key != "from-kori" {
		t.Errorf("judge key = %q, want the namespaced variable when TypeSafe's is empty", key)
	}
}
