package settings

import (
	"fmt"
	"os"
)

// Template is the scaffold written to ~/.kori.yml on first boot: every
// setting present, every value the default. The point is discoverability —
// someone opening the file sees the whole surface with names to grep for,
// instead of an empty file and a docs page.
//
// example.kori.yml in the repo root is this same text, byte for byte, and
// TestExampleConfigIsTheScaffoldTemplate fails when the two drift apart: the
// README promises the example is what a first boot writes.
const Template = `# kori settings — every setting, every default, explicit.
# Written here on first boot; delete this file and it is written again.
provider:
  backend: anthropic
  model: ""
  base_url: ""
  api_key: ""

session:
  root: .
  # system_prompt replaces the built-in harness prompt outright; leave it empty
  # to keep kori's own tool and safety guidance.
  system_prompt: ""
  # additional_prompt is appended after the base prompt and everything layered
  # onto it: the place for a specialist persona that must not cost the harness
  # guidance. It composes with system_prompt rather than competing with it.
  additional_prompt: ""
  continue: false

limits:
  max_iterations: 5
  # compact_at is an absolute token ceiling. Left unset (the default) the
  # ceiling comes from the backend's context window and soft_ratio below; 0
  # turns compaction off outright.
  # compact_at: 75000
  max_concurrency: 16
  # max_parallel_agents: 16
  # grind budget: the per-run minimum spend a run must reach before stopping
  # counts as finished. Both floors at 0 turn it off; grind_continuations caps
  # how many continuation notices one run can be given. grind_continuations: 0
  # turns the budget off even when a floor is set.
  grind_min_cost: 0
  grind_min_tokens: 0
  grind_continuations: 2
  # compaction decides, per history block, what to keep, prune or fold into the
  # state ledger. The judge is OPT-IN and off by default: enabling it sends
  # conversation history to TypeSafe (see docs/configuration.md); its key is
  # TYPESAFE_API_KEY, with judge.api_key as the file-side alternative.
  compaction:
    soft_ratio: 0.65
    mid_ratio: 0.80
    hard_ratio: 0.90
    keep_turns: 3
    anchor_messages: 1
    judge:
      enabled: false
      model: jev-latest
      base_url: https://api.typesafe.ai
      api_key: ""
      prune_threshold: 0.85
      max_blocks_per_call: 64

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
  # deny_elevation refuses sudo-style elevation in run_command: a policy guard
  # against accidents and injected instructions, not a security boundary.
  # OS-level enforcement is the real wall.
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
  show_hooks: true
  show_hook_output: true

editor:
  editor: ""              # empty uses the system EDITOR; set a path to force one (e.g. /usr/bin/vim)
  prompt_edit_key: ctrl+g # opens the external editor on the prompt

sources:
  skill_dirs: []
  mcp_files: []
  mcp: {}

hooks: []
# Scheduled jobs no longer live in this file: one YAML file per job under
# ~/.kori/jobs/ (e.g. news.yml), trusted with: kori cron trust <name>
gates: []

# Sandbox: the sandbox command runs kori on the host and sends every tool
# call into a local boite microVM over SSH. Uncomment targets: to name one.
sandbox:
  default: ""
  vm_name: ""
  port: 2226
  ssh_key_path: ~/.ssh/id_ed25519
  # an empty root uses the VM's own workspace, where boite mounted the project
  root: ""
  auto_snapshot: false
  # targets:
  #   dev-vm:
  #     vm_name: ""
  #     port: 2226
  #     ssh_key_path: ~/.ssh/id_ed25519
  #     root: ""
  #     auto_snapshot: false

# Remote: the remote command runs kori on the host and sends every tool call
# to an SSH host. A bare user@host:port argument works without an entry here.
remote:
  default: ""
  user: ""
  # 0 lets ssh choose: ssh_config's Port, then ssh's own default (22)
  port: 0
  # an empty ssh_key_path lets ssh choose: ssh_config's IdentityFile, then the agent
  ssh_key_path: ""
  # an empty root starts in the SSH login directory, which is the user's home
  root: ""
  # targets:
  #   staging:
  #     host: staging.example.com
  #     port: 22
  #     user: deploy
  #     ssh_key_path: ""
  #     root: /srv/app
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
