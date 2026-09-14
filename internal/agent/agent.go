// Package agent provides the agent configuration, tools, and execution loop.
package agent

import (
	"fmt"

	"github.com/FacileStudio/nacelle"
	"github.com/FacileStudio/nacelle/anthropic"
	"github.com/FacileStudio/nacelle/google"
	"github.com/FacileStudio/nacelle/openai"
	"github.com/FacileStudio/nacelle/openrouter"

	"github.com/FacileStudio/kori/internal/diagnostics"
	"github.com/FacileStudio/kori/internal/settings"
)

// Assembling the agent from settings, split out of main.go because that file
// sat on filet's 250-line cap and this is the seam already there: main.go
// decides what the settings are, and this turns them into the thing that
// answers. Nothing here reads a flag or touches the terminal.

// built is what build assembles: the agent, the backend it answers on, and the
// nacelle.Config it was built from. Grouped so build returns two values instead
// of four — the config is only there so /parallel can clone it for its own
// detached fan-out; callers that do not delegate ignore it.
type built struct {
	agent   *nacelle.Agent
	backend nacelle.Backend
	config  nacelle.Config
}

// build assembles the agent the settings describe, and hands the backend back
// so the caller can say which one answered. approve is nil unless
// -approve-tools was asked for — see nacelle.Approve's own doc comment for
// why nil, not a rubber-stamp function, is what "off" means here.
//
// The three reasoning settings fold into one nacelle.Thinking here, and the
// one rename in that fold is worth knowing about: this client's -thinking
// becomes Show, which decides what the transcript displays and nothing else:
// the model reasons, is billed, and replays its reasoning either way.
//
// built.config is the nacelle.Config the agent was built from, kept so a
// /parallel fan-out runs its own agents from the same tools, system prompt and
// iteration ceiling instead of a hand-built subset.
func build(config settings.Config, local []nacelle.Tool, approve nacelle.Approve, hooks map[nacelle.HookPoint][]nacelle.Hook) (built, error) {
	approve = unwrapCallTool(approve)
	backend, err := chosen(config)
	if err != nil {
		return built{}, err
	}

	retrying := nacelle.Retry(backend, nacelle.RetryOptions{})
	local, err = withParallelAgents(config, retrying, local, approve)
	if err != nil {
		return built{}, err
	}
	local = withTasks(config, local)
	if settings.DerefBool(config.Diagnostics) {
		local = append(local, diagnostics.Tool())
		hooks = withDiagnosticsHook(hooks)
		diagnostics.UseChain(chainOf(config.Gates))
	}

	cfg := nacelle.Config{
		Backend: retrying,
		System:  config.System,
		Thinking: nacelle.Thinking{
			Effort: nacelle.Effort(config.Effort),
			Budget: *config.Budget,
			Show:   *config.Thinking,
		},
		Tools:         local,
		MaxIterations: *config.MaxIterations,
		Approve:       approve,
		Hooks:         hooks,
	}
	agent, err := nacelle.New(cfg)
	if err != nil {
		return built{}, err
	}
	return built{agent: agent, backend: backend, config: cfg}, nil
}

// chosen builds the backend the settings ask for.
//
// An unknown name is refused rather than quietly falling back to a model the
// caller did not choose and will be billed for, which is the same reason the
// library itself ships no default backend.
func chosen(config Config) (nacelle.Backend, error) {
	switch config.Backend {
	case "anthropic":
		if config.BaseURL != "" || config.APIKey != "" {
			return nil, fmt.Errorf("anthropic takes no custom endpoint: base_url and api_key apply only to google, openai, or openrouter")
		}
		return anthropic.New(anthropic.Config{Model: config.Model}), nil
	case "google":
		return google.New(google.Config{
			Model:   config.Model,
			BaseURL: config.BaseURL,
			APIKey:  config.APIKey,
		})
	case "openai":
		return openai.New(openai.Config{
			Model:   config.Model,
			BaseURL: config.BaseURL,
			APIKey:  config.APIKey,
		})
	case "openrouter":
		if config.Model == "" {
			return nil, fmt.Errorf("openrouter needs a model: pass -model, or set model in ~/%s", settings.ConfigFile)
		}
		return openrouter.New(openrouter.Config{
			Model:   config.Model,
			BaseURL: config.BaseURL,
			APIKey:  config.APIKey,
		})
	default:
		return nil, fmt.Errorf("unknown backend %q, want anthropic, google, openai, or openrouter", config.Backend)
	}
}
