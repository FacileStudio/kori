package tui

import (
	"fmt"
	"strings"

	"github.com/FacileStudio/nacelle"

	"github.com/FacileStudio/kori/internal/herdr"
	"github.com/FacileStudio/kori/internal/provider"
	"github.com/FacileStudio/kori/internal/sessions"
	"github.com/FacileStudio/kori/internal/settings"
	"github.com/FacileStudio/kori/internal/usage"
)

// boot wires the session config the model was built from into its live fields —
// the ones that arrive as pointers or are only meaningful on the running model
// rather than at construction.
func boot(m *Model, c UISession) {
	m.groupTools = c.GroupTools != nil && *c.GroupTools
	m.Expanded = c.ShowThinking
	m.run.root = c.Root
	m.run.diffs = c.Diffs
	m.delegate = c.DelegateConfig
	m.activeBackend = c.Backend
	m.activeModel = c.Model
	m.activeBaseURL = c.BaseURL
	m.activeAPIKey = c.APIKey
	m.mode = renderMode(c.Mode)
	m.transparent = c.TransparentBlocks
	m.diagLoop = c.Startup.Diagnostics
	m.sink = usage.NewSink(c.Root, c.Model)
	if c.Resume != "" || c.AutoResume {
		if res := sessions.RestoreAtLaunch(c.Resume, c.Root, c.AutoResume); res.Path != "" {
			m.session = sessions.OpenResumeSession(res.Path, c.Backend, c.Model, c.Root)
		}
	}
	if m.session == nil {
		m.session = sessions.OpenSession(c.Backend, c.Model, c.Root)
	}
	herdr.SetSession(m.herdrClient, m.session.Path())
}

// startupContextNote says which context files the system prompt grew by and
// what that growth roughly costs, before the first turn has ever been sent —
// the one moment a context bill cannot be read off the run counter, which
// starts at nothing and only ticks after the user has spoken. The paths are
// named because the count the banner carries cannot answer "which file did
// this"; the figure is the same four-characters-to-the-token guess, named a
// guess by the tilde, not a backend count.
func startupContextNote(c LaunchContext) string {
	if len(c.ContextPaths) == 0 {
		return "context: no files loaded"
	}
	return fmt.Sprintf("context: %s loaded · ~%s tokens · %s",
		countedNoun(len(c.ContextPaths), "file"), shortTokens(c.ContextTokens),
		strings.Join(c.ContextPaths, ", "))
}

// profileBackend builds the backend a profile names, resolving a key the profile
// left to a command. Switching profiles mid-session reads the profile itself
// rather than the resolved config, so this is where its api_key_command runs —
// under the same rule the session applied at startup: a literal key wins, the
// command fills what is empty, and a command that fails stops the switch instead
// of quietly answering with no credential.
func profileBackend(p settings.Profile) (nacelle.Backend, string, error) {
	apiKey := p.Provider.APIKey
	if apiKey == "" {
		key, err := settings.KeyFromCommand(p.Provider.APIKeyCommand)
		if err != nil {
			return nil, "", err
		}
		apiKey = key
	}
	backend, err := provider.New(provider.Config{
		Backend: p.Provider.Backend,
		Model:   p.Provider.Model,
		BaseURL: p.Provider.BaseURL,
		APIKey:  apiKey,
	})
	if err != nil {
		return nil, "", err
	}
	return backend, apiKey, nil
}

// startupPrint hands the pre-queued banner lines to the terminal — unless it is
// running on the alternate screen, where Println would be a no-op and the lines
// are held to be drawn back into the view instead.
func startupPrint(m *Model) {
	if m.mode == modeTUI {
		return
	}
	for _, line := range m.unprinted {
		fmt.Println(line)
	}
	m.unprinted = nil
}
