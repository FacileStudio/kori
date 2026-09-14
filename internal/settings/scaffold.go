package settings

import (
	"fmt"
	"os"
)

// Template is the scaffold written to ~/.kori.yml on first boot: every
// setting present, every value the default. The point is discoverability —
// someone opening the file sees the whole surface with names to grep for,
// instead of an empty file and a docs page.
const Template = `provider:
  backend: anthropic
  model: ""
  base_url: ""
  api_key: ""

session:
  root: .
  system_prompt: ""
  continue: false

limits:
  max_iterations: 5
  compact_at: 75000
  grind_min_cost: 0
  grind_min_tokens: 0
  grind_continuations: 2

tools:
  run_command: true
  web_fetch: true
  tasks: true
  parallel_agents: true
  diagnostics: true
  search_content: true
  find_files: true

security:
  approve_tools: false
  path_isolation: false
  deny_elevation: true
  env_isolation: false

reasoning:
  effort: ""
  thinking: true
  budget: 0

discovery:
  project_context: true
  skills: true
  trust_skills: false
  trust_hooks: false

ui:
  rendering_mode: tui
  group_tools: true
  show_thinking: true
  diffs: true
  prompt_placeholder: "Ask something. Esc stops a run, ctrl+c stops or quits, ctrl+\\ forces it."
  transparent_blocks: true
  cron_list_json: false

sources:
  skill_dirs: []
  mcp: {}

hooks: []
# Scheduled jobs no longer live in this file: one YAML file per job under
# ~/.kori/jobs/ (e.g. news.yml), approved with kori cron trust <name>.
`

// Scaffold writes the template when no config file exists yet, and reports
// whether it did. An existing file is never touched: the file is the user's
// answer to "what do I want", not a cache. A parse error in an existing file
// is surfaced by Load, not papered over here.
func Scaffold(path string) (bool, error) {
	if path == "" {
		return false, nil
	}
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("stating %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(Template), 0o644); err != nil {
		return false, fmt.Errorf("writing %s: %w", path, err)
	}
	return true, nil
}
