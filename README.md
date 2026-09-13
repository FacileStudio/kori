# nacelle-tui

Terminal coding agent — the human harness for the
[nacelle](https://github.com/FacileStudio/nacelle) agent SDK.

The binary is named `nacelle`. The SDK is a Go library for building agents;
this program is its first consumer and lives to exercise every part of it from
a terminal, where someone is watching: text, reasoning, tools starting and
finishing, why a turn ended, what it cost. It is deliberately small — sessions,
profiles and panes are what a product grows, not what a contract test needs.

## What it does

- Streams one model turn at a time in a full-screen Bubble Tea v2 interface
- Runs against any backend the SDK ships: `anthropic`, `google`, `openai`, or `openrouter`
- Lets the model read and edit files under a root you choose, run commands when
  `-bash` is on, fetch web pages, and call MCP server tools from files
  every other client already has (`-mcp ~/.claude/.mcp.json`)
- Lets the model lay a large job out as steps and keep them current while it
  works, drawn live above the prompt and scrolled to the step in flight
- Fans independent side tasks out to concurrent nested runs with `-subagents`,
  so a wide search or a log dump costs the conversation one answer instead of
  its whole output
- Discovers project context (CLAUDE.md, AGENTS.md) and skills into the system
  prompt, each behind its own flag
- Gates tool calls behind an approval prompt with `-approve-tools`, and trusts
  project hook files only after an explicit, remembered decision

## Stack

| Layer | Tech |
|---|---|
| TUI | Go 1.26.4, `charm.land/bubbletea/v2`, lipgloss v2, glamour v2 |
| Agent | [FacileStudio/nacelle](https://github.com/FacileStudio/nacelle), pinned by tag |
| State | `~/.nacelle.yml`, `~/.nacelle/hooks.json` for hook trust |
| Release | GoReleaser, GitHub Actions on tag push, Homebrew tap `FacileStudio/tap` |

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/FacileStudio/nacelle-tui/main/install.sh | bash
```

Installs to `~/.local/bin` via [facile](https://github.com/FacileStudio/facile), the suite
installer. Pass `--bin-dir <dir>` to change that, `--source` to build from source.

Already have `facile`:

```sh
facile install nacelle
```

Or Homebrew:

```sh
brew install FacileStudio/tap/nacelle
```

## Usage

Run it in the directory you want it to work in:

```sh
nacelle
```

It reads API keys from the environment: `ANTHROPIC_API_KEY` for the default
backend, `GEMINI_API_KEY` (or `GOOGLE_API_KEY`) for `-backend google`,
`OPENAI_API_KEY` for `-backend openai`, and `OPENROUTER_API_KEY` for `-backend openrouter`.

Settings layer bottom-up: defaults, then `~/.nacelle.yml`, then `NACELLE_*`
environment variables, then flags. The useful ones:

| Flag | Env | What |
|---|---|---|
| `-backend` | `NACELLE_BACKEND` | `anthropic`, `google`, `openai`, or `openrouter` |
| `-model` | `NACELLE_MODEL` | model id; the backend's own default otherwise |
| `-root` | `NACELLE_ROOT` | directory the file tools may reach |
| `-bash` | `NACELLE_BASH` | let the model run commands (off by default) |
| `-continue` | — | auto-resume the newest session for the current project |
| `-resume` | — | resume a specific session by id or file path |
| `-no-config` | — | start with default settings, ignoring ~/.nacelle.yml |
| `-tasks` | `NACELLE_TASKS` | task planning tool (on by default) |
| `-approve-tools` | `NACELLE_APPROVE_TOOLS` | ask before every tool call runs |
| `-subagents` | `NACELLE_SUBAGENTS` | give the model the parallel delegate tool (on by default) |
| `-max-iterations` | `NACELLE_MAX_ITERATIONS` | how many times the model may be asked |
| `-mcp` | — | MCP servers file (repeatable) |
| `-skill-dir` | `NACELLE_SKILL_DIRS` | extra skills directory (repeatable) |

`nacelle -version` prints exactly `nacelle <semver>`. See `-h` for the full set:
reasoning effort and budget, web fetch, project-context and skill
discovery, hooks trust.

Full settings reference: [docs/configuration.md](docs/configuration.md).

## Configuration

Settings live in `~/.nacelle.yml`, **written on first boot with every default
explicitly set** — delete it to regenerate. An existing file is never touched.
The example file with all defaults, and the full reference with the
precedence order and the traps in each setting:
[docs/configuration.md](docs/configuration.md).

`example.nacelle.yml` in this repo is the same file the first boot writes:

```yaml
provider:
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

tools:
  run_command: true
  web_fetch: true
  tasks: true
  parallel_subagent: true

security:
  approve_tools: false
  path_isolation: false
  env_isolation: false
  deny_elevation: true

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
  rendering_mode: inline
  group_tools: true
  show_thinking: true
  prompt_placeholder: "Ask something. Esc stops a run, ctrl+c stops or quits, ctrl+\\ forces it."
  start_message: ""
  transparent_blocks: false
  cron_list_json: false

sources:
  skill_dirs: []
  mcp: {}

hooks: []
```

Scheduled jobs are not configured here anymore: one YAML file per job under
`~/.nacelle/jobs/`, trusted with `nacelle cron trust <name>` before it runs.

## herdr

Run inside [herdr](https://herdr.dev), `nacelle` reports its live state and
session identity over herdr's socket API (`internal/herdr/`). A nacelle pane
shows as an agent with an idle / working / blocked state, and herdr holds a
reference to the run's transcript. This needs no herdr binary update and works
on any machine, including stock herdr.

After a herdr **server restart**, herdr restores a nacelle pane as a plain
shell in its saved directory — nacelle is not in herdr's compiled-in resume
table, and no config or plugin adds it. Reopen the session in that directory
with `nacelle` (auto-resumes the newest session by cwd) or
`nacelle --resume <transcript-path>`. Real auto-restore awaits herdr adding
nacelle to its resume table; `--resume` already accepts the exact absolute
transcript path the reporter reports.

## Structure

```
main.go             Entrypoint: calls agent.Run
internal/agent/     Agent lifecycle, tools wiring, headless mode, banner, flags
internal/approval/  Interactive and batch tool call approvals
internal/cost/      Token usage and cost calculation
internal/diff/      Syntax-highlighted unified diff generator
internal/history/   Command history and navigation
internal/layout/    Terminal dimensions and line truncation
internal/menu/      Autocompletion menu for commands and skills
internal/queue/     Input queueing during active turns
internal/sessions/  Session persistence, listing, rotation, resume
internal/settings/  CLI flags, ~/.nacelle.yml, and environment configuration
internal/skills/    Agent skills discovery and execution
internal/status/    Spinner and progress status indicator
internal/tasks/     Task planning tool, validation, step updates
internal/theme/     Terminal color palettes and syntax themes
internal/thinking/  Collapsible reasoning viewport
internal/toolview/  Compact and grouped tool rendering
internal/tui/       Bubble Tea v2 model, key handling, rendering, slash commands
internal/usage/     Token accounting and context window headroom
internal/herdr/     Reports agent state and session identity to herdr over its socket API
```

---

Part of the [Facile Suite](https://facile.studio) — self-hosted tools for creative studios
and freelancers. One login, zero cloud dependency.
