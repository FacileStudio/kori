package settings

import "github.com/FacileStudio/nacelle/mcp/client"

// merge overwrites every setting the layer above actually mentions, and leaves
// the rest alone.
func (c *Config) merge(over Config) {
	c.mergeStrings(over)
	c.mergeToggles(over)
	c.mergeSecurity(over)
	c.mergeUI(over)
	if over.MaxIterations != nil {
		c.MaxIterations = over.MaxIterations
	}
	if over.CompactAt != nil {
		c.CompactAt = over.CompactAt
	}
	if over.Budget != nil {
		c.Budget = over.Budget
	}
	if over.Fetch != nil {
		c.Fetch = over.Fetch
	}
	if len(over.SkillDirs) > 0 {
		c.SkillDirs = over.SkillDirs
	}
	for name, def := range over.MCP {
		if c.MCP == nil {
			c.MCP = map[string]client.ServerDef{}
		}
		c.MCP[name] = def
	}
	c.MCPFiles = append(c.MCPFiles, over.MCPFiles...)
	c.Hooks = append(c.Hooks, over.Hooks...)
	if len(over.Gates) > 0 {
		c.Gates = over.Gates
	}
}

// mergeStrings overwrites every string setting over actually mentions. A
// provider's backend, model, base URL and API key merge field by field, so a
// layer that sets only NACELLE_PROVIDER_BASE_URL leaves the rest alone.
func (c *Config) mergeStrings(over Config) {
	if over.Backend != "" {
		c.Backend = over.Backend
	}
	if over.Model != "" {
		c.Model = over.Model
	}
	if over.BaseURL != "" {
		c.BaseURL = over.BaseURL
	}
	if over.APIKey != "" {
		c.APIKey = over.APIKey
	}
	if over.Effort != "" {
		c.Effort = over.Effort
	}
	if over.Root != "" {
		c.Root = over.Root
	}
	if over.System != "" {
		c.System = over.System
	}
}

// mergeToggles overwrites every *bool setting over actually mentions.
func (c *Config) mergeToggles(over Config) {
	if over.Bash != nil {
		c.Bash = over.Bash
	}
	if over.ParallelAgents != nil {
		c.ParallelAgents = over.ParallelAgents
	}
	if over.Thinking != nil {
		c.Thinking = over.Thinking
	}
	if over.ProjectContext != nil {
		c.ProjectContext = over.ProjectContext
	}
	if over.Skills != nil {
		c.Skills = over.Skills
	}
	if over.TrustSkills != nil {
		c.TrustSkills = over.TrustSkills
	}
	if over.TrustHooks != nil {
		c.TrustHooks = over.TrustHooks
	}
	if over.Diffs != nil {
		c.Diffs = over.Diffs
	}
	if over.Tasks != nil {
		c.Tasks = over.Tasks
	}
	if over.Diagnostics != nil {
		c.Diagnostics = over.Diagnostics
	}
}

// mergeSecurity overwrites the security toggles over actually mentions.
func (c *Config) mergeSecurity(over Config) {
	if over.ApproveTools != nil {
		c.ApproveTools = over.ApproveTools
	}
	if over.PathIsolation != nil {
		c.PathIsolation = over.PathIsolation
	}
	if over.DenyElevation != nil {
		c.DenyElevation = over.DenyElevation
	}
	if over.EnvIsolation != nil {
		c.EnvIsolation = over.EnvIsolation
	}
}

// mergeUI overwrites every pointer field in UI that over actually mentions.
func (c *Config) mergeUI(over Config) {
	if over.GroupTools != nil {
		c.GroupTools = over.GroupTools
	}
	if over.ShowThinking != nil {
		c.ShowThinking = over.ShowThinking
	}
	if over.Continue != nil {
		c.Continue = over.Continue
	}
	if over.Resume != nil {
		c.Resume = over.Resume
	}
	if over.Mode != nil {
		c.Mode = over.Mode
	}
	if over.PromptPlaceholder != nil {
		c.PromptPlaceholder = over.PromptPlaceholder
	}
	if over.StartMessage != nil {
		c.StartMessage = over.StartMessage
	}
	if over.TransparentBlocks != nil {
		c.TransparentBlocks = over.TransparentBlocks
	}
	if over.JSON != nil {
		c.JSON = over.JSON
	}
}
