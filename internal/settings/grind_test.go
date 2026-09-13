package settings

// The grind budget's config surface: off by default, readable from the file
// and from the environment, and merged with the same precedence everything
// under limits: follows.

import "testing"

func TestGrindBudgetDefaultsOff(t *testing.T) {
	fallback := Defaults("")
	if *fallback.GrindCost != 0 {
		t.Errorf("default grind_min_cost = %g, want 0", *fallback.GrindCost)
	}
	if *fallback.GrindTokens != 0 {
		t.Errorf("default grind_min_tokens = %d, want 0", *fallback.GrindTokens)
	}
	if *fallback.GrindContinuations != 2 {
		t.Errorf("default grind_continuations = %d, want 2", *fallback.GrindContinuations)
	}
}

func TestGrindBudgetFromTheFile(t *testing.T) {
	written(t, "limits:\n  grind_min_cost: 0.25\n  grind_min_tokens: 2000\n  grind_continuations: 3\n")
	config, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if *config.GrindCost != 0.25 {
		t.Errorf("grind_min_cost = %g, want 0.25", *config.GrindCost)
	}
	if *config.GrindTokens != 2000 {
		t.Errorf("grind_min_tokens = %d, want 2000", *config.GrindTokens)
	}
	if *config.GrindContinuations != 3 {
		t.Errorf("grind_continuations = %d, want 3", *config.GrindContinuations)
	}
}

func TestGrindBudgetFromTheEnvironmentBeatsTheFile(t *testing.T) {
	written(t, "limits:\n  grind_min_cost: 0.25\n")
	t.Setenv(EnvPrefix+"GRIND_MIN_COST", "1.5")
	t.Setenv(EnvPrefix+"GRIND_MIN_TOKENS", "4000")
	t.Setenv(EnvPrefix+"GRIND_CONTINUATIONS", "1")

	config, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if *config.GrindCost != 1.5 {
		t.Errorf("grind_min_cost = %g, want the environment's 1.5", *config.GrindCost)
	}
	if *config.GrindTokens != 4000 {
		t.Errorf("grind_min_tokens = %d, want the environment's 4000", *config.GrindTokens)
	}
	if *config.GrindContinuations != 1 {
		t.Errorf("grind_continuations = %d, want the environment's 1", *config.GrindContinuations)
	}
}
