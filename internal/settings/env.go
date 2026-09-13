// Package settings resolves the linear precedence chain that turns flags, environment
// variables, a YAML file, and built-in defaults into the one Config the session acts on.
package settings

import (
	"os"
	"strconv"
	"strings"
)

// EnvPrefix is what every setting's environment variable starts with.
const EnvPrefix = "NACELLE_"

// providerEnv reads the active provider's four fields. Backend and Model reuse
// the NACELLE_BACKEND/NACELLE_MODEL names every other layer uses; the endpoint
// and key are the ones this layer adds, so they carry the PROVIDER_ prefix.
func providerEnv() Provider {
	return Provider{
		Backend: os.Getenv(EnvPrefix + "BACKEND"),
		Model:   os.Getenv(EnvPrefix + "MODEL"),
		BaseURL: os.Getenv(EnvPrefix + "PROVIDER_BASE_URL"),
		APIKey:  os.Getenv(EnvPrefix + "PROVIDER_API_KEY"),
	}
}

// FromEnv is the settings layer the environment supplies.
func FromEnv() Config {
	return Config{
		Provider: providerEnv(),
		Session:  Session{Root: os.Getenv(EnvPrefix + "ROOT"), System: os.Getenv(EnvPrefix + "SYSTEM_PROMPT")},
		Limits: Limits{
			MaxIterations: envInt(EnvPrefix + "MAX_ITERATIONS"), CompactAt: envInt64(EnvPrefix + "COMPACT_AT"),
			GrindCost: envFloat(EnvPrefix + "GRIND_MIN_COST"), GrindTokens: envInt64(EnvPrefix + "GRIND_MIN_TOKENS"),
			GrindContinuations: envInt(EnvPrefix + "GRIND_CONTINUATIONS"),
		},
		Sources: Sources{SkillDirs: envList(EnvPrefix + "SKILL_DIRS")},
		UI:      UI{Mode: envString(EnvPrefix + "MODE"), TransparentBlocks: envBool(EnvPrefix + "TRANSPARENT_BLOCKS"), Diffs: envBool(EnvPrefix + "DIFFS")},
		Toggles: Toggles{
			Bash:           envBool(EnvPrefix + "BASH"),
			ParallelAgents: envBool(EnvPrefix + "PARALLEL_AGENTS"),
			Fetch:          envBool(EnvPrefix + "FETCH"),
			Tasks:          envBool(EnvPrefix + "TASKS"),
			Diagnostics:    envBool(EnvPrefix + "DIAGNOSTICS"),
		},
		Security: Security{
			ApproveTools:  envBool(EnvPrefix + "APPROVE_TOOLS"),
			PathIsolation: envBool(EnvPrefix + "PATH_ISOLATION"),
			DenyElevation: envBool(EnvPrefix + "DENY_ELEVATION"),
			EnvIsolation:  envBool(EnvPrefix + "ENV_ISOLATION"),
		},
		Reasoning: Reasoning{
			Effort:   os.Getenv(EnvPrefix + "EFFORT"),
			Thinking: envBool(EnvPrefix + "THINKING"),
			Budget:   envInt64(EnvPrefix + "REASONING_BUDGET"),
		},
		Discovery: Discovery{
			ProjectContext: envBool(EnvPrefix + "PROJECT_CONTEXT"),
			Skills:         envBool(EnvPrefix + "SKILLS"),
			TrustSkills:    envBool(EnvPrefix + "TRUST_SKILLS"),
		},
	}
}

// envString reads a string setting, returning nil when the variable is unset.
func envString(name string) *string {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return nil
	}
	return &raw
}

// envBool reads a toggle, returning nil when the variable is unset or is not
// something strconv recognises.
func envBool(name string) *bool {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil
	}
	return &value
}

// envInt reads a count, with the same treatment of an unreadable value.
func envInt(name string) *int {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	return &value
}

// envInt64 is envInt in the width a token count is measured in.
func envInt64(name string) *int64 {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &value
}

// envFloat is envInt in the width a dollar amount is measured in.
func envFloat(name string) *float64 {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	return &value
}

// envList reads a colon-separated list, returning nil when unset or empty.
func envList(name string) []string {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return nil
	}
	return strings.Split(raw, ":")
}
