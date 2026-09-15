package agent

import (
	"fmt"
	"os"
	"time"

	"github.com/FacileStudio/nacelle"
	"github.com/FacileStudio/nacelle/tools"

	"github.com/FacileStudio/kori/internal/approval"
	"github.com/FacileStudio/kori/internal/settings"
	"github.com/FacileStudio/kori/internal/skills"
	"github.com/FacileStudio/kori/internal/tui"
)

// Run is the main entry point for the agent.
func Run(v string) error {
	if handled, err := checkVersionFlag(v); handled {
		return err
	}
	if handled, err := checkCronFlag(); handled {
		return err
	}
	if handled, err := checkBenchFlag(); handled {
		return err
	}
	if handled, err := checkPrintFlag(); handled {
		return err
	}
	sess, cleanup, err := bootOrAsk(v)
	if err != nil {
		return err
	}
	defer cleanup()
	return tui.Launch(*sess)
}

type preparedTools struct {
	config settings.Config
	set    *tools.Set
	mcp    connected
	local  []nacelle.Tool
}

func setupAgentTools(noConfig bool) (preparedTools, error) {
	flags := settings.FromFlags(settings.Defaults(""))
	if noConfig {
		flags.NoConfig = &noConfig
	}
	config, err := settings.Settings(DefaultSystemPrompt(), flags)
	if err != nil {
		return preparedTools{}, err
	}
	set, local, err := localTools(config)
	if err != nil {
		return preparedTools{}, err
	}
	mcp, local, err := mcpTools(config, local)
	if err != nil {
		return preparedTools{}, closeOnErr(err, set)
	}
	return preparedTools{config: config, set: set, mcp: mcp, local: local}, nil
}

func setupAgentSession(p preparedTools, v string) (*tui.UISession, error) {
	found := augmentSystem(&p.config, p.mcp)
	approvalGate, approve := approval.Build(*p.config.ApproveTools)

	hooks, hookNotice, err := settings.SessionHooks(p.config)
	if err != nil {
		return nil, err
	}

	get, err := build(p.config, p.local, approve, hooks)
	if err != nil {
		return nil, err
	}

	return &tui.UISession{
		Agent:             get.agent,
		Banner:            banner(get.backend, p.config, found, p.mcp, v),
		Skills:            found.skills,
		HookNotice:        hookNotice,
		Gate:              approvalGate,
		DelegateConfig:    get.config,
		Mode:              *p.config.Mode,
		TransparentBlocks: *p.config.TransparentBlocks,
		SessionConfig:     sessionConfig(p, found, get.backend),
	}, nil
}

// sessionConfig folds the runtime settings an interactive session reads into
// one struct — the banner and the launch notes take theirs from the same
// snapshot, so the two cannot disagree about what was loaded.
func sessionConfig(p preparedTools, found loaded, backend nacelle.Backend) tui.SessionConfig {
	return tui.SessionConfig{
		Root:         p.config.Root,
		Model:        p.config.Model,
		Backend:      p.config.Backend,
		Diffs:        *p.config.Diffs,
		GroupTools:   p.config.GroupTools,
		ShowThinking: *p.config.ShowThinking,
		CompactAt:    resolveCompactAt(*p.config.CompactAt, backend),
		Grind: tui.GrindConfig{
			Cost:          *p.config.GrindCost,
			Tokens:        *p.config.GrindTokens,
			Continuations: *p.config.GrindContinuations,
		},
		AutoResume:        *p.config.Continue,
		Resume:            *p.config.Resume,
		PromptPlaceholder: *p.config.PromptPlaceholder,
		StartMessage:      *p.config.StartMessage,
		Editor: tui.EditorConfig{
			PromptEditKey: p.config.PromptEditKey,
			Editor:        p.config.Editor.Editor,
		},
		Startup: tui.LaunchContext{
			ContextPaths:  found.contextPaths,
			ContextTokens: tokenEstimate(found.contextChars),
			SystemTokens:  tokenEstimate(found.systemChars),
			Diagnostics:   settings.DerefBool(p.config.Diagnostics),
		},
		Hooks: tui.HookUIConfig{
			Show:       settings.DerefBool(p.config.ShowHooks),
			ShowOutput: settings.DerefBool(p.config.ShowHookOutput) || settings.DerefBool(p.config.HookOutput),
		},
	}
}

func buildUISession(v string, noConfig bool) (*tui.UISession, func(), error) {
	prep, err := setupAgentTools(noConfig)
	if err != nil {
		return nil, nil, err
	}
	sess, err := setupAgentSession(prep, v)
	if err != nil {
		return nil, nil, closeOnErr(err, prep.set, prep.mcp.set)
	}
	return sess, func() {
		if err := closeAll(prep.set, prep.mcp.set); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}, nil
}

type loaded struct {
	notice       string
	skills       []skills.Skill
	contextFiles int
	contextPaths []string
	contextChars int
	systemChars  int
}

// tokenEstimate guesses the tokens a prompt chunk costs at four characters
// to the token — the same order of guess a wrapped line makes, since the
// client has no real count before the first turn. It answers "is the prompt
// the reason my context is half gone", not an invoice: the run's own
// counter reports the backend's number once a turn has happened.
func tokenEstimate(chars int) int64 {
	return int64(chars / 4)
}

func augmentSystem(config *settings.Config, mcp connected) loaded {
	var found loaded
	config.System += environment(*config, time.Now(), mcp)
	if *config.ProjectContext {
		text, files, paths := projectContext(config.Root)
		config.System += text
		found.contextFiles = files
		found.contextPaths = paths
		found.contextChars = len(text)
	}
	if !*config.Skills {
		found.systemChars = len(config.System)
		return found
	}
	res := skills.LoadSkills(config.Root, *config.TrustSkills, config.SkillDirs)
	config.System += res.System
	found.notice = res.Notice
	found.skills = res.Skills
	found.systemChars = len(config.System)
	return found
}
