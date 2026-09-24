package settings

import (
	"strings"
	"testing"
)

// This file pins the compaction keys that left the surface, kept apart from the
// ladder's own tests because a migration is a different question from a
// validation: one is about a config written against an older release, the other
// about a config written against this one.

// Two ratio keys left in one release: hard_ratio went with the third rung, and
// mid_ratio was renamed to smart_ratio. Both are refused by KnownFields(true)
// rather than silently ignored — the same arrangement session.resume has —
// because a setting that quietly does nothing is worse than one that says it is
// gone.
//
// The rename is the half that could be missed. A config still naming mid_ratio
// would otherwise fall back to the default rather than to the value its owner
// wrote, which is a silent change in how often the session compacts.
func TestCompactionRefusesTheRemovedRatioKeys(t *testing.T) {
	for _, body := range []string{
		"limits:\n  compaction:\n    mid_ratio: 0.75\n",
		"limits:\n  compaction:\n    hard_ratio: 0.9\n",
	} {
		written(t, body)
		_, err := settings(Config{})
		if err == nil {
			t.Fatalf("settings accepted %q, want a refusal: the key is gone", body)
		}
		if !strings.Contains(err.Error(), "ratio not found") {
			t.Errorf("error = %q, want it to name the unknown ratio key", err)
		}
	}
}

// The renamed key is the one that has to work, so the refusal above is a
// judgement and not a refusal to have any ratio at all: a config naming
// smart_ratio loads, and the value it wrote is the value the ladder gets.
func TestCompactionCarriesTheRenamedSmartRatio(t *testing.T) {
	written(t, "limits:\n  compaction:\n    smart_ratio: 0.75\n")

	c, err := settings(Config{})
	if err != nil {
		t.Fatalf("the renamed key was refused: %v", err)
	}
	if _, smart := c.Compaction.Ratios(); smart != 0.75 {
		t.Errorf("smart_ratio = %v, want the file's 0.75", smart)
	}
}

// The environment layer is lenient by design — a value it cannot read is treated
// as unmentioned — so a renamed variable would be silently ignored and the
// session would quietly run on the default instead of the ratio its owner tuned.
// The file layer refuses the old key by name; this is what gives the environment
// the same signal.
func TestCompactionRefusesTheRemovedRatioVariables(t *testing.T) {
	for _, name := range []string{"COMPACTION_MID_RATIO", "COMPACTION_HARD_RATIO"} {
		t.Run(name, func(t *testing.T) {
			written(t, "")
			clearEnv(t, EnvPrefix+name, legacyEnvPrefix+name)
			t.Setenv(EnvPrefix+name, "0.75")

			_, err := settings(Config{})
			if err == nil {
				t.Fatalf("%s was accepted, want a refusal naming the variable", name)
			}
			if !strings.Contains(err.Error(), EnvPrefix+name) {
				t.Errorf("error = %q, want it to name %s", err, EnvPrefix+name)
			}
		})
	}
}

// An empty variable reads as unmentioned, the way every other value in that layer
// does, so a shell that exports the name with nothing in it does not block a boot.
func TestCompactionIgnoresAnEmptyRemovedRatioVariable(t *testing.T) {
	written(t, "")
	clearEnv(t, EnvPrefix+"COMPACTION_MID_RATIO", legacyEnvPrefix+"COMPACTION_MID_RATIO")
	t.Setenv(EnvPrefix+"COMPACTION_MID_RATIO", "")

	if _, err := settings(Config{}); err != nil {
		t.Errorf("an empty %sCOMPACTION_MID_RATIO refused a boot: %v", EnvPrefix, err)
	}
}
