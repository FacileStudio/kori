package settings

import "strings"

// DefaultCompactAt is the transcript size, in tokens, at which a session
// with no opinion of its own compacts.
const DefaultCompactAt int64 = 75_000

// Defaults is the bottom layer, and the only one that answers everything.
func Defaults(system string) Config {
	bash, thinking, projectContext, skills, trustSkills, approveTools, trustHooks, diffs, tasks, strict :=
		true, true, true, true, false, false, false, true, true, false
	envIsolation, denyElevation, parallelAgents, diagnostics := false, true, true, true
	iterations, budget, compactAt, grindCost, grindTokens, grindContinuations := 5, int64(0), int64(75000), 0.0, int64(0), 2
	maxConcurrency := 16
	fetch, searchContent, findFiles, groupTools, showThinking, showHooks, showHookOutput := true, true, true, true, true, true, true
	cont, resume, mode, transparent, json := false, "", "tui", true, false
	promptPlaceholder := "Ask something. Esc stops a run, ctrl+c stops or quits, ctrl+\\ forces it."
	startMessage := strings.Join([]string{
		"▄▄ ▄▄  ▄▄▄  ▄▄▄▄  ▄▄ ",
		"██▄█▀ ██▀██ ██▄█▄ ██ ",
		"██ ██ ▀███▀ ██ ██ ██ ",
	}, "\n")
	autoSync, autoSnapshot := true, false
	return Config{
		Provider:  Provider{Backend: "anthropic"},
		Session:   Session{Root: ".", System: system, Continue: &cont, Resume: &resume},
		Toggles:   Toggles{Bash: &bash, ParallelAgents: &parallelAgents, Fetch: &fetch, Tasks: &tasks, Diagnostics: &diagnostics, SearchContent: &searchContent, FindFiles: &findFiles},
		Security:  Security{ApproveTools: &approveTools, PathIsolation: &strict, DenyElevation: &denyElevation, EnvIsolation: &envIsolation},
		Limits:    Limits{MaxIterations: &iterations, CompactAt: &compactAt, GrindCost: &grindCost, GrindTokens: &grindTokens, GrindContinuations: &grindContinuations, MaxConcurrency: &maxConcurrency},
		Reasoning: Reasoning{Thinking: &thinking, Budget: &budget},
		Discovery: Discovery{
			ProjectContext: &projectContext,
			Skills:         &skills,
			TrustSkills:    &trustSkills,
			TrustHooks:     &trustHooks,
		},
		UI:      UI{Mode: &mode, GroupTools: &groupTools, ShowThinking: &showThinking, Diffs: &diffs, PromptPlaceholder: &promptPlaceholder, StartMessage: &startMessage, TransparentBlocks: &transparent, JSON: &json, ShowHooks: &showHooks, ShowHookOutput: &showHookOutput},
		Editor:  Editor{Editor: "", PromptEditKey: "ctrl+g"},
		Sandbox: Sandbox{VMName: "", Port: 2226, SSHKeyPath: "~/.ssh/id_ed25519", Root: "/workspace", AutoSync: &autoSync, AutoSnapshot: &autoSnapshot},
	}
}
