# Changelog

## [Unreleased]

## [0.65.0] - 2026-09-15

### Fixed
- Re-render diff and command output blocks on terminal resize.
- Skip empty answer blocks from whitespace-only model output.

## [0.64.0] - 2026-09-15

### Added
- Hook execution reporting in the interactive conversation display, showing hook event type, tool, command, execution duration, and exit status.
- Hook stdout and stderr output preview in conversation with configurable line truncation (6 lines by default with ellipsis overflow indicator).
- Hook display configuration settings (`ui.show_hooks`, `ui.show_hook_output`, `ui.hook_output`), environment variables (`KORI_SHOW_HOOKS`, `KORI_SHOW_HOOK_OUTPUT`, `KORI_HOOK_OUTPUT`), and CLI flags (`--show-hooks`, `--show-hook-output`).

## [0.63.1] - 2026-09-15

### Fixed
- Interactive prompt external editor execution via `Ctrl+E` using `tea.ExecProcess` to pause Bubbletea and restore terminal raw mode, preventing `read /dev/stdin: resource temporarily unavailable` crashes.
- Allow opening the external editor via `Ctrl+E` when the prompt input is empty.
- Prompt textarea border gutter width reservation (`prompt.SetPromptFunc(2, ...)`), preventing line overflow and cursor displacement over the left border on soft-wrapped lines.


### Added
- External prompt editor configuration (`editor:` section in settings with `editor` and `prompt_edit_key`, and `KORI_EDITOR` / `KORI_PROMPT_EDIT_KEY` environment variables).
- `ctrl+e` keybinding in prompt to open and edit prompt content in an external editor (`GIT_EDITOR` > `EDITOR` > `VISUAL` > `vi`).

## [0.62.0] - 2026-09-15

### Added
- Configurable `search_content` and `find_files` flags and settings toggles (`--search-content`, `--find-files`, `KORI_SEARCH_CONTENT`, `KORI_FIND_FILES`).
- `mcp_files` setting in YAML configuration and `KORI_MCP_FILES` (alongside `NACELLE_MCP_FILES`) environment variable for loading external MCP server definition files.
- Subprocess user directory PATH discovery (`~/.local/bin`, `~/.cargo/bin`, `~/.bun/bin`) ensuring MCP servers and tools can locate binaries.
- Home directory tilde (`~`) expansion in MCP server commands, arguments, working directories, and environment variables.

### Changed
- Raised MCP catalog threshold from 20 to 50 tools, allowing common multi-server setups to expose tools directly rather than requiring two-stage discovery (`search_tools`/`call_tool`).
- Updated system prompt instructions to guide models to prefer mounted MCP tools over running arbitrary shell commands with `run_command`.

### Fixed
- Argument normalization in `call_tool` for models passing stringified JSON or null arguments.
- Fallback suffix matching in MCP tool catalog lookup to resolve doubled server name prefixes.
- Toggle merging in cron job override application (`cronapply.go`).
- Removed phantom `list_directory` reference from default system prompt.

## [0.61.1] - 2026-09-14

### Fixed
- Restore `sudo` in `shellElevationCommands` map (replacing `kori` which was mistakenly inserted during the rename).
- Refactor `security.deny_elevation` command scanning to target executable positions across shell command segments instead of testing all whitespace arguments, eliminating abusive false positives on commit messages, flags, regex patterns, and setuid file arguments.

## [0.61.0] - 2026-09-14

### Added
- Cobra and Fang v2 CLI command tree (`cmd/`), adding structured subcommands for `bench` and `cron` with `--runs` and `--json` flags.
- Alternate-screen transcript dynamic reflow on terminal window resize in TUI mode, retaining raw transcript entries and re-rendering on width change.

### Fixed
- Slice memory retention in `tuireflow.go` when capping held entries.
- Flag argument normalization in `cmd/` now respects the `--` delimiter.
- Sudo command documentation drift in `docs/sudo.md`.

## [0.60.0] - 2026-09-14

### Changed
- The project is renamed to `kori`. The repository (`FacileStudio/kori`), the
  goreleaser project name and binary, the Homebrew cask, the facile catalog
  entry and the documentation move from `nacelle-tui`/`nacelle` to `kori`, and
  `example.nacelle.yml` is renamed `example.kori.yml`. The `nacelle` SDK keeps
  its name: kori stays the terminal harness for the nacelle agent SDK.
- The changelog URLs for all releases now point to `github.com/FacileStudio/kori`
rather than `github.com/FacileStudio/nacelle-tui`.
- All internal documentation now uses `kori` instead of `nacelle-tui`, and
  releases on GitHub and homebrew refer to `kori` rather than `nacelle-tui`.
- The historical changelog entries retain their `nacelle-tui` URLs for archive
  purposes; new releases are tagged and documented under `kori`.

## [0.58.1] - 2026-09-14

### Fixed
- Shell teardown errors surface instead of being discarded: pipe and process
  cleanup report through `errors.Join` to their callers, a failed read
  deadline marks the session broken, and `Close` reports what it hit.
- The system prompt's bash rules now say the persistent shell keeps state
  between calls (when strict path isolation is off), matching the tool
  description.

## [0.58.0] - 2026-09-14

### Added
- `run_command` can hold one persistent shell session: `cd`, exports and shell
  options survive between calls. State turns the result into a stateless run
  the same shape, and the session survives the shell being killed or exited by
  respawning it. Disabled when strict path or environment isolation is on.
- `nacelle bench` runs a prompt through the headless path one or more times
  and reports duration, tokens, cost, tool calls and final context per run,
  with `--json` output.
- Grind budgets: `grind_min_cost` and `grind_min_tokens` set a per-run minimum
  spend the run must reach before stopping counts as finished;
  `grind_continuations` caps the notices (0 turns the budget off).
- Parallel agent tasks accept an inline title, and fan-outs without one get a
  short generated title instead of the raw task text.

### Fixed
- A crypto/rand failure in the session shell's done-marker no longer falls
  back to a constant nonce; the call fails as shell-down instead.
- The post-command drain uses a short positive read deadline, so leftover
  background output is caught instead of resurfacing in the next call.

## [0.57.1] - 2026-09-13

### Fixed
- The live streaming region kept no blank row above itself, and client lines
  rendered inside the margin lost their spacing; both now hold one blank row.

All notable changes to `kori` are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow semver —
while on `v0`, a breaking change bumps the minor.

## [0.57.0] - 2026-09-13

### Changed
- Every tool call in the conversation shares the `⏺` glyph; only shell runs
  keep `$`. Tool call lines use a single space after the glyph, and tool
  batching classifies calls by name rather than by glyph.

### Fixed
- The blank row between two consecutive answer panes broke the left spine;
  it is now painted as an empty pane row of its own.

## [0.56.0] - 2026-09-13

### Added
- The `session_start` hook point is accepted in `hooks:` entries and its
  exit-0 stdout is injected into the conversation, so a hook like `cercle sync`
  can drop inbox state in before the first model call.

## [0.55.0] - 2026-09-13

### Added
- The answer's left-border spine paints while the block streams, not only
  once it finishes: the live region renders through the same pane painter the
  committed block uses.

### Fixed
- Committed transcript lines sat flush against each other when they arrived
  in separate flushes (thinking trace, then tool call, then result): every
  batch after the first now leads with one blank row, matching the spacing
  inside a batch.
- Skill menu items read in yellow, matching the input prompt, instead of the
  blue-green nothing else used. Tool call lines put a wider gap after the
  pencil, read and write glyphs, which ambiguous-width terminals render two
  cells wide and swallow the space after.

## [0.54.0] - 2026-09-13

### Added
- Cron jobs move to their own folder: one YAML file per job under
  `~/.nacelle/jobs/` (news.yml, healthchecks.yml, ...), the filename stem as
  the default name. The folder is rescanned on every cron command, so a file
  added or removed takes effect on the next invocation with no daemon. The
  inline `cron:` key is removed — a config still carrying it fails at load
  with a pointer to the folder (breaking).
- Job trust: a new job file shows in `nacelle cron list` immediately but
  `cron run` and `cron install` refuse it until `nacelle cron trust <name>`
  shows the file and records the human's approval. Trust keys on the file
  path and content hash in `~/.nacelle/trust.json` (the hooks trust store,
  generalized; existing `hooks.json` approvals carry over), so an edited job
  re-arms the gate.
- Per-job overrides: a job file can carry `provider:`, `security:`, `tools:`,
  `reasoning:`, `limits:` plus `hooks:` and `gates:` lists. Empty members
  fall back to the config; `provider.model` beats the flat `model:` key;
  the approval gate and shell stay disarmed regardless of what the job asks
  for.
- Cron run metrics: every `cron run` appends one JSON line to
  `~/.local/state/nacelle/cron.jsonl` — name, start, duration, status, model,
  tool calls, tokens in and out, cost, final context size, and the number of
  compactions.

### Fixed
- Transcript spacing: committed blocks, tool lines, results and turn
  boundaries sat one to four blank rows apart depending on what rendered
  them. Spacing is decided in one place now — exactly one blank row between
  any two committed lines.

## [0.53.9] - 2026-09-13

### Fixed
- Parallel agent rows register again under the prompt: the return-control
  prose appended to the `parallel_agents` stub made the batch decoder fail,
  so the fan-out ran detached with no status lines. The decoder now reads the
  first JSON value and ignores the prose.
- Answer panes sit flush against their border again: glamour's restored stock
  styles brought a document margin of two spaces on top of the border's own,
  leaving three columns before the text. The margin is dropped.

## [0.53.8] - 2026-09-13

### Added
- Parallel tool results now carry the return-control rule in the stub itself:
  the model is told to end its turn at the moment the fan-out dispatches.

### Changed
- Markdown renders with glamour's stock dark/light styles, colours included,
  the way glow does, instead of stripping every colour field by reflection.
- `/skill:name` highlighting resets bold and foreground together, so the
  token's bold no longer leaks into the rest of the row.
- Streaming commits whole paragraphs to the scrollback at blank-line
  boundaries, so tables and lists are never rendered half-structured.

### Fixed
- Requeue follows the edit anchor, not text similarity: a full rewrite of a
  recalled queued line lands in place, and a fresh message typed after
  browsing history is never swallowed as a phantom edit.

## [0.53.7] - 2026-09-13

### Added
- `ctrl+j` and `ctrl+k` move the slash command suggestion menu selection down
  and up, alongside the existing arrow keys.
- `/skill:name` tokens in the input prompt and transcript render bold as well
  as yellow.

### Changed
- The ASCII nacelle banner is now the default `start_message`; a fresh
  `~/.nacelle.yml` scaffold no longer writes an explicit empty value that
  would have silenced it.

## [0.53.6] - 2026-09-13

### Changed
- The selected row in the slash command suggestion menu renders white
  foreground, including the `/command` value inside it, instead of the
  previous dim adaptive color.

## [0.53.5] - 2026-09-13

### Added
- `/skill:name` tokens render yellow in the input prompt while typing
  (including half-typed prefixes and multiple tokens) and in sent questions
  in the transcript.

## [0.53.4] - 2026-09-13

### Fixed
- Alt+Enter no longer goes dead after pasting ten or more lines: the prompt's
  `MaxHeight` acted as a content gate that made bubbles silently drop every
  newline insertion past ten logical lines. The gate moved to
  `MaxContentHeight`, so `MaxHeight` is only the viewport cap and the prompt
  scrolls instead of eating newlines.

## [0.53.3] - 2026-09-13

### Fixed
- Skill descriptions parsed from folded block scalars (`description: >`)
  collapse to a single line, so the skills menu stops rendering a blank
  second line for each of them.

## [0.53.2] - 2026-09-13

### Fixed
- Multiline paste works again. Bracketed paste was disabled, so a pasted
  newline arrived as a raw Enter and submitted each line as its own message;
  with it re-enabled, the bubbles textarea sanitizer still flattened pasted
  newlines to spaces. Pastes now insert line by line, newline between them.
- `parallel_agents` no longer rejects a fan-out when the model sends `tasks`
  as a string or a double-encoded array (nacelle v0.25.1).

## [0.53.1] - 2026-09-12

### Fixed
- The pinned prompt no longer jumps up a row when the slash suggestion menu
  opens: menu row accounting double-counted the separator blank.
- Subagent status lines render again in tui mode; the below-prompt block was
  not reserved from the frame height and got clipped off the bottom of the
  alternate screen.

## [0.53.0] - 2026-09-12

### Changed
- **Breaking:** gate chains lost `scope:`. Every gate now runs on the edited
  path (or the path the diagnostics tool names), same as the built-in filet
  check; whole-tree sweeps go through `run_command`. Configs carrying
  `scope:` fail to parse.

### Added
- `format: true` on a gate: the gate may rewrite the edited file, and the
  post-edit injection then answers `reformatted <path> — re-read it` instead
  of the gate's output, so the model knows its buffer went stale.
- Rejected SKILL.md manifests are named in the skills notice with their
  rejection reason, instead of vanishing silently.

## [0.25.0] - 2026-09-12

### Added
- **`CancelParallel(batch)`.** Cancels a detached fan-out by its batch key:
  each still-running task ends on the Results stream as a cancelled error,
  tasks that finished keep their results.
- **`NewParallelCancelTool` / `ParallelCancelToolName`.** A model-callable
  `parallel_cancel` tool, so an agent can stop its own fan-outs.
- **`tools.Config.DenyElevation` / `WithDenyElevation`.** Refuse run_command
  calls that try to elevate privileges (sudo, su, doas, pkexec).

## [0.52.0] - 2026-09-12

### Added
- Configurable gate chain: `gates:` in the settings (or a whole file via
  `--gates-file`) runs project checks after every edit and on demand through
  the diagnostics tool, with per-gate scope and timeout.
- `security.deny_elevation` (on by default): run_command calls that try to
  elevate privileges (sudo, su, doas, pkexec) are refused. Off switch in
  settings or `NACELLE_DENY_ELEVATION`.
- `/parallel cancel [batch]` stops live detached fan-outs; `parallel_cancel`
  gives the model the same lever.
- Parallel task rows show what each agent is doing between tool calls:
  thinking, waiting, or the tool call's live state.

## [0.51.0] - 2026-09-12

### Added
- Startup diagnostics note: when `tools.diagnostics` is on and a `filet.yml`
  (or `.filet.yml`) exists under the opened root, the launch transcript runs
  one `filet check` over the root and prints the result under the banner —
  the same compiler-style findings the post-edit loop injects, seen before
  the first turn instead of after the first edit.

## [0.50.0] - 2026-09-12

### Added
- Post-edit diagnostics loop: filet findings for a file just written by
  `edit_file`/`write_file` are injected into the session automatically
  (errors only, attribution-diffed so repeats are not re-injected, 2s
  timeout with a background warm). Kill switch: `tools.diagnostics: false`.
- `diagnostics` tool: on-demand filet check of one file or the repo,
  compiler-style `file:line:col` output.

### Fixed
- Subagent task rows laid out at full terminal width while the margin pass
  clips to two cells narrower, cutting off the timer at the right edge.
- `example.nacelle.yml` still listed the removed `parallel_subagent` key,
  so the example config failed to parse.

## [0.49.1] - 2026-09-12

### Fixed
- Markdown answers could leave every following line underlined in tmux-over-ssh
  setups: glamour's OSC 8 hyperlink sequences pass through tmux mangled and
  leave kitty with a hyperlink that never terminates. Links now render as plain
  text.
- The colourless style strip left glamour's hardcoded code-block background
  colour (`#373737`) in place; code blocks now use the terminal's own
  background.

## [0.49.0] - 2026-09-11

### Added
- **`security.env_isolation`**, off by default. With it on, MCP servers and
  `run_command` children start with a minimal PATH/HOME base instead of the
  environment nacelle-tui inherited from the shell. With it off — the
  default — children see the launching shell's full environment, so servers
  that shell out to PATH helpers or read exported keys work as they do by
  hand.

### Changed
- **`strict_confinement` is renamed `path_isolation`** to match the SDK's
  `tools.PathIsolation`; same behaviour.
- Markdown answers render in the terminal's own default colours: glamour's
  layout kept, its hardcoded palette stripped, emphasis carried by bold and
  underline only.

### Fixed
- Command output blocks had two extra spaces between the `▌` border and the
  text; one space now.

## [0.48.1] - 2026-09-11

### Fixed
- **The diff recap footer reads `+n -n`**: one margin space after the spine,
  one space between the counters (joined with the block background, not a
  bare reset), instead of two leading spaces and glued figures.
- **The input prompt's margin space sits inside the backdrop**, so the
  background bar starts flush against the `▌` spine instead of the gap
  floating on the terminal's default background.
- `main.go`'s fallback version string now matches the tag (it trailed at
  `v0.47.0` through the `0.48.0` release; GoReleaser's ldflags override hid
  it).

## [0.48.0] - 2026-09-11

### Added
- **A white left spine on the input prompt**, one space between the spine and
  the text, so no prompt row sits flush against the left edge. The spine moved
  from the muted tone to the new `Border` palette style.
- **A left spine on every row of a multi-line user question** in the
  transcript, wrapped or not — the question now reads as one bordered pane.

### Changed
- **Tool boxes carry a margin space after their left border**, so box text no
  longer touches the spine; every `Box` caller now sizes content two columns
  narrower and the live region stops double-margining rows that open with the
  border glyph.
- **One run recap instead of a per-turn boundary.** The muted
  "duration · tokens · cost" line that landed after every turn is gone; the
  same recap is said once when a run settles. The live footer already ticks
  the counts mid-run.
- **Thinking traces keep their one-space screen margin** when they are
  committed to the transcript, matching what streaming showed.
- **The diff footer sits one blank row below the diff** and the `+n -n` counts
  join without a gap.

### Fixed
- A run's final `done` event no longer overwrites the accumulated usage with
  an empty one, which would have made the run recap report zero tokens.

## [0.47.0] - 2026-09-11

### Added
- **TUI mode is the default rendering.** `rendering_mode` defaults to `tui` and
  `transparent_blocks` to `true`; set `rendering_mode: inline` to keep the old
  behaviour.
- **Startup context notes.** The transcript opens with the context files the
  system prompt grew by and a rough token cost, plus the whole system prompt's
  cost, before the first message is sent.
- **Shift+Enter inserts a newline** in the prompt, alongside Alt+Enter. On
  terminals that do not speak the Kitty keyboard protocol Shift+Enter stays
  Enter, which is unchanged.

### Changed
- **Wrapped block rows keep their border.** Diff, command-output and tool
  blocks wrap long lines themselves instead of leaving it to the terminal, so
  every continuation carries the pane's spine.
- **Diff line numbers sit one space off the left border.**
- **Tool lines and box spines are yellow while running**, finished green;
  parallel task clocks stay green at start.

### Fixed
- **Resizing mid-run re-wraps the transcript.** A shrink no longer clips held
  rows; rows painted at the old width are broken again at the new width,
  continuations keeping their border.
- **The approval gate sees the bridged tool's name in MCP catalog mode.**
  Allowing one tool no longer opens every bridged tool behind `call_tool`.
- **Wrapped prompt rows keep their background**, and the running-tool/queue
  spine and margin spacing were cleaned up.

## [0.46.0] - 2026-09-11

### Changed
- **The prompt has no configurable prefix.** `prompt_prefix` is gone; the
  prompt text starts directly after the status bar. The setting was useless
  next to the placeholder and only pushed the first typed line down.

### Added
- **A numbered gutter on diff views.** Every code row is numbered in a left
  gutter — green on additions, red on removals, muted on context. Removals keep
  their old-file count, everything else the new-file count.
- **Softened diff backdrops.** Added/removed line backgrounds fade to about
  20% over the pane backdrop instead of painting solid ANSI blocks, so the
  pane's own tint still shows through.

### Fixed
- **The prompt no longer wastes a blank row in `inline` mode.** A spurious
  empty line appeared under the prompt whenever the status line repainted; the
  prompt now hugs its content, with a little breathing room above the status
  line instead.

## [0.45.1] - 2026-09-11

### Fixed
- **The input prompt now sits flush at the bottom of the screen.** In `tui` mode
  the prompt used to float a row up with blank lines beneath it; in `inline`
  mode it drifted up when fresh content arrived. `tuiUpper` was counting the
  two-row status bar as a single row, so `len()` understated the region and the
  prompt (and cursor) landed a row too high. Each `aboveContent` entry is now
  split into its visual rows and nothing is painted beneath the prompt.
- **The cursor no longer sits a row above the typing line.** The cursor's y was
  computed from the same off-by-one region count; with the count fixed it lands
  exactly on the prompt's text row.
- **The prompt is one line tall by default.** Its `MinHeight` floor dropped
  from 3 rows to 1, so an empty or one-line input reads as a single line and
  only grows when the input actually wraps. The two padding rows under the text
  are gone.

## [0.45.0] - 2026-09-11

### Changed
- **`tools.parallel_agents` replaces `tools.parallel_subagent`.** The key,
  the `NACELLE_PARALLEL_AGENTS` variable and the `-parallel-agents` flag follow
  the tool's new name in nacelle v0.23.0 (`ParallelAgentsToolName`). A pre-0.45
  file carrying `parallel_subagent` fails to parse rather than silently
  disabling the tool.
- **`session:` group tightened.** `resume` left the file — it is
  `-resume` on the command line only; a `session.resume` key is refused. The
  `system` key is renamed `system_prompt` and the `-system` flag is now
  `-system-prompt` (`NACELLE_SYSTEM_PROMPT`).
- **`security:` group.** `approve_tools` and `strict_confinement` moved out of
  `tools:` — both decide how much a tool call may do, neither mounts a tool.

### Added
- **`-no-config`.** Starts with default settings, ignoring `~/.nacelle.yml`
  entirely: defaults plus environment plus flags. An invalid settings file no
  longer dies on the spot: nacelle prints a coloured report of every unknown
  field and offers one yes/no prompt to boot with defaults; declining exits
  with the configuration documentation link and the `-no-config` escape hatch.

## [0.44.0] - 2026-09-11

### Changed
- **First boot scaffolds the config.** When no `~/.nacelle.yml` exists,
  nacelle writes one with every setting present at its default value, so the
  whole surface is visible and greppable from day one. An existing file is
  never touched; deleting yours regenerates it. `example.nacelle.yml` in the
  repo is the same document.
- **BREAKING: the config file is grouped.** `~/.nacelle.yml` keys now live
  under their own family headers — `provider:`, `limits:`, `tools:`,
  `reasoning:`, `web:`, `discovery:`, `ui:` and `sources:` — instead of all
  sitting flat at the top level. `continue` and `resume` stay top level. The names inside each group are unchanged.
  A pre-0.44 file fails to parse with `KnownFields(true)` naming the flat
  keys, not silently ignoring them; docs/configuration.md carries the full
  old-key → new-location migration table. Three keys also renamed to say
  what they do: `mode` → `rendering_mode`, `json` → `cron_list_json`, and
  `reasoning_budget` → `reasoning.budget`. Flags and `NACELLE_*` environment
  variables are untouched.

## [0.43.1] - 2026-09-11

### Fixed
- **Flags typed ahead of the `cron` subcommand are honoured.** The dispatch
  only recognised `cron` as the very first argument, so `nacelle --json cron
  list` opened the TUI instead of printing the job list; the scan now skips
  leading dash-tokens. A flag that takes a value still belongs after the
  subcommand.
- **The search glob quirk is fixed at the source.** nacelle v0.22.1 makes
  `search_content` and `find_files` match relative globs against the walk's
  paths; this release picks it up.

### Changed
- nacelle v0.22.0 → v0.22.1.

## [0.43.0] - 2026-09-11

### Added
- **Background scheduling (cron), phase 1.** `nacelle cron run|list|install`
  arms a job from the settings file as a systemd service and timer: unattended
  runs deliver their transcript to a file or a webhook, `install` refuses a
  disabled job so a test run always comes first, and generated units carry
  `WorkingDirectory` and a start timeout that actually bounds the run.
- **`cron list --json`** prints one JSON document on stdout, colors off, for
  scripts.
- **TUI render mode** (`-mode tui`): the transcript paints as fixed rows
  instead of reflowing, with the held scrollback capped so per-frame cost
  stays bounded on long sessions.
- **Transparent tool blocks** (`-transparent-blocks`): tool output renders
  without the surrounding box.

### Fixed
- **Compaction works on endpoints where reasoning is mandatory.** The
  summarizer sent an explicit reasoning-off, which OpenRouter endpoints with
  mandatory reasoning reject with a 400; the request now omits the reasoning
  key entirely, so models that default off stay off and mandatory models use
  their own default.
- **Usage errors exit 2, real errors exit 1.** `cron` dispatch accepts
  `help`/`-h`/`--help`, per the CLI standard.
- **Cron delivery targets validate before the billed run**, and a failed log
  append no longer hides behind a successful run (`errors.Join`).
- Headless stdout write errors propagate instead of being discarded.

## [0.42.2] - 2026-09-11

### Fixed
- **The eviction cut never splits a tool pair.** A compaction pass could land
  its cut between an assistant `tool_use` message and the immediately-following
  user `tool_result` message, so the rebuilt tail opened with a tool result
  whose call id had been evicted. Anthropic rejects that with a 400. The cut is
  now pulled back one message to keep the pair together, and the three cut sites
  (post-turn, `/compact`, manual) all route through the same alignment.

## [0.42.1] - 2026-09-11

### Fixed
- **Compaction no longer freezes the spinner or strands queued lines.** A
  post-turn or `/compact` pass now keeps its elapsed timer ticking instead of
  stopping after the first frame, and the lines queued behind a running pass
  are delivered from the outcome handlers once the rebuilt conversation is in
  place — no more racing the install or quietly dropping them.
- **The summarizer runs reasoning-off.** A reasoning-first model spent the
  output budget on its chain of thought and streamed back no final text, so a
  pass landed empty and fell back to the weak mask. With reasoning off the
  budget goes to the summary, and an empty summary is reported instead of
  failing silently.
- **The agent stops advertising `web_fetch` when fetching is off.** The tool
  list no longer claims a fetch capability that is disabled.

## [0.42.0] - 2026-09-11

### Added
- **`/compact` compacts on demand.** Run a compaction pass now, at a natural
  break, instead of waiting to overshoot the threshold. It fires only while
  idle, says why when it cannot (disabled, too short, run in flight, already
  compacting), and clears the thrash notice so a fresh attempt is made.
- **A parallel fan-out returns the prompt to ready.** Dispatching
  `parallel_subagent` (or `/parallel`) no longer leaves the main thread pecking
  at work it cannot read: the harness ends the model's turn the moment the
  batch registers, so the subagents grind under a live prompt and the main only
  re-engages to synthesize when they finish. Any concurrent main-thread work is
  your next input, not a turn the model drags on.
- **Parallel rows show the running tool's outcome.** Each row's leading marker
  is a yellow spinner while the task runs, a green ✓ once it finishes and a red
  ✗ when it fails; alongside, the current tool's icon (`$`, `☰`, `✎`, …) keeps
  its own tone while the call is in flight and flips green or red the moment the
  call lands (nacelle v0.22.0's `ToolDone`).

### Changed
- **nacelle bumped to v0.22.0** for the `ToolDone` parallel-outcome callback,
  replacing the local `replace`-directed checkout this release built on during
  development.
- **Context compacts by itself after a turn that overshoots.** Compaction used
  to wait for the next message to be sent, so a turn that ended with the
  conversation over the threshold sat at that size until you typed again. A
  turn that finishes too heavy now triggers the pass on the spot (when nothing
  is queued), and the report it leaves in the transcript is a readable card:
  before/after tokens, freed, kept share, and what was summarized or masked.
  A message typed while the pass is running is held and sent once the context
  is rebuilt, so it never goes out against a conversation mid-compaction.
- **The light lever runs before the summary.** A pass that is over the
  threshold now first trims oversized tool results and thinking blocks — the
  cheap, backend-free stage — and only escalates to the summarizer call when
  that alone cannot land the conversation back under the threshold. A pass the
  mask clears costs nothing but a byte sweep.
- **A thrash guard stops futile re-compaction.** When several passes in a row
  end with the context still over the threshold — one very large result, usually
  in the kept tail, that eviction cannot clear — the automatic triggers back off
  and a notice suggests `/clear` or reading in chunks, instead of repeating a
  pass that cannot help. A single miss does not disable auto-compaction; a pass
  that lands under resets the count, and `/status` keeps reporting the stuck
  state until it clears or you `/compact` by hand.

## [0.41.1] - 2026-09-11

### Added
- **herdr integration documented.** When run inside herdr, nacelle reports its
  live state and session identity over herdr's socket API; the README now notes
  this and the one-keystroke restore path after a herdr server restart.

### Changed
- **The tool box panes read as a solid block under any font.** The left spine is
  now the half-block (`▌`) — a full-cell bar — instead of a box-drawing vertical
  that renders at a hairline in many monospace fonts, `run_command` output draws
  in the terminal-default foreground rather than a muted grey, and the diff
  recap's `+x -y` join space carries the block background so the footer no
  longer patches a bare gap.

### Fixed
- **Resume by absolute transcript path is pinned.** The herdr reporter hands
  herdr the session's absolute `.jsonl` path; a test now proves that exact form
  (not just a basename) round-trips through `--resume`.

## [0.41.0] - 2026-09-11

### Changed
- **nacelle bumped to v0.21.0** for the native anthropic agent loop. The
  anthropic backend now owns its own tool-running loop and drains the sink on a
  live tick, so a `run_command` on the anthropic path streams one line at a
  time instead of bursting its output once the command finished.

## [0.40.0] - 2026-09-11

### Added
- **The price ticks live as the model writes.** Previously the dollar figure in
  the footer only appeared once a turn ended, because a backend reports `Cost`
  only at turn boundaries. The TUI now derives a cost-per-token rate from each
  finished turn and projects it over the live output estimate, so the `$` moves
  up as tokens stream in and lands on the authoritative `Cost` at the next turn
  boundary. No price is invented for backends that never report a cost
  (anthropic returns tokens only), and nothing shows before the first turn.

### Fixed
- **A running command's streamed output no longer squishes onto one line.**
  nacelle emits each completed output line as its own fragment without a
  trailing newline, and the TUI was concatenating them raw, so the box's line
  split never saw a break and every line ran together. Each fragment is now
  normalised to end in a newline, so the running box grows one row per line and
  the finished box keeps them separate.
- **The tool-block left border is a coherent, filled spine.** The border colour
  was being fed to `lipgloss.Color` as a bare SGR number, which lipgloss reads
  as an ANSI-256 index — so green rendered as teal, red as slate, and the
  running tool's own colour as a desaturated grey, reading as a thin pipe. The
  border and the tool's glyph now share the same true-colour hue (green on
  success, red on failure, the tool's own colour while running), drawn as one
  self-contained block spine.
- **The edit diff recap's removed-line count no longer drops onto the terminal
  background.** The added and removed figures were Foreground-only fragments
  whose reset killed the block background set once on the outer cell, so the
  `-y` figure fell through to the default ground. Each figure now carries the
  block background itself.

## [0.39.0] - 2026-09-10

### Added
- **Edit and command results render as full-width boxes** so they stand out in the transcript: a shared background, a left border coloured by outcome (green on success, red on failure, the tool's own colour while the call runs) and, beneath an edit, a footer recap of `+x -y` counting the added and removed lines. Changed lines sit on green or red tinted grounds. A `run_command`'s raw output is drawn in the same pane — its stdout and stderr now appear in the transcript — capped at 200 lines and stripped of ANSI and carriage returns.
- **A running command's output streams into its box live.** nacelle v0.20.0 lets a tool implement `OutputTool` and emit its output as it is produced, and `run_command` does, one line at a time. The TUI consumes the new `KindToolOutput` events and grows the running box under the tool's line as lines arrive, without duplicating the output when the result arrives carrying the same text.

### Changed
- **The token and context counters tick as the model writes.** The stream reports usage only at a turn boundary, so the output and context counts now carry a live estimate built from the streamed deltas and replace it with the authoritative figure when the turn ends. A detached `/parallel` fan-out's spend joins the footer as it streams too.
- **nacelle bumped to v0.20.0** for `OutputTool` / `KindToolOutput`.

### Fixed
- **The expanded thinking trace streams as one block** instead of rendering a blank row between every line: each completed line was committed as its own widget, and widgets join by a blank row, so a multi-line trace read as one blank line between each.

## [0.38.0] - 2026-09-10

### Added
- **MCP servers configured inline in `~/.nacelle.yml`** — `mcp:` now takes a map of server name to definition (`command`/`args`, `type`/`url`, `env`, `cwd`, `headers`, `disabled`) instead of a list of `.mcp.json` file paths, so a server needs no second file. `-mcp <file>` still loads another client's `.mcp.json` and merges its servers in by name, the flag's winning; the `mcpServers` format keys are the `ServerDef` shape nacelle v0.19.0 exports, which is how the map decodes straight out of YAML.

## [0.37.0] - 2026-09-10

### Changed
- **A completed detached fan-out wakes the main agent to review the work**: when the last subagent finishes while the main run is idle (and at least one task returned a real result), the client sends the results back as a user message and starts a run — each subagent's title and outcome inlined — so the main agent folds the work into its answer instead of leaving it sitting in rows under the prompt. A fan-out that error'd everywhere, or a busy main run, skips the wake-up.
- **`/clear` now drops finished subagent rows while leaving still-running agents untouched**: finished tasks are hidden and a batch is removed once nothing in it runs any more, so a `/clear` starts your session fresh without killing fan-outs that are still grinding.

## [0.36.0] - 2026-09-10

### Changed
- **Parallel rows lead with the spend and end with the timer**: a running task reads `≫ <title>: <tool> <price> <tokens> <clock>`, putting the timer last so the stats line up across the rows. The clock and the token counter now both tick live while a fan-out grinds — the clock for a run launched from an idle prompt too, not just when you type — instead of only at the end.
- **A running subagent's cost shows as it is spent**, not only when it finishes. Relies on nacelle v0.18.0's `LiveUsage` hook, which streams each nested turn's spend per task while the fan-out runs.
- **A task title is always a plain 6-7 word description**: the summarizer is told to never include file paths, URLs or directory names, and both generated and fallback titles strip path- and URL-shaped tokens defensively.

## [0.35.1] - 2026-09-10

### Fixed
- **The input prompt keeps a one-space margin after its prefix** on the first line, so a continuation row no longer sits flush against the prompt start.

### Changed
- **The TUI model takes a `SessionConfig` struct** instead of seven positional arguments, so UI runtime settings can no longer be swapped by accident when building the model.
- **The filet gate now fails on any info-severity finding locally**, matching the CI gate, so a red check can no longer be a green local run.

## [0.35.0] - 2026-09-10

### Changed
- **Parallel rows now show what each subagent is doing right now**: a running task's row reads `≫ <title>: <tool> <clock> <price> <tokens>`, with the live tool name coloured like the same tool in the transcript. Relies on nacelle v0.17.0's live tool-call reporting. A row whose title hasn't landed yet shows a 7-word-capped summary instead of the full delegate prompt, so the fan-out never dumps the whole task text under the prompt.
- **The input prompt's prefix, placeholder and start message are configurable** (`prompt_prefix`, `prompt_placeholder`, `start_message` in `~/.nacelle.yml`). An empty `prompt_prefix` draws no prefix and no continuation indent; `start_message` is a multiline block printed above the version banner. Both follow the default > config precedence of the other UI settings.
- **The system prompt now tells the model a dispatched `parallel_subagent` is the point to end its turn**, so a non-blocking fan-out doesn't hold the main thread open.
- **nacelle bumped to v0.17.0** for the live tool-call hook that drives the per-task tool column.

## [0.34.0] - 2026-09-10

### Changed
- **The input prompt no longer pipes its first line**: the bottom textarea opens bare, so what you type starts at the left margin instead of after a bold `| `. Continuation rows keep their two-space indent. The `| ` prefix stays wherever it already reads as a delivered message — in the transcript and in the queue list — so a typed prompt and a sent message no longer look identical while you are composing.

### Changed
- **The model's own `parallel_subagent` call no longer blocks**: the tool runs detached (nacelle v0.16.0 `Detach` mode). When the model calls it, the tool returns a stub immediately and the fan-out grinds in the background, so the parent's turn — and your main thread — stay free. The call reads as a "started N parallel agents" message, the tool row completes at once instead of sitting "running", and each subagent's result still lands as a titled row under the prompt with its own spend. You can keep chatting while subagents work, whether the fan-out came from `/parallel` or from the model choosing the tool itself.

## [0.32.0] - 2026-09-10

### Changed
- **`/parallel` no longer blocks the main thread**: a typed fan-out is detached. The TUI owns the work — it launches the nested agents itself (via nacelle `DelegateParallel`) instead of asking the parent model to, so the parent run's `busy` flag is never set and the prompt stays live the whole time. "started N parallel agents" appears in the main thread, and each subagent still renders as a titled row under the prompt with a live clock and its own spend. You can now chat while the agents grind. The `parallel_subagent` tool the model calls mid-turn is unchanged and still waits on its result.

## [0.31.0] - 2026-09-10

### Removed
- **No more side runs**: a message typed while the main run is busy queues again instead of being answered by a fresh concurrent agent. The `⇄` side-run feature (v0.30.0) is removed entirely — when the run is busy every message waits and sends when the run settles, matching the behaviour before v0.30.0.

## [0.30.2] - 2026-09-10

### Changed
- **MCP glyph is now a four-pointed star**: the orange MCP marker changed from `❋` to `✻`. The glyph keeps the `mcp` grouping kind and the ANSI 208 orange — an MCP call still never collapses into a write batch — but reads as a star rather than a snowflake.

## [0.30.1] - 2026-09-10

### Fixed
- **A hung summarizer can no longer freeze a session during compaction**: the summary call behind a pass now runs under a 120-second deadline. When the deadline fires, the partial text is dropped and the pass falls back to the mask — which still frees the bulky tool output — instead of holding the session at "compacting" forever.

## [0.30.0] - 2026-09-10

### Added
- **Parallel task rows are titled, not pasted**: each `parallel_subagent` row now shows a short 6-7 word description of the task instead of dumping the full delegate prompt. One extra summarizer call per fan-out (the same backend, no tools) condenses the tasks into titles; a row falls back to its collapsed prompt if the summarizer errors or returns nothing.
- **Messages no longer queue behind a running fan-out**: a question typed while the main run is busy is answered on a fresh side agent that runs concurrently and streams its reply into the transcript, shown under the prompt as a `⇄` row with a live clock. The side run is isolated from the main conversation — it can never corrupt the parent's context or the session's ordering — so the prompt stays usable instead of pinning every keystroke until the fan-out finishes. A command (like `/quit`) still queues as before.

## [0.29.0] - 2026-09-10

### Changed
- **Per-subagent spend instead of one shared total**: a `parallel_subagent` fan-out now reports each task's own token burn (`↑in ↓out`, plus cost) and an elapsed clock on its row, drawn from the per-task `usage` map the result carries (nacelle v0.14.0). The running rows no longer copy the session's combined spend onto every line. Completed rows stay visible under the prompt — showing what each subagent cost and how long it took — until the next send or run end clears them.

### Fixed
- **One compaction cut, not three**: the pass's `evictCut` (how many messages sit in the evicted middle) is computed once in `beginCompaction` and threaded through the pass outcome to `settleCompaction`, `compactApply`, and `applyMaskFallback`. It was being recomputed from the live conversation at settle time, which could disagree with the cut the summarizer summarized across — a latent divergence that a conversation change mid-pass would have turned into a real mask/apply misalignment.

## [0.28.0] - 2026-09-10

### Changed
- **Parallel is the only delegate**: `-subagents` / `subagents:` / `NACELLE_SUBAGENTS` now mount `parallel_subagent` alone — independent tasks fan out to concurrent nested runs, and a one-item task list is the smallest parallel call. The single `subagent` tool is no longer wired in.

### Removed
- **Web search**: the `web_search` tool is gone. The `-search` flag, `NACELLE_SEARCH`, and the `search:` config key no longer exist, and the banner no longer reports `search on`. Delete any `search:` line from `~/.nacelle.yml` — with `KnownFields(true)` a leftover key refuses the client at startup. `web_fetch` (and `fetch:` / `-fetch` / `NACELLE_FETCH`) is unchanged.

### Fixed
- **Turn boundary held apart from the answer it closes**: the muted timing line (`11.159s · 318k tokens · $0.0028`) no longer glues straight under a single-line answer. A blank row now separates a turn boundary from the answer above it, matching the spacing it already had from the tool line below.

## [0.27.0] - 2026-09-09

### Added
- **MCP tools get their own look**: tool calls whose source is an MCP server render with an orange `❋` glyph and group under an `mcp` kind, distinct from every built-in read/write/network/delegate family. The colour holds on success *and* failure — an MCP tool keeps its identity instead of flipping to the green/red outcome colour, so a server's tools stand out in the transcript and in batched groups.

### Changed
- **Queued lines read as the reader's own**: a queued message is prefixed with `| ` and its background bleeds across the full prompt width, matching how the sent question renders, so a held line reads as waiting rather than printed.

## [0.26.2] - 2026-09-09

### Fixed
- **Thinking line stays above the answer it reasoned for**: the collapsed `▶ thought for Xs` line (or, expanded, the last reasoning line) no longer slides under the answer's already-printed paragraphs. The 0.26.1 fix reordered the turn-close flush, which only moved the *final* line of the answer. Because every completed paragraph is committed to the terminal's scrollback the moment it streams, the line is now introduced at the moment the model switches from reasoning to output — the first answer token or tool call — so it is always printed above the answer rather than after it. The reasoning text is still fully retained for `ctrl+t` expansion.

## [0.26.1] - 2026-09-09

### Fixed
- **Thinking line placement**: the collapsed or expanded thinking line now prints above the answer it reasoned for, instead of sliding under the answer text at the end of an agent turn.
- **Queued messages when idle**: a message typed once the agent is back to ready is sent immediately, rather than being held in the queue when no run is coming to drain it.

### Changed
- **Transcript spacing**: thinking lines, tool calls, tool results and turn timings each get a blank row after them, so the transcript reads as separate steps instead of one dense block.

## [0.26.0] - 2026-09-09

### Added
- **Parallel agents under the prompt**: running `parallel_subagent` tasks now render under the input prompt with status and results as they complete.
- **Summarize-and-mask hybrid compaction**: masked context compaction keeps high-signal material by building a summary alongside the mask while compacting history.
- **Compaction on history chunks**: history compacts by summarizing on chunks, cutting context size without losing the thread.

### Fixed
- **Slash menu position**: slash suggestions render below the prompt and preserve the typed text.
- **Slash menu trigger**: the menu opens only at prompt start and on Tab; a mid-sentence slash no longer pops it open on its own.
- **Thinking rendering**: expanded thinking streams through the theme renderer instead of raw markdown.
- **Streamed spacing**: no more blank lines between streamed answer fragments.
- **Queued edits**: a stale queue edit-marker is released so queued lines send, and queued messages use the question style.

### Changed
- **filet config**: harmonised `.filet.yml` to the strict suite standard.

## [0.25.0] - 2026-09-09

### Added
- **Custom providers**: point nacelle at any server that speaks an existing backend's protocol (typically OpenAI-compatible) by setting `base_url` and `api_key` alongside `backend` and `model`. Configured through `NACELLE_PROVIDER_BASE_URL` / `NACELLE_PROVIDER_API_KEY` or `base_url:` / `api_key:` in `~/.nacelle.yml`. The new keys are additive, so existing config files and env vars keep working. Backends `openai`, `openrouter` and `google` already accepted `BaseURL`; the client now forwards them.

### Fixed
- **Input prompt key handling**: Shift+Enter and Ctrl+J no longer insert a new line. Only Alt+Enter now inserts a new line; Enter always sends the message. The TUI now explicitly passes Alt+Enter through to the prompt (returns `false, nil` from `key()`), so only Enter is consumed by the client.
- **Dependency bump**: upgraded `github.com/FacileStudio/nacelle` to `v0.13.0`

### Changed
- **Key handling**: Removed Shift+Enter and Ctrl+J from new line insertion; only Alt+Enter now inserts newlines

## [0.24.1] - 2026-09-09
### Changed
- **Model paste consolidation**: paste handling now lives in `internal/tui/model.go`, removing `internal/tui/paste.go`.

## [0.24.0] - 2026-09-09
### Added
- **Parallel subagent UI**: display running parallel_subagent tasks with status and results in the TUI
- **Strict confinement mode**: new `strict_confinement` toggle controls whether file tools are confined to the working directory
- **Default system prompt**: built-in harness prompt describing available tools and safety rules
- **Pipe-prefixed reader questions**: user questions now render with a bold `|` prefix for better readability

### Changed
- **Dependency bump**: upgraded `github.com/FacileStudio/nacelle` to `v0.12.0`
- **Subagents enabled by default**: parallel delegation is now on out of the box
- **Default search endpoint**: set to `https://furet.facile.studio`
- **Default iterations**: increased from 0 to 5
- **Default compact threshold**: lowered from 100KB to 75KB
- **Default bash mode**: enabled by default
- **Default thinking mode**: enabled by default
- **Cleaner transcript spacing**: removed extra blank lines between transcript entries

### Fixed
- **Tool result spacing**: fixed double blank lines between grouped tool results and failure lines
- **Skill directory walking**: fixed error handling in skill discovery to properly surface filesystem errors
- **Headless cleanup**: improved error handling during agent shutdown in headless mode
- **Test assertions**: updated tests to match new default behaviors

## [0.23.3] - 2026-09-08
### Added
- **Paste sanitization**: added sanitizePaste function to strip terminal control sequences, normalize line endings, and remove bracketed-paste wrappers from pasted text, preventing `[106;5u` and similar artifacts.
- **Paste handling**: route `tea.PasteMsg` through a dedicated handler that inserts only sanitized content into the prompt.
- **Tests**: comprehensive test suite for paste sanitization covering bracketed-paste removal, Windows line ending normalization, embedded control sequence stripping, and unicode preservation.
- **Documentation**: added internal/tui/paste-sanitize-guide.md explaining the problem, solution, and best practices.

## [0.23.2] - 2026-09-08
### Changed
- **Dependency bump**: upgraded `github.com/FacileStudio/nacelle` to `v0.10.1`.
- **Version flag logic moved**: `-version`/`-v`/`--version` handling now lives in `internal/agent/runflags_check.go`, keeping runflags focused on launch configuration.
- **Removed parallel experiment artifacts**: deleted `README_PARALLEL.md` and `parallel-plan.md`, which documented the now-retired parallel agents feature.

## [0.22.3] - 2026-09-08
### Fixed
- **Print flag handling**: combined detection and stripping of `-print` into a single pass, fixing a bug where the flag was detected but not always removed from `os.Args`.
- **Dependency bump**: upgraded `github.com/FacileStudio/nacelle` to `v0.9.5`.
### Changed
- **Removed dead test code**: deleted unused `answeringStub` and imports (`context`, `iter`) from `command_test.go`.

## [0.22.2] - 2026-09-07
### Added
- **Version flag**: `nacelle --version` / `nacelle -v` / `nacelle -version` now prints the version and exits, enabling installer verification.

## [0.22.1] - 2026-09-07
### Refactored
- **Modular package architecture**: decomposed the monolithic root package into 18 internal packages under `internal/` (`agent`, `approval`, `cost`, `diff`, `history`, `layout`, `menu`, `queue`, `sessions`, `settings`, `skills`, `status`, `tasks`, `theme`, `thinking`, `toolview`, `tui`, `usage`), leaving `main.go` as a clean entrypoint.
- **Code health and complexity limits**: refactored routines and split subpackages to satisfy `filet` function count and file length rules across all 134 files.

### Fixed
- **Task plan synchronization**: restored `currentPlan` synchronization on task execution and model resets, preventing lost plan state during subsequent `step_update` calls.
- **Concurrent slice mutation**: eliminated data race in `step_update` by cloning task lists before in-place mutation.
- **Session listing resilience**: guarded session file listing against non-existent project directories with fallback to base sessions directory, and bounds-checked session timestamp formatting.
- **Headless mode input handling**: hardened standard input ingestion against non-EOF errors and empty piped input.
- **Approval gate nil check**: protected against nil dereference when calling `Ask` on unwired approval gates.

## [0.22.0] - 2026-09-04
### Added
- Added `--continue` flag to auto-resume the newest session for the current project on startup
- Added `/resume` command to interactively resume a session from `~/.nacelle/sessions/`
- Added `/sessions` command to list all available sessions for the current project with timestamps
- Model now loads conversation history from session file on resume
- Session project detection (current directory or -root) - handles both absolute and relative paths
- Made the tasks tool more permissive with input preprocessing to handle common model syntax errors

## [0.21.9] - 2026-09-03
### Added
- **`tasks` toggle**: the task planning tool is now configurable through `~/.nacelle.yml` (`tasks:`), `NACELLE_TASKS` environment variable, or `-tasks`/`-no-tasks` flag. Default is on. Follows the standard precedence chain (flag > env > file > default). This completes the set of tool toggles — every tool now has a setting.
- **Task list display**: the plan now shows all steps without truncation or a "... and N more" line, ensuring the full task list is always visible.

## [0.21.7] — 2026-09-03

### Added
- **Turn boundary subtotals**: assistant turns print a muted turn boundary line on `KindTurn` displaying elapsed duration, turn tokens, and cost when greater than zero.
- **Parallel read-only tool guidance**: `defaultSystem` instructs the model to run independent read-only inspection calls in parallel.
- **Tool batch headroom clamping**: grouped tool lines and in-flight status truncate within terminal width to prevent wrapping in the live region.
- **Dynamic compaction window scaling**: automatically scales `compactAt` to 75% of `Capabilities.ContextWindow` when unconfigured and supported by the backend.
- **Dependency bump**: upgraded `github.com/FacileStudio/nacelle` to `v0.8.7`.

## [0.21.6] — 2026-09-03

### Fixed
- **Tool group deduplication across multiple failures**: grouped tool lines now print exactly once on batch completion regardless of distinct failure messages, followed by each failure report.
- **Diff preservation in partially failed tool groups**: completed edits in a group are rendered even when subsequent calls in the batch fail.
- **Group duration accuracy**: grouped tool lines now show the total duration of the batch rather than only the final call's duration.
- **Compaction wire budget isolation**: thinking block trimming no longer consumes wire token budget, and context size is immediately adjusted upon compaction.
- **Conversation preservation across streaming commits**: `flush()` now returns the full answer to `closeTurn()` while printing only the uncommitted tail to the terminal.
- **Session rotation hardening**: extracted rotation to `sessionrotate.go`, set file permissions to `0600`, clean up failed archives, and use millisecond timestamps.
- **Code cleanliness**: removed unused blank identifier, inline comments, and in-body comments.

## [0.21.4] — 2026-09-03

### Fixed
- **Grouped tool lines now print once**: finishing each call in a kind-based batch no longer duplicates the group line. A `finishedCount` counter and `isGroupComplete()` guard ensure the group renders only after the last call lands.
- **`flush()` no longer reprints committed paragraphs**: `committedLen` tracks how much of the streaming answer has already been said, so `flush()` returns only the uncommitted tail and `commitParagraphs()` no longer leaks duplicates.
- **In-flight tool groups render correctly in the live region**: streaming groups show their ticking duration each frame; finished groups are printed once and do not reappear in the live view.

### Refactored
- **`finished()` split into three helpers** — `canPrintTool()`, `trackFailure()`, and the original body — to reduce cognitive complexity and make the group-guard path explicit.

## [0.21.0] — 2026-09-02

### Added

- **`compact_at` setting**: the compaction threshold is now configurable through `~/.nacelle.yml`, the `NACELLE_COMPACT_AT` environment variable, or the `-compact-at` flag. Defaults to 100,000 (absolute tokens, matching the previous hard-coded constant). Setting it to 0 disables compaction entirely. Follows the same precedence chain as every other setting (flag beats env beats file beats default).

## [0.20.0] — 2026-09-02

### Fixed

- **Extra space before the thinking icon**: `fromThinking` text is now rendered
  through the theme style instead of the markdown renderer, which was adding
  leading whitespace to the "▶ thought" line. Streaming always used the theme
  style; the committed line now matches.
- **Shift+Enter now inserts a newline without sending the prompt**: added
  `shift+enter` to the textarea's `InsertNewline` key binding alongside the
  existing `alt+enter` and `ctrl+j`.

## [0.21.0] — 2026-09-02

### Added

- **Session log rotation**: session files rotate at 256 KB. Old files are gzipped (`.gz`) and a new timestamped file starts. A `⚠️ could not write to session log` warning appears in the status line when `write()` fails.
- **Broad tool grouping by kind**: consecutive calls of the same category (read, write, network, delegate) now batch into a single line showing `⏺ 4 commands · cmd1 · cmd2 · …` instead of N identical lines. Single calls render as before. Track D1, E5.
- **`/status` command**: prints session summary — elapsed time, tool counts, tokens, cost, log size, and write-error status.

### Fixed

- **Session log write failures now surfaced**: `sessionLog.write` sets `writeError` on failure; the status line shows a warning so users know logging is broken.

## [0.20.5] — 2026-09-02

### Fixed

- **Compaction now fires during a run, not only between them**. `compact()`
  was only called from `send` (pre-flight CountTokens check) and `settle`
  (between runs), so a long single conversation could grow past the threshold
  and never trim — the context blowup that stopped the agent early. `absorb`
  now compacts on every `KindTurn` and `KindDone`, the same place `sized`
  records the token count.
- **`buildHeadlessAgent` use-after-close**: toolset and MCP resources are now
  closed after the agent finishes streaming, not before it starts.
- **`DeclareFlags`** renamed to `declareFlags` (unexported) — the return type
  `declared` was also unexported, making the function impossible to call from
  outside the package.
- **`HookPointOf`** changed from a `var` to a `func` — same behaviour, no
  mutable package-level state.
- **Doc comments** restored on type aliases in `config.go` — each alias now
  has a one-liner pointing to its `internal/settings` origin.

## [0.17.0] — 2026-09-02

### Added

- **Headless mode** (`-print "prompt"`): builds the agent the same way the TUI
  does, streams text to stdout, no bubbletea. Piped stdin works when `-print` is
  passed alone (`echo "list files" | nacelle`). Track G.
- **Settings chain extracted to `internal/settings/`**: `Config`, `Toggles`,
  `UI`, `Reasoning`, `Web`, `Discovery`, `Sources`, `HookSpec` and the full
  precedence chain (defaults → file → env → flags) now live in their own
  package. Net -779 lines in `package main`. Track G.

## [0.16.1] — 2026-09-02

### Fixed

- **Trailing whitespace clipped** from streaming ansified text, and every
  completed line committed to scrollback rather than whole paragraphs.

## [0.16.0] — 2026-09-02

No changelog recorded — minor fixes between 0.15.0 and 0.16.1.

## [0.15.0] — 2026-09-02

### Added

- **Duplicate-key tool input is refused unconditionally** — `strictObject` now runs
  on every tool call, not only when `-approve-tools` is on. A call with a repeated
  JSON key (`{"command":"ls","command":"rm -rf /"}`) is refused as ambiguous before
  the tool runs, fixing the display/execution split where the transcript showed
  one value while the tool decoded another. Track D3.
- **Identical consecutive tool failures collapse into one line with a count** —
  when the same tool fails with the same error twice in a row, the transcript
  shows `run_command failed 2 times` instead of two identical two-line blocks.
  Track D2.

## [0.14.0] — 2026-09-02

### Added

- **Pre-flight token count compaction**: `send()` now calls `agent.CountTokens()`
  before launching the stream. If the conversation already exceeds `compactAt`,
  compaction runs synchronously before the turn starts — catching growth that
  happened between turns rather than only after a turn finishes. Silent skip
  on backends that don't support `CountTokens` (OpenRouter).
- **Old thinking blocks are now trimmed alongside tool results**: `compact()`
  replaces `nacelle.Reasoning` parts outside the keep window with a
  `[dropped thinking: N bytes...]` placeholder. Thinking is never sent back to
  the API, so the replacement is invisible on the wire; the assistant's
  conclusion (its `Text` part) stays untouched.
- **Checkmark icon and green styling on the "ready" status line**: when the agent
  finishes work and the client is idle, the status line now shows `✓ ready` in
  green (ANSI 2) instead of plain `ready`. A new `palette.ready` style in
  `theme.go` holds the green foreground; `status()` uses it only for the idle
  state, so busy, stopping, and approval prompts keep their own colours.

### Fixed

- **Tool glyph colour was ANSI SGR 1 (bold) and 2 (dim), not 31 (red) and 32 (green)**.
  The `colorGlyph` calls in `finished()` used `"1"` for failure and `"2"` for
  success, which emits `\x1b[1m` (bold) and `\x1b[2m` (dim) instead of the
  intended red and green foreground colours. A failed tool's icon showed bold
  but not red; a successful one showed dim but not green. Both now use the
  correct 3-digit ANSI colour codes, so the icon actually turns green on
  success and red on failure.

## [0.12.0] — 2026-09-02

### Added

- **Blocked and failed task statuses**: a step that hits trouble now marks
  itself `blocked` or `failed` with a `reason`, rather than lying or hanging
  forever. Glyphs: `⊗` (red) for failed, `⊘` (yellow) for blocked.
- **`step_update` mode**: update one step by its 0-based index instead of
  resending the whole plan. Lighter for status changes, fewer tokens per call.
- **Task reminder in settle**: when a run ends normally with unfinished steps,
  a line prints (`☆ tasks: 3/5 steps complete`) and a user-message is injected
  so the model sees it on the next turn.
- **System prompt rule**: "When you lay out work with the tasks tool, keep the
  plan current as you go" — visible to every model on every turn.
- **Markdown rendering for thinking traces**: expanded reasoning now renders
  through glamour like answers do, not just italic grey.

### Changed

- **Tool glyph colour restored after outcome marker**: the old code used
  `\x1b[m` (full reset) after the green/red check/cross, which killed the
  tool colour for the rest of the line. Now the glyph switches back to the
  tool's own hue so `✎ edit_file(view.go) · 12ms` reads as green marker +
  magenta tool line.
- **User question contrast**: `faintFg` moved to ANSI 15 (white) on dark
  terminals, so reader questions and queued messages are white text on the
  dark grey background instead of mid-grey on grey.

### Fixed

- **Thinking trace colour restored**: the third pass of tool-glyph colouring
  (commit ada4611) coloured only the glyph, which fixed tool lines but broke
  the agent's thinking traces — they lost their markdown rendering. Now both
  live-streaming and committed thinking use `m.markdown(text)` like answers do.

## [0.11.0] — 2026-09-02

### Added

- **Mid-line `/command` completion**: tab with more than one match opens the
  dropdown instead of silently picking the first. Re-selecting a command now
  preserves any text before and after the `/command` word.
- **Paragraph streaming**: complete paragraphs are committed to scrollback as
  they arrive, before the model finishes.
- **Token counter shows millions**: output like `2033k cached` now reads
  `2.0M cached`.

### Changed

- **`flush()` reads `fullAnswer`**: the entire conversation (not just the
  visible window) is printed when entering full scrollback.
- **Failed tool lines**: the tool call and its error are rendered as one entry
  instead of two, so there is no blank line between them.
- **Result `⤷ ` has no leading indent**: the two spaces before the arrow were
  removed.
- **Diff output keeps a trailing blank line** for better visual separation
  from the next transcript entry.

## [0.10.0] — 2026-09-01

### Fixed

- **Muted thinking trace** no longer printed to stdout.
- **Tool glyph colours** (green/red) apply only to the check/cross, not the
  whole line.
- **Queued/question backgrounds** are no longer muted.
- **Removed dead group functions** and formatted with gofmt.
- **Fixed embedded field selectors** for QF1008 compliance.

## [0.9.1] — 2026-09-01

### Fixed

- **Tool icon color only applies to the glyph, not the whole line**
  (regression in 0.9.0 where the entire tool line turned green/red).
- **Queued messages now have a background**, matching the prompt styling for
  better visual distinction from the transcript region.
- **Removed unused speaker types** (`fromToolOk`, `fromToolFail`) in favour of
  embedding ANSI color codes in the line string itself.

## [0.9.0] — 2026-09-01

### Fixed

## [0.8.0] — 2026-09-01

### Added

- **Mid-message `/command` suggestions**: typing `/` anywhere in the prompt
  now opens the dropdown menu, not only at the start. The word after the last
  slash is the filter, and selecting a command replaces only that word,
  leaving the rest of the sentence intact.
- **Prompt highlighting**: the prompt text turns cyan while a `/command` word
  is active, giving visual feedback that the menu is open.
- **Colored task icons**: `✓` (green) for completed, `▶` (cyan) for
  in_progress, `○` (muted) for pending. Characters are text-safe unicode, not
  emoji.
- **Scroll-aware dropdown**: when the match list exceeds the visible height,
  selection scrolls the window so the highlighted row stays on screen.
- **Tool grouping**: identical consecutive tool calls fold into a single
  transcript row labelled `(N×)`, reducing noise when the model reads the
  same file or searches the same pattern repeatedly. Toggle with
  `group_tools` in `~/.nacelle.yml`.

### Changed

- **Task list reordered** above the status line, so the loading/status row
  sits below the plan rather than above it.
- **Go toolchain** bumped to 1.26 across `mise.toml` and `go.mod`.

### Fixed

- **The `running` map is gone**, replaced by the group-based tracking. The old
  map was keyed by call ID and never cleaned up for discarded calls, and its
  iteration order was deliberately random, which made `stranded()` sort the
  keys first to keep the transcript stable. The group list is order-preserving
  by construction, and a discarded group is simply not added.

## [0.7.0] — 2026-09-01

### Added

- **`google` and `openai` backends**: `-backend google` runs on Google Gemini
  (`GEMINI_API_KEY` / `GOOGLE_API_KEY`, defaulting to `gemini-3.7-flash`), and `-backend openai`
  runs on OpenAI (`OPENAI_API_KEY`, defaulting to `gpt-5.4`). Both support streaming, tool
  calling, and reasoning.

## [0.6.1] — 2026-08-28

### Fixed

- **`scripts/check.sh` ran golangci-lint against a toolchain it had not chosen.** The build, vet
  and test passes go through `$GO`, resolved from `GOROOT` so they honour the version this
  repository pins. The lint pass called `golangci-lint` bare, and golangci-lint type-checks with
  whatever `go` it finds on `PATH`. Where a newer one sat there, a Homebrew install ahead of the
  pinned one being the ordinary way, the pass died mid-run on `file requires newer Go version
  go1.27 (application built with go1.26)` under two hundred lines of goroutine dump. It now runs
  under the same toolchain as everything else.

  The failure mode is worse than a broken lint pass, which is why it is worth the paragraph:
  `lefthook` runs this script on `pre-push` and resets the environment, so the gate could not be
  repaired from the calling shell, and the message pointed at nothing the reader could act on. A
  gate that fails for an unactionable reason is a gate that gets pushed past with `--no-verify`,
  which is what happened to 0.6.0.

- **The 0.6.0 note on batching overstated what it saves, and is corrected in place.** It said the
  harness already ran a turn's calls concurrently. That holds on `anthropic`, whose backend hands
  the turn to the SDK's runner; it does not hold on `openrouter`, whose backend runs a turn's
  calls in order. The rule is still worth having on both, because the model round trip is the
  expensive part, but the entry now says which backend does which.

- **The 0.6.0 note on `max_iterations` described the risk without naming the case that carries
  it**, and is likewise corrected in place. The SDK conditions "zero means no cap" on every tool
  being read-only and cheap. `bash: true` mounts a real shell, which is neither, and that is the
  configuration the no-cap default is least safe under.

## [0.6.0] — 2026-08-28

### Added

- **The model is told to batch independent tool calls.** When several calls need nothing from one
  another, it makes them in a single turn rather than one after another, and waits only where a
  later call needs an earlier one's output. What that saves depends on the backend: `anthropic`
  hands a turn's calls to the SDK's own runner, which executes them together, while `openrouter`
  runs them in order. Batching earns its place on both, because the round trip to the model costs
  more than the tools do. The rule lives in the environment preamble rather than the default
  system prompt, because
  `-system` replaces that prompt outright and this is the one piece of working advice that has to
  survive a custom persona.

### Changed

- **`max_iterations` now defaults to `0`, which is no cap.** The old default of 40 ended a long
  run with `Stop: StopIterations` partway through the work, and 40 was never a number chosen for
  a particular task; it was a guess that any genuine multi-step job outgrows. A run now continues
  until the model stops asking for another turn. Put a ceiling back with `max_iterations` in
  `~/.nacelle.yml`, the `-max-iterations` flag or `NACELLE_MAX_ITERATIONS`, all unchanged.

  The trade is worth stating plainly. The SDK conditions "zero means no cap" on every tool being
  read-only and cheap, and `bash: true` mounts `run_command`, which is neither: with no cap, a
  model looping on a failing shell command loops until you interrupt it. Keep a cap where the
  shell is mounted, where the tools are expensive, or where the run is unattended.

### Removed

- **`mycelium: true` in `~/.nacelle.yml` is now a startup error. Delete the key.** The
  `-mycelium` flag, the `NACELLE_MYCELIUM` variable and the `mycelium:` setting are gone, along
  with the `list_flows`, `run_flow` and `search_memory` tools they mounted. This client reads
  `~/.nacelle.yml` with `KnownFields(true)`, which refuses a key it does not recognise rather
  than ignoring it, so a config that still carries the line will not load until the line goes.
  That strictness is deliberate and it is why this is called out here.

  Nothing is lost. `mycelium mcp` serves the same three tools over stdio, so add mycelium's
  server to a file in your `mcp:` list and they come back through the path every other MCP
  server already uses — gated by `-approve-tools` like any other tool, and working on either
  backend. `mycelium install` writes that `mcpServers` entry for you.

  The reason for the change is layering, not the tools. The switch mounted
  `nacelle/tools.Mycelium()`, which made a general-purpose Go SDK depend on one company's
  binary; the SDK dropped the package in the same round.

- Reading mycelium's usage feed is **not** affected. Each finished turn is still appended to
  `~/.mycelium/events/nacelle/`, gated on that directory existing rather than on any setting,
  and the dashboard still shows a live session. That is a separate integration from the tools.

## [0.5.0] — 2026-08-24

### Changed

- A delegate may now use the tools it was given. `-subagents` handed the nested
  run the parent's tool set and then refused every call it made, which is the
  library's default when the caller supplies no policy, so a delegate asked to
  search wide or read a log could do neither and answered from its task
  description alone. It now inherits the parent's own policy: where the parent
  runs every call unasked, so does the delegate, and where `-approve-tools` is
  on, the delegate's calls reach the same prompt. Two things worth knowing
  before turning it on. The prompt names the tool and cannot say that a
  delegate asked for it, and allowing a tool for the session allows it for the
  parent too, because the allow-list is keyed by name.
- `-approve-tools` no longer asks about the plan. The `tasks` tool writes
  nothing outside this process, and everything it says is drawn on screen the
  moment it is recorded, so the question bought nothing and cost a keypress per
  step of every plan.

### Fixed

- The plan no longer outlives the session it belonged to. `/clear` reset the
  conversation, the running total and the reasoning clock but not the steps, so
  a cleared screen opened with the last session's plan still drawn above the
  prompt, and a model with no memory of it never wrote over it. The rows it
  held go back to the live region with it.
- A delegate can no longer replace the plan on screen. The nested run inherits
  the parent's tools minus only the sub-agent tool, so it inherited `tasks` as
  well: a delegate that reported its own steps overwrote the plan the parent
  was working to, and nothing corrected the screen when the delegation ended.
  The tool is now mounted after the delegate has taken its copy, which is where
  anything that draws belongs. Only reachable with `-subagents` on.

## [0.4.3] — 2026-08-24

### Fixed

- A queued message being edited could still be sent. The offset naming it
  counts from the end of the queue, which survives the queue draining from the
  front — but skipping the edited line meant lines behind it were drained too,
  and each of those shortened the queue without moving the edited line closer
  to the end. The offset then pointed past the queue, nothing looked like it
  was being edited, and the next line out was the one still being rewritten.
  The reader's own edit then arrived behind it as a second, near-identical
  message. Found by review, not by use.
- The banner no longer wraps on terminals wider than 80 columns. It is painted
  before any window size has been reported, so it was held to the 80 the model
  starts at and put `· bash on` on a line of its own. What the client says
  about itself is now left for the terminal to wrap, like everything else.
- Session filenames are UTC and carry no punctuation. Local time is not in
  chronological order across a DST boundary or a flight, and sorting the names
  is the only index the directory has; RFC3339 also spells the offset with a
  colon, which Windows and SMB refuse and the Finder renders as a slash.

### Changed

- The reader's own questions carry a background again, not just bold. An
  answer is full of bold — every heading and emphasised phrase the model
  writes — so the one bold line meaning "you said this" was competing with a
  page of them, and scrolling back for your own question meant reading rather
  than glancing.
- The client separates clauses with `·` and no dashes, so the empty-run report
  reads `no answer · nothing billed · try another model`.

## [0.4.2] — 2026-08-24

### Added

- The banner names the client and its version — `nacelle 0.4.2 · openrouter ·
  <model>`. It is the one thing on screen a bug report needs and nothing else
  supplied, on a line that was already there.

### Changed

- The empty-run report is one line and says what to do rather than whose fault
  it is. The first version explained that a model refusing tool definitions is
  the usual cause: true, but it wrapped onto a second row and read as the
  client making an excuse for itself. The diagnosis lives in the source now.

## [0.4.1] — 2026-08-24

### Fixed

- A run that produced nothing now says so instead of returning to `ready`
  under a transcript holding only the question. A stream that yields a turn
  and a done with no text between them is well-formed — nothing errored and
  nothing was refused — so there was no ending to report and the client
  looked like it had ignored the question. When the turn billed no tokens
  either, the report says so: that is a request the provider dropped before
  running it, which is what a model that will not accept tool definitions
  does. Measured against `openrouter/stealth-ox-alpha`, which returns exactly
  this for any request carrying a tool, and answers normally without one.
- The launch banner is visible again without scrolling up. It was handed to
  `tea.Println`, which does not append: it makes room by scrolling the screen
  and inserting above the frame, so on a freshly cleared terminal — where the
  frame stands on the first row — there was nothing above it to insert into
  and the banner went straight into the scrollback. Which backend and model
  are about to be billed now prints before the program takes the screen.
- A queued message pulled into the prompt for editing no longer appears twice.
  It is drawn in the prompt, and was still listed above in the state it was
  being rewritten out of, which read as the edit having failed. It is also no
  longer delivered while it is being edited: a run settling mid-edit used to
  send the wording the reader had already decided was wrong.

## [0.4.0] — 2026-08-24

### Added

- Queued messages can be edited. Up from the prompt now walks the lines still
  waiting to be sent before it walks the questions already sent, and enter puts
  the edited line back where it came from instead of adding a second copy. A
  line that goes out while it is being edited becomes a new message rather than
  overwriting whichever line has taken its place.
- Esc hands the session to the queue. It still stops the answer being written,
  but what was typed behind it now starts immediately instead of being thrown
  away — esc means "not this one, move on", and ctrl+c keeps meaning stop.
- Session transcripts under `~/.nacelle/sessions`, one JSONL file per run.
  Questions, answers, and the name and duration of each tool call. Reasoning,
  tool arguments, tool output and file diffs are never written: the only
  reliable redaction is not collecting it. Directory `0700`, file `0600`.
- A task list the model keeps through a `tasks` tool, drawn under the status
  line, for a job big enough to need splitting into steps.
- Each finished turn is appended to mycelium's event feed when mycelium is
  installed, so a session shows up in its dashboard while it is still running.

### Fixed

- Pressing Up twice with one question in the history crashed the client. The
  walk ran past the start of the list and indexed it at -1.
- Muted text is readable again. Every dimmed style used ANSI 8, which a dark
  scheme is free to park on the background, so "dimmed" came out invisible on
  the terminals that move it. Greys now come from the 256-colour ramp, which is
  not themeable, and follow the terminal's own background.
- The status row says which phase a run is in: the spinner, the phrase and the
  clock share one colour, cyan while the model is being waited on and the
  tool's own colour once something runs. A tool with no glyph of its own — every
  MCP tool — was drawn with no colour at all.
- Truncating a styled line charged a cell for every character of an escape
  sequence, so a coloured status line was cut a dozen cells early and cut
  mid-sequence, leaking the colour into everything printed after it.

## [0.3.0] — 2026-08-24

### Added

- Prompt history: Up recalls sent questions newest-first, Down walks back
  forward and restores the draft being written; from a wrapped question's
  later rows, Up still moves the caret.
- A two-row status bar — what is happening (spinner, phrase, run clock) on
  top, what the session has spent underneath.
- `/cost` says what the session has spent so far, on demand.
- The command menu marks its selected row with an arrow.

### Changed

- The palette is ANSI indices now, so every shade follows the terminal's own
  scheme; the question entry is bold instead of carrying a background. The
  status spinner takes its colour from the phase — dim while thinking,
  tool-blue once something runs.
- The suggestion menu draws below the prompt, which no longer moves the
  input line when the list changes; the prompt gained a gutter marker and a
  blank row of its own, and one empty row follows the launch banner.
- The core SDK is pinned at v0.4.1, up from v0.4.0: sub-agent delegation
  (`NewSubAgentTool`) arrives in core, opt-in here via `subagents:`.

### Added

- File edits draw a git-style diff under the tool's one-line report: removals in the
  terminal's red, additions in its green, with three lines of context around each block and a
  cap of four hundred lines. `edit_file` diffs from its own old/new arguments; `write_file`
  from what the file held when the call was seen against the contents it wrote. Off with
  `diffs: false` (`NACELLE_DIFFS`, `-diffs`), which restores the bare one-line report.

- `-subagents` / `subagents:` / `NACELLE_SUBAGENTS`, off by default. When on, the model
  gets a `subagent` tool that delegates a self-contained task to a fresh nested run — its
  own context window, the same backend and tools — and only the delegate's final answer
  comes back. The nested run cannot delegate further and denies every tool call by
  default; it runs unattended or not at all. A delegate's token spend lands in the
  session's totals as it is spent.

### Fixed

- The gofmt pass in `scripts/check.sh` handed git's file list to `xargs` as
  whitespace-delimited text. Git passes spaces in filenames through unquoted, so a path
  with a space was split and half of it formatted; the list is null-delimited now (`-z`
  feeding `xargs -0`), which also stops GNU `xargs` running gofmt once on an empty list
  with stdin attached.
- The status line's waiting phrase is bucketed from when this run began rather than from
  the wall clock, so every wait opens on the first phrase instead of wherever the epoch
  happened to be.

## [0.2.1] — 2026-08-24

### Changed

- Homebrew installs from a cask rather than a formula. `brew install
  FacileStudio/tap/nacelle` is unchanged; what moves is the tap file, from
  `Formula/nacelle.rb` to `Casks/nacelle.rb`. GoReleaser soft-deprecated `brews` in v2.10 and
  hard-deprecated it in v2.16, and release CI runs 2.17.1, so the previous config would have
  failed its own check on the next tag. A cask download carries the quarantine attribute
  where a formula's did not, so a `postflight` hook strips it — without that, the first run
  of the unsigned binary dies on "the developer cannot be verified".
- The core SDK is pinned at v0.4.0, up from v0.3.1. The only change between the two is the
  split that created this repository, so nothing here had to adapt.

## [0.2.0] — 2026-08-24

### Added

- `ctrl+t` expands a turn's reasoning, and keeps showing it in full until pressed again. It
  costs the prompt its `transpose-character-backward` binding.

### Changed

- A tool call reads as `read_file(view.go) · 12ms` instead of the tool name followed by its
  raw JSON input. The argument is picked by key (`path`, `file_path`, `file`, `command`,
  `pattern`, `query`, `url`, `name`), falling back to the sole key of a one-argument call.
- A tool that succeeds is one line carrying its duration, not a call line plus a
  `done in 12ms` line under it. Failures keep both lines. The call line is held until the
  result arrives, which the status line already covers by naming the running tool; a run that
  ends with calls in flight still prints them, without a duration.
- Reasoning collapses to `· thought for 4.2s`, with the `ctrl+t` hint shown once per session.

### Fixed

- The thinking duration measured the whole turn. A turn is committed at its end, which is
  after the answer has streamed and after any tool the model called has run, so 0.6s of
  reasoning followed by a 0.6s answer printed `thought for 1.2s`. The clock now stops at the
  first answer delta or tool call.
- `/clear` left the retained reasoning behind, so `ctrl+t` reprinted the thinking from the
  session that was just cleared.
- A run that ended mid-tool said the held call line above the sentence that announced it.
- The build binary `nacelle-tui` was tracked rather than ignored; plain `go build` writes it
  and `.gitignore` only listed the old `nacelle` name. Untracked here, though the objects
  already in history stay there.

### Security

- Tool-call input carrying a repeated key is refused rather than summarised.
  `encoding/json` silently keeps the last value, so `{"command":"ls","command":"rm -rf /"}`
  could be shown as `run_command(ls)` over a call that ran something else. With
  `-approve-tools` on, such a call is denied before any prompt is drawn and before the
  session allow-list is consulted, since allow-for-session is permission for a tool granted
  against a legible call, not a standing waiver on whatever input arrives afterwards.

## [0.1.0] — 2026-08-23

### Added

- First tagged release of the terminal client, extracted from
  [FacileStudio/nacelle](https://github.com/FacileStudio/nacelle) where it lived as `tui/`.
  The module is now `github.com/FacileStudio/nacelle-tui` and pins core v0.3.1.
- `-version` flag printing exactly `nacelle <semver>`, stamped by goreleaser's ldflags into
  `main.version`.
- Distribution per CLI-STANDARD §5/§9: GoReleaser archives for darwin/linux amd64+arm64,
  Homebrew formula in `FacileStudio/tap`, an `install.sh` shim, and a `nacelle` entry in the
  `facile` catalog.

[0.20.3]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.20.3
[0.20.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.20.0
[0.18.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.18.0
[0.17.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.17.0
[0.15.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.15.0
[0.16.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.16.0
[0.16.1]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.16.1
[0.14.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.14.0
[0.12.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.12.0
[0.11.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.11.0
[0.10.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.10.0
[0.9.1]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.9.1
[0.9.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.9.0
[0.8.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.8.0
[0.7.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.7.0
[0.6.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.6.0
[0.5.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.5.0
[0.4.3]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.4.3
[0.4.2]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.4.2
[0.4.1]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.4.1
[0.4.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.4.0
[0.3.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.3.0
[0.2.1]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.2.1
[0.2.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.2.0
[0.1.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.1.0
[0.25.0]: https://github.com/FacileStudio/nacelle-tui/releases/tag/v0.25.0
