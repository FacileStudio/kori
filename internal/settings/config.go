// Package settings resolves the precedence chain that turns flags, environment
// variables, a YAML file, and built-in defaults into the one Config the session acts on.
package settings

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/FacileStudio/nacelle/mcp/client"
	"go.yaml.in/yaml/v4"
)

// ConfigFile is where settings are read from when the flags leave them out.
const ConfigFile = ".kori.yml"

// Limits is the threshold settings that cap the run. Embedded in Config so
// every field still reads as c.MaxIterations.
type Limits struct {
	MaxIterations *int   `yaml:"max_iterations"`
	CompactAt     *int64 `yaml:"compact_at"`
	// GrindBudget is the per-run minimum spend the model must reach before a
	// stop counts as finished: a run that ends below it on a stop the model
	// chose is continued with a notice naming what is left. GrindCost is
	// dollars, GrindTokens counts output tokens, both default to zero which
	// turns the budget off, and when both are set a run owes both.
	GrindCost          *float64 `yaml:"grind_min_cost"`
	GrindTokens        *int64   `yaml:"grind_min_tokens"`
	GrindContinuations *int     `yaml:"grind_continuations"`
}

// Provider is the backend in use plus the endpoint and key that reach it.
// The yaml keys live under the group names (provider:, limits: and the rest);
// the KORI_ names are new; NACELLE_ names still resolve for existing environments.
type Provider struct {
	Backend string `yaml:"backend"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
}

// Session is the launch settings that are not display choices — where the
// session starts, what its base prompt says, whether it resumes. They live
// under the session: group in the file, except resume: which is flag-only.
type Session struct {
	Root     string  `yaml:"root"`
	System   string  `yaml:"system_prompt"`
	Continue *bool   `yaml:"continue"`
	Resume   *string `yaml:"-"`
}

// NoConfig skips the file entirely; it comes from -no-config, never the
// file it is about to ignore.

// Config is one layer of settings. Every field is a pointer or an empty-able
// string so a layer can say nothing about a setting rather than saying zero:
// "false" and "not mentioned" are different answers and a bool cannot tell
// them apart. Its only credential is a custom endpoint's own api_key.
type Config struct {
	NoConfig *bool `yaml:"-"`

	// GatesFile is the --gates-file path; flag-only, like resume.
	GatesFile string `yaml:"-"`

	Provider   `yaml:"provider"`
	Session    `yaml:"session"`
	Limits     `yaml:"limits"`
	Toggles    `yaml:"tools"`
	Reasoning  `yaml:"reasoning"`
	Security   `yaml:"security"`
	Discovery  `yaml:"discovery"`
	UI         `yaml:"ui"`
	Sources    `yaml:"sources"`
	Automation `yaml:",inline"`
}

// Toggles is the tool-mount settings: whether the model gets each optional
// tool. Every toggle is a pointer so "not in this file" can be told from "false".
type Toggles struct {
	Bash           *bool `yaml:"run_command"`
	ParallelAgents *bool `yaml:"parallel_agents"`
	Fetch          *bool `yaml:"web_fetch"`
	Tasks          *bool `yaml:"tasks"`
	Diagnostics    *bool `yaml:"diagnostics"`
	SearchContent  *bool `yaml:"search_content"`
	FindFiles      *bool `yaml:"find_files"`
}

// Security holds the settings that decide how much a tool call may do
// before something stops it: ask before every call runs, confine to the
// root, refuse a command that tries to elevate its privileges, and whether
// MCP servers and run_command start with this process's environment or a
// minimal one.
type Security struct {
	ApproveTools  *bool `yaml:"approve_tools"`
	PathIsolation *bool `yaml:"path_isolation"`
	// DenyElevation refuses run_command calls that try to elevate
	// privileges (kori, su, doas, pkexec): a policy guard against accidents
	// and injected instructions, not a security boundary; the OS decides
	// who may elevate.
	DenyElevation *bool `yaml:"deny_elevation"`
	// EnvIsolation starts MCP servers and run_command children with PATH,
	// HOME and configured env entries instead of the inherited environment.
	EnvIsolation *bool `yaml:"env_isolation"`
}

// UI holds display settings for the interactive client. Diffs shows a
// git-style diff when the model edits a file; GroupTools collapses
// consecutive read-only tool calls into one line; ShowThinking expands
// thinking traces by default (ctrl+t toggles).
//
// PromptPlaceholder is the ghost text while the prompt is empty, StartMessage
// prints on launch above the banner, Mode is "inline" or "tui" rendering.
type UI struct {
	Mode              *string `yaml:"rendering_mode"`
	GroupTools        *bool   `yaml:"group_tools"`
	ShowThinking      *bool   `yaml:"show_thinking"`
	Diffs             *bool   `yaml:"diffs"`
	PromptPlaceholder *string `yaml:"prompt_placeholder"`
	StartMessage      *string `yaml:"start_message"`
	TransparentBlocks *bool   `yaml:"transparent_blocks"`
	JSON              *bool   `yaml:"cron_list_json"`
}

// Reasoning holds the three settings that decide how hard the model thinks.
// Effort and Budget spell one idea twice — the backends disagree on which
// they accept, so each sends the one its own API understands.
type Reasoning struct {
	Effort   string `yaml:"effort"`
	Thinking *bool  `yaml:"thinking"`
	Budget   *int64 `yaml:"budget"`
}

// Discovery holds the three settings that decide what this session folds into
// its system prompt from outside the conversation itself.
type Discovery struct {
	ProjectContext *bool `yaml:"project_context"`
	Skills         *bool `yaml:"skills"`
	TrustSkills    *bool `yaml:"trust_skills"`
	TrustHooks     *bool `yaml:"trust_hooks"`
}

// Sources names what to read. SkillDirs replaces when set. MCP holds servers
// written inline (the way most people configure one); MCPFiles names .mcp.json files to load as well
// (the -mcp flag), merged by server name with them win, so ~/.claude/.mcp.json keeps working.
type Sources struct {
	SkillDirs []string                    `yaml:"skill_dirs"`
	MCP       map[string]client.ServerDef `yaml:"mcp"`
	MCPFiles  []string                    `yaml:"-"`
}

// HookSpec is one entry under a config's `hooks:` key.
type HookSpec struct {
	On      string   `yaml:"on"`
	Match   []string `yaml:"match"`
	Run     string   `yaml:"run"`
	Timeout string   `yaml:"timeout"`
	Async   bool     `yaml:"async"`
}

// GateSpec is one entry under a config's `gates:` key.
type GateSpec struct {
	Name        string   `yaml:"name"`
	Command     []string `yaml:"command"`
	TimeoutSecs int      `yaml:"timeout_secs"`
	Format      bool     `yaml:"format"`
}

// Automation groups the config's chained machinery. It is inline in the
// YAML, so the keys stay top-level: hooks, gates. Scheduled jobs are not
// here anymore — LoadJobs reads them from ~/.kori/jobs/ instead.
type Automation struct {
	Hooks []HookSpec `yaml:"hooks"`
	Gates []GateSpec `yaml:"gates"`
}

// DerefBool reads a pointer out of a toggle. Every toggle is filled in by
// defaults, so the pointer is never nil by the time it reaches a caller.
func DerefBool(b *bool) bool {
	return b != nil && *b
}

// ConfigPath is where the config file lives (HOME).
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ConfigFile)
}

// Load reads the config file. A missing file is not an error — most people
// never write one — but an unreadable or malformed one is.
func Load(path string) (Config, error) {
	if path == "" {
		return Config{}, nil
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("reading %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	var settings Config
	if err := decoder.Decode(&settings); err != nil && !errors.Is(err, io.EOF) {
		return Config{}, &ParseError{Path: path, Err: repointCronKey(err)}
	}
	return settings, nil
}

// Settings resolves every layer in one place: flag beats environment beats
// file beats default. The scaffold runs before the file is read.
func Settings(system string, flags Config) (Config, error) {
	migrateLegacyHome()
	migrateLegacyConfig()
	if flags.NoConfig != nil && *flags.NoConfig {
		resolved := Defaults(system)
		resolved.merge(FromEnv())
		resolved.merge(flags)
		return resolveGates(resolved, flags.GatesFile)
	}
	if created, err := Scaffold(ConfigPath()); err != nil {
		return Config{}, err
	} else if created {
		fmt.Fprintln(os.Stderr, "wrote ~/.kori.yml with the default settings — edit it, or delete it to regenerate")
	}
	file, err := Load(ConfigPath())
	if err != nil {
		return Config{}, err
	}

	resolved := Defaults(system)
	resolved.merge(file)
	resolved.merge(FromEnv())
	resolved.merge(flags)
	return resolveGates(resolved, flags.GatesFile)
}
