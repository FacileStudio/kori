package tui

import (
	"fmt"
	"strings"

	"github.com/FacileStudio/kori/internal/herdr"
	"github.com/FacileStudio/kori/internal/sessions"
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
	m.mode = renderMode(c.Mode)
	m.transparent = c.TransparentBlocks
	m.diagLoop = c.Startup.Diagnostics
	m.sink = usage.NewSink(c.Root, c.Model)
	m.session = sessions.OpenSession(c.Backend, c.Model, c.Root)
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
