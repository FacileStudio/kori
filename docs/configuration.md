# kori — Configuration

Every field the core `Config` reads, then the client's own settings layer — a separate thing,
with its own precedence order — and the traps in each.

## `kori.Config`

The core reads nothing from the environment or from disk. Every field is passed in the struct
literal a consumer builds.

| Field | Default | What it does |
|---|---|---|
| `Backend` | — required | The model this agent runs on. No default: a package that picks one for you is a package that hides the most consequential decision in the configuration |
| `System` | — required | The system prompt. Empty is refused rather than defaulted — an agent with no system prompt is a general-purpose assistant wearing a product's name |
| `Thinking.Effort` | the backend's own default | `none`, `minimal`, `low`, `medium`, `high`, `xhigh`, `max`. `none` is the only value that asks for no reasoning, and a model whose reasoning is mandatory answers it with a 400 rather than ignoring it (measured on `stealth/ox-alpha`, 2026-08-23). Empty hands the decision to the provider, which on a model whose reasoning is mandatory means that model's own maximum. Not validated against the model: OpenRouter clamps a level a model does not advertise rather than refusing it |
| `Thinking.Budget` | `0` (no ceiling from here) | Reasoning-token ceiling for one turn. The other spelling of `Effort`. Anthropic takes both at once; OpenRouter answers the pair with a 400 (`Only one of reasoning.effort and reasoning.max_tokens can be specified`, measured 2026-08-23) and its backend resolves that by letting the budget win, since the gateway defines each level as a percentage of the budget anyway. Refused at construction when it cannot work: at or above `MaxTokens` it leaves the turn nothing to answer with, and below a backend's `Capabilities.MinBudget` (1024 on `anthropic`) the API rejects it |
| `Thinking.Show` | `false` | Stream the model's reasoning as `KindThinking`. Off by default, matching the APIs: the raw chain of thought is never returned and a readable summary is opt-in. It changes what a consumer is shown and nothing else. The reasoning always travels back over the wire, because that is what the tool loop replays to keep the model's train of thought intact across a tool call |
| `MaxTokens` | `DefaultMaxTokens` (32000) | Per-turn output ceiling. Generous on purpose — every request is streamed, so a large ceiling costs nothing in latency, while a small one truncates an answer mid-sentence and buys a retry |
| `MaxIterations` | `0` (no cap) | How many times the model may be asked. A cap is only safe when every tool is read-only and cheap; reaching it ends the run with `Stop: StopIterations`, not a failure |
| `Tools` | `nil` | Tools the model may call in this process |
| `MCP` | `nil` | MCP servers the model may call tools on. Only `anthropic` can reach them — see [architecture.md](architecture.md#capabilities) |
| `Logger` | `slog.Default()` | Receives the few things worth recording that are not events — retry attempts, mainly |

`Backend`, `Thinking` and `MCP` are all checked against the chosen backend's
`Capabilities` at construction; a config asking for something the backend cannot do is refused
with an `*Unsupported` error rather than silently running with less.

## `anthropic.Config`

| Field | Default | What it does |
|---|---|---|
| `Client` | built from the environment | Set to share a client, point at a proxy, or hand a test a stub transport |
| `Model` | `DefaultModel` (`claude-opus-5`) | — |

## `openrouter.Config`

| Field | Default | What it does |
|---|---|---|
| `APIKey` | `OPENROUTER_API_KEY` | — |
| `Model` | — required | An OpenRouter model slug, e.g. `anthropic/claude-opus-5`. Required: OpenRouter fronts hundreds of models and a default would be this package choosing the one thing the caller came here to choose |
| `BaseURL` | `DefaultBaseURL` | Point at a compatible gateway or a recording proxy in tests |
| `Referer`, `Title` | — | Attribution only — `HTTP-Referer` and `X-OpenRouter-Title`. Neither affects the answer; without `Referer` the usage simply does not appear against an app |
| `Provider` | `nil` | OpenRouter's provider-routing object, passed through untouched. Worth setting `require_parameters: true` when tool calling matters — it keeps the request away from providers that would drop the tool schema |
| `Options` | `nil` | Extra request options for the underlying client |

## `google.Config`

| Field | Default | What it does |
|---|---|---|
| `APIKey` | `GEMINI_API_KEY` or `GOOGLE_API_KEY` | — |
| `Model` | `DefaultModel` (`gemini-3.7-flash`) | The Gemini model to run |
| `BaseURL` | `DefaultBaseURL` (`https://generativelanguage.googleapis.com/v1beta/openai`) | Point at a compatible proxy or gateway |
| `Options` | `nil` | Extra request options for the underlying client |

## `openai.Config`

| Field | Default | What it does |
|---|---|---|
| `APIKey` | `OPENAI_API_KEY` | — |
| `Model` | `DefaultModel` (`gpt-5.4`) | The OpenAI model to run |
| `BaseURL` | `DefaultBaseURL` (`https://api.openai.com/v1`) | Point at a compatible gateway or proxy in tests |
| `Organization` | `""` | Optional `OpenAI-Organization` header |
| `Project` | `""` | Optional `OpenAI-Project` header |
| `Options` | `nil` | Extra request options for the underlying client |

## `tools.Config`

| Field | Default | What it does |
|---|---|---|
| `Root` | — required | The directory every tool is confined to, through `os.Root` |
| `AllowBash` | `false` | Mounts the command tool. Not a limit that can be tuned into safety — running commands is either something this agent may do or it is not |
| `CommandEnv` | `PATH` + `HOME` only | The process environment is never inherited — it is where a service keeps the credentials a model must not read |
| `CommandTimeout` | `DefaultCommandTimeout` (2 min) | A command outliving it is killed with its children |
| `MaxOutputBytes` | `DefaultMaxOutputBytes` (64 KiB) | Caps what one tool call may return — output stays in the context window for the rest of the conversation |
| `MaxReadBytes` | `DefaultMaxReadBytes` (48 KiB) | Caps a single file read, below the output cap so a read that hit the limit still leaves room for the notice saying so |

## The TUI's settings layer

The client's settings layer is a separate settings problem from the core, and deliberately does not share a
mechanism with it: **the library must never read configuration from disk or environment.** A
`nacelle` package that read `~/.kori.yml` would let a file on a *different* machine's disk
change a headless consumer's behaviour, the way Bubble Tea's own config never reaches past the
program that imports it.

### Precedence

**Flag beats environment beats file beats default**, resolved in one function
(`settings.Load()` in `internal/settings/settings.go`) and nowhere else. The suite has already paid for the
alternative once: a CLI that read its environment inside one code branch and its file inside
another turned what its README called "overrides" into two mutually exclusive modes nobody
could tell apart.

| Layer | Source | Notes |
|---|---|---|
| Flags | `-backend`, `-model`, `-effort`, `-root`, `-system-prompt`, `-bash`, `-thinking`, `-project-context`, `-skills`, `-trust-skills`, `-skill-dir`, `-mcp`, `-fetch`, `-approve-tools`, `-diffs`, `-show-hooks`, `-show-hook-output`, `-max-iterations`, `-compact-at`, `-max-concurrency`, `-max-parallel-agents`, `-tasks`, `-continue`, `-resume`, `-gates-file`, `-no-config` | Only flags actually **typed** are collected, via `flag.Visit` — Go's `flag` package cannot otherwise tell a flag left alone from one passed its own default value. `-skill-dir` and `-mcp` are repeatable (`-mcp a.json -mcp b.json`); every other flag keeps only its last occurrence. `-resume` names one session by id or file path and, when given, beats `-continue`. `-no-config` skips `~/.kori.yml` entirely: defaults plus environment plus flags. An invalid file gets a coloured report and one prompt — yes boots with defaults, no exits with the documentation link |
| Environment | `KORI_BACKEND`, `KORI_MODEL`, `KORI_PROVIDER_BASE_URL`, `KORI_PROVIDER_API_KEY`, `KORI_EFFORT`, `KORI_REASONING_BUDGET`, `KORI_ROOT`, `KORI_SYSTEM_PROMPT`, `KORI_BASH`, `KORI_THINKING`, `KORI_PROJECT_CONTEXT`, `KORI_SKILLS`, `KORI_TRUST_SKILLS`, `KORI_SKILL_DIRS`, `KORI_APPROVE_TOOLS`, `KORI_DIFFS`, `KORI_SHOW_HOOKS`, `KORI_SHOW_HOOK_OUTPUT`, `KORI_MAX_ITERATIONS`, `KORI_COMPACT_AT`, `KORI_MAX_CONCURRENCY`, `KORI_MAX_PARALLEL_AGENTS`, `KORI_FETCH`, `KORI_TASKS` | A misspelt boolean (`KORI_BASH=yez`) is treated as unmentioned, not as `false`, and falls through to the layer below. `KORI_SKILL_DIRS` is colon-separated, the same convention `PATH` itself uses for a list of directories. `KORI_PROVIDER_BASE_URL` and `KORI_PROVIDER_API_KEY` belong to the active provider — see [Custom providers](#custom-providers) |
| File | `~/.kori.yml` | Preferences only, **no credentials** — those already have two homes: the environment, and the Anthropic SDK's own profile. `KnownFields(true)`: an unrecognised key (`max_iteration:`, one letter short) is refused rather than silently ignored |
| Defaults | — | `provider.backend: anthropic`, `root: .`, `tools.run_command: true`, `reasoning.thinking: true`, `discovery.project_context: true`, `discovery.skills: true`, `discovery.trust_skills: false`, `discovery.trust_hooks: false`, `sources.skill_dirs: []`, `sources.mcp: {}`, `security.approve_tools: false`, `security.deny_elevation: true`, `ui.diffs: true`, `ui.show_hooks: true`, `ui.show_hook_output: true`, `limits.max_iterations: 5`, `limits.compact_at: 75000` (absolute tokens), `limits.max_concurrency: 16`, `limits.max_parallel_agents: 16`, `tools.web_fetch: true`, `tools.tasks: true`, `tools.parallel_subagent: true`, `ui.rendering_mode: inline`, `ui.group_tools: true`, `ui.show_thinking: true` |

`project_context` and `skills` default **on**, unlike `bash`: each fails soft to nothing when
there is nothing to find — no `AGENTS.md`/`CLAUDE.md` anywhere above `root`, no
`~/.agents/skills/` — so a machine without either is no worse off for asking. `trust_skills` and `approve_tools` both default **off**, and for the same
underlying reason: a project's own `.agents/skills/` can carry instructions to run arbitrary
scripts, and asking before every tool call is a decision that changes how the client feels to
use — blanket-trusting a directory, or blanket-interrupting every call, are both decisions only
the person running this should opt into, not defaults sprung on them. See
[Context and skills](#context-and-skills) and
[Tool approval](#tool-approval) below.

### `~/.kori.yml`

The file is grouped: each family of settings lives under its own key, the same
families the code groups them into. The old flat layout (every key at the top
level) was replaced by these groups in 0.44.0 — a flat key is now a parse
error, because `KnownFields(true)` refuses anything unknown.

```yaml
provider:
  backend: anthropic
  model: claude-opus-5
  # base_url: and api_key: point at a custom endpoint — see "Custom providers".
  # leave them out to use the vendor's own API and key resolution.
  # base_url: http://localhost:3001/v1
  # api_key: sk-your-unified-key
reasoning:
  effort: high
  thinking: true
  budget: 8192
limits:
  compact_at: 75000
  max_iterations: 5
  max_concurrency: 16
  max_parallel_agents: 16
session:
  root: .
  system_prompt: You are a terminal coding assistant.
  continue: false
tools:
  run_command: true
  web_fetch: true
  tasks: true
  parallel_subagent: true

security:
  approve_tools: false
  path_isolation: false
  deny_elevation: true
  env_isolation: false

discovery:
  project_context: true
  skills: true
  trust_skills: false
  trust_hooks: false
ui:
  prompt_placeholder: 'Ask something. Esc stops a run, ctrl+c stops or quits, ctrl+\ forces it.'
  rendering_mode: inline
  group_tools: true
  show_thinking: true
  diffs: true
  transparent_blocks: false
  show_hooks: true
  show_hook_output: true
  # cron_list_json: false
  # start_message: |-
  #   Welcome to kori.
  #     _   _
  #    | | | |
  #    | |_| |
  #    |_____|
sources:
  skill_dirs:
    - ~/.claude/skills
  mcp:
    mycelium:
      command: mycelium
      args: [mcp]
hooks: []
# Gates are deterministic checks run after every edit: the edited path is
# appended to the command's arguments, findings are injected into the edit
# result. No gates key keeps the built-in filet check.
gates: []
# gates:
#   - name: filet
#     command: [filet, check]
#     timeout_secs: 30
#   - name: prettier
#     command: [prettier, --write]
#     format: true
```

Old flat key → new home, for migrating a pre-0.44 file:

| Old key | New location |
|---|---|
| `backend`, `model`, `base_url`, `api_key` | `provider:` |
| `max_iterations`, `compact_at`, `max_concurrency`, `max_parallel_agents` | `limits:` |
| `bash` | `tools.run_command` |
| `subagents` | `tools.parallel_subagent` |
| `fetch` | `tools.web_fetch` |
| `tasks` | `tools.tasks` |
| `approve_tools`, `path_isolation`, `deny_elevation`, `env_isolation` | `security:` |
| `effort`, `thinking`, `reasoning_budget` (now `budget`) | `reasoning:` |
| `project_context`, `skills`, `trust_skills`, `trust_hooks` | `discovery:` |
| `continue` | `session:` |
| `resume` | dropped from the file — `-resume` on the command line only |
| `mode` (now `rendering_mode`), `json` (now `cron_list_json`), `group_tools`, `show_thinking`, `diffs`, `prompt_placeholder`, `start_message`, `transparent_blocks` | `ui:` |
| `skill_dirs`, `mcp` | `sources:` |

Every field is optional. A missing file is not an error — on first boot
kori writes this file itself with every default explicit, and you edit from
there; deleting it regenerates it. But an unreadable or malformed one is: a
unreadable or malformed one is: a config silently ignored is worse than no config, because the
setting carefully written is simply not in effect and nothing says so.

Scheduled jobs are no longer part of this file at all — they live one YAML file
per job in `~/.kori/jobs/`. See [Cron jobs](#cron-jobs) for the folder, its
trust gate and the per-job overrides.

`prompt_placeholder` is the ghost text shown while the prompt is empty. The prompt has no prefix:
the first row opens at a single margin space, and wrapped rows of a long question hang under it.

`start_message` is printed as the first thing on launch, above the version banner. It is a plain
string that may span lines, so use a `|`-literal (or `|-` to drop the trailing newline) to put a
welcome block or an ascii banner there. Empty (the default) prints nothing.

**No per-project `./.kori.yml` yet.** A second precedence layer before the first has real
users is a layer nobody has asked for the shape of.

### Custom providers

A custom provider points kori at an endpoint that is not one of the four vendor APIs, using an
existing backend's protocol when the endpoint speaks it. It is four fields with the same precedence
as everything else: `backend` and `model` (the `KORI_BACKEND` / `KORI_MODEL` behind them), plus
`base_url` and `api_key` (the new ones), all read into one `Provider` group.

The common case is an OpenAI-compatible gateway — any server that speaks the
`v1/chat/completions` dialect the `openai` backend does, a class that includes most self-hosted
proxies. Point `backend` at `openai`, set your own `base_url` and `api_key`, and the gateway is used
as-is:

```sh
export KORI_BACKEND=openai
export KORI_PROVIDER_BASE_URL=http://localhost:3001/v1
export KORI_PROVIDER_API_KEY=sk-your-unified-key
export KORI_MODEL=auto
kori
```

Or the same provider in `~/.kori.yml`:

```yaml
provider:
  backend: openai
  base_url: http://localhost:3001/v1
  api_key: sk-your-unified-key
  model: auto
```

`base_url` and `api_key` are optional everywhere. Leave `base_url` out and the backend's own default
stands (`https://api.openai.com/v1` for `openai`); leave `api_key` out and the backend falls back to
its usual environment variable (`OPENAI_API_KEY` and the rest). The `openai` backend always requires
*a* key, so for a genuinely keyless local endpoint set `OPENAI_API_KEY` to any non-empty dummy value;
the config errors otherwise. They only matter together with a custom endpoint.

The file note from the precedence table above softens here: the config file **can** hold an
`api_key`, for a private endpoint where a committed dotfile is the reasonable home for it. The old
warning still stands for vendor keys — a file holding an `OPENAI_API_KEY` you actually use is a file
that can never be committed to a dotfiles repo. A custom endpoint like freellmapi has its own key,
so the two stories do not collide. In either case the env-var route is the one that keeps the key
out of any file, and `KORI_PROVIDER_API_KEY` shares the `KORI_` prefix the rest of the layer
uses.

`anthropic` cannot be pointed at a custom endpoint: its `Config` takes a pre-built client rather
than a URL and key. `openai`, `openrouter` and `google` all take a `BaseURL`. A local OpenAI-compatible
gateway is the overwhelmingly common case, and it is the case that works.

### Profiles and /model command

Profiles define reusable model and provider configurations in `~/.kori/profiles/<name>.yml`. Kori reads any `.yml` file in that folder on startup and during interactive sessions.

```yaml
# ~/.kori/profiles/fast.yml
name: fast
provider:
  backend: google
  model: gemini-2.5-flash
reasoning:
  effort: low
  thinking: true
  budget: 2048
```

You can select a profile in `~/.kori.yml` using the top-level `profile` key:

```yaml
profile: fast
```

Or pass `-profile <name>` on the command line, or set `KORI_PROFILE=<name>` in the environment. Precedence order follows:

```
defaults < profile < ~/.kori.yml < environment < flags
```

During an interactive session, the `/model` command switches models on the fly:

- `/model` opens an interactive dropdown menu listing discovered profiles and detected provider models.
- `/model <name>` switches immediately to the named profile or model string (e.g. `/model fast` or `/model openai/gpt-5.4`).

The conversation history remains intact when switching models mid-session.

## Context and skills

Four things the TUI adds beyond flags and the model's own tools, all client-side, none of them
in the core `nacelle` package — the same "the library must never read configuration from disk"
rule above applies to these too, not only to settings.

**This session's own facts** (always on, `environment.go`). Prepended to whatever `System`
says, before context and skills, because it is the most general layer and the only one with no
switch: the absolute working directory, that the file and search tools are confined to it and
strip a leading `/` rather than refusing it, today's date, that the answer is rendered as
markdown, and whether anyone reviews a tool call before it runs. Two clauses are conditional the
way the banner's own lines are — the `run_command` path exception and the paragraph naming
irreversible commands appear only under `-bash`, because explaining a tool the model was not
given teaches it to skim the rest. These are facts the model cannot recover from any tool schema.
Advice about *which* tool to reach for is deliberately not here; it lives in the tool
descriptions, where it travels with the tool that needs it and costs nothing when that tool is
not mounted.

The persona above it (`defaultSystem`, `main.go`) is the one layer `-system-prompt` replaces
outright, and it stays deliberately short. Codex ships two prompts for one harness — 6.6&nbsp;KB
for the models post-trained on it, 24&nbsp;KB for general GPT-5 — so prompt size mostly measures
how much the model was *not* trained on your harness, and Claude is well inside the trained case
for this shape of tool. What it carries is only what a terminal changes about answering: brevity,
paths rather than pasted source, and not claiming something works unchecked.

**Project and global context** (`-project-context`, `context.go`). Every `CLAUDE.md` and
`AGENTS.md` found walking up from `-root` to the filesystem root, plus `~/.agents/AGENTS.md` —
the [AGENTS.md standard](https://agentsstandard.com)'s own global-base path, also read by
Codex, Cursor, Copilot, Gemini and pi (at its own equivalent, `~/.pi/agent/AGENTS.md`) — all
appended to the system prompt, most general first. A user's own `~/.claude/CLAUDE.md` is
deliberately **not** read: it assumes Claude Code's own tools, hooks and slash commands, where
an `AGENTS.md` at the standard's own path is written, by the convention's premise, to make
sense to whichever agent finds it. None of this is trust-gated — see
[architecture.md](architecture.md) for why a plain instruction file and a skill are not the
same risk.

**Skills** (`-skills`, `-trust-skills`, `-skill-dir`, `skills.go`), following the
[Agent Skills specification](https://agentskills.io/specification): every `SKILL.md` under
`~/.agents/skills/` (no trust needed, the user's own machine), every one under a **trusted**
`.agents/skills/` found the same way the context walk works, and every one under a directory
named by `-skill-dir` (repeatable), `KORI_SKILL_DIRS` (colon-separated) or `skill_dirs` in
`~/.kori.yml`. Only a skill's `name` and `description` ever reach the system prompt; the
model reads the rest with `read_file` once it decides a skill applies. `-trust-skills` trusts
every project-local `.agents/skills/` found on that run and remembers the decision in
`~/.kori/trust.json`, keyed by canonical directory — run it once per project, not on every
launch.

**MCP servers** (`mcp.go`). A server is written inline under `sources.mcp:` in `~/.kori.yml` — nothing
else is needed to start one:
```yaml
sources:
  mcp:
    mycelium:
      command: mycelium
      args: [mcp]
```
The key is the server name;the definition uses the keys the `mcpServers` format spells — `command` + `args`
(stdio), `"type": "http"` + `url` (remote), `env`, `cwd`, `headers`, `disabled`. Both stdio and HTTP
servers work, on **either backend**:the tools are bridged to `nacelle.Tool`, so
`-approve-tools` gates them like every other tool and `-backend openrouter` gets them too. `${VAR}`和
`${VAR:-default}` expand,, so a token stays in the environment rather than in `~/.kori.yml`. Writing
the definition here means there is no second file to keep in step and no pointer to remember to point tor

For the times a server is already configured for another client, `-mcp <file>` (repeatable) names
a `.mcp.json` in the `mcpServers` format and merges its servers in — by server name,, with the flag's
servers winning over the inline ones,, so `-mcp ~/.claude/.mcp.json` reads what Claude Code already
has without copying it. Replacing would mean naming one server silently switching off nine already
configured; merging is how a personal inline list plus another client's file stack the way every client
in this ecosystem layers its own scopes.

A server that will not start **ends the run**, which is the opposite of how skills and project
context fail. Those are discovered, so finding nothing is indistinguishable from there being
nothing to find; this was asked for by name, and a tool that is quietly missing reads as a model
refusing to work rather than as a server that is down.

Nothing is discovered. A `.mcp.json` sitting in the working directory is **not** read, because it
names executables to run — strictly worse than the project-local `.agents/skills/` this client
already gates behind `~/.kori/trust.json`, since a skill is text the model may decline to act
on and this is a subprocess started before the model is asked anything. Every server kori
opens was named on the command line or in your own `~/.kori.yml`.

The spec defines the `SKILL.md` file, not where it has to live on disk — every tool picks its
own directory (Claude Code reads `~/.claude/skills/`, pi reads `~/.agents/skills/` and its own
`~/.pi/agent/skills/`), so `~/.agents/skills/` is kori's own choice, not a location any other
tool already reads. `-skill-dir` is how kori sees a directory that belongs to one of those,
without moving or copying anything into its own — the same problem pi itself solves with a
`skills` array in `settings.json` pointed at `~/.claude/skills` or `~/.codex/skills`. No trust
gate applies to it, same reasoning as `~/.agents/skills/` above: naming a directory here is
something only the person running kori, on their own machine, can do in the first place.

**The banner** (`main.go`'s `banner`) is how much of this is actually visible before typing
anything: line one names the backend and model, line two the resolved `-root`, how many skills
loaded (from every source above, combined), how many `CLAUDE.md`/`AGENTS.md` files did, and
whether bash is on. Nothing on it is decorative — each answers a real "is that actually on"
question this client otherwise had no way to check short of a debug build.

`bash off` earns its place because the symptom arrives from the model rather than from this
client: asked to build and run something, it answers that it has no terminal and cannot run a
command. That is true and deliberate — `run_command` is unconfined, so it stays opt-in — but
nothing connected that answer back to a `tools.run_command: false` written once in `~/.kori.yml` and
forgotten. Turn it on with `-bash`, `KORI_BASH=1`, or `tools.run_command: true`.

## Reading a page

`tools.web_fetch` (`KORI_FETCH`, `-fetch`) lets the model read one web page by URL. **On by default**,
unlike bash, and it is what turns a page's URL into text the model can act on.

It is on by default because it cannot change anything and cannot reach anything but the public
internet: loopback, private ranges, the cloud metadata endpoint and the special-use ranges are
refused in the dialer, on every redirect hop. Pages come back as text — headings, lists, code
blocks, and links resolved to absolute URLs so the model can follow one by calling the tool
again. `text/markdown` is asked for first, which Cloudflare and Vercel answer by converting at
the edge for roughly 80% fewer tokens.

Turn it off with `web_fetch: false` for the one risk the guard cannot cover: a fetched page is
written by a stranger and read by the model as instructions, so it can ask for another URL with
something from the conversation in its query string. With `tools.run_command: true` that channel already
exists through `run_command`; with bash off, this is the only one. The banner says `fetch off`
when it is off, since a model that cannot read a page it just found needs the reason on screen.

Sites do refuse automated clients. This one identifies itself honestly rather than impersonating
a browser — every current source agrees that negotiated access is what gets an agent through and
that a spoofed user agent is what gets an address banned — so a 403 from an anti-bot layer is
reported as one, with the suggestion to try another source, rather than as the page not existing.

## Tool approval

`-approve-tools` (`KORI_APPROVE_TOOLS`, `approve_tools`) asks before every tool call runs.
**Off by default**: a nil `Config.Approve` means the SDK itself runs every call unasked, and
the TUI's own default matches — nobody gets a behaviour change without opting in.

When on, a call blocks in the status line on `y` (allow this one call), `a` (allow this tool
for the rest of the session — remembered in memory only, never written to disk, unlike
`-trust-skills`) or `n` (deny; asked again next time). Asks are serialized to one question at
a time, because a backend can run several tool calls concurrently within a turn and this
client has exactly one place to show a question. A pending question is answered or abandoned
by the same ctrl+c/ctrl+\ that already gets a wedged run's way out — both wait on the run's
own cancellable context, so cancelling one unblocks the other with no separate mechanism.

A denial is reported on `ToolEvent.Refused`, not as a tool failure — the SDK converts it into
a normal tool-result error block for the model to see, the same as any other failed call.

## Deny elevation

`security.deny_elevation` (`KORI_DENY_ELEVATION`) refuses `run_command` calls that try to
elevate privileges: `sudo`, `su`, `doas`, `pkexec`, checked after every chain separator and
inside subshells, not only at the start of the line. **On by default.** The refusal is an
ordinary tool error the model reads, so it adapts instead of retrying.

It is a policy guard against accidents and injected instructions, not a sandbox. A string
check cannot stop a setuid binary that is not sudo, `docker run -u root`, or anything already
running as root; the OS decides who may elevate, and an account outside the `sudo` group is
the actual wall. Set it to `false` to allow elevation. Phase 2 adds a pty password flow, where
sudo prompts on a terminal the harness allocates but never reads, so the agent never sees the
password. [docs/sudo.md](sudo.md) carries the full analysis.

## Cron jobs

Cron is kori run unattended: the same agent, driven headless by an external
scheduler — a systemd timer or a crontab — while nobody watches. kori owns
the job definitions and the run itself; the clock belongs to systemd, which is
why there is no daemon here to reload or supervise. The `kori cron`
subcommand ties the two together:

```sh
kori cron list            # every job, trusted or not
kori cron run news        # run one job now, through the headless path
kori cron install news    # install the job directly into crontab
kori cron uninstall news  # remove the job from crontab
kori cron trust news      # approve one job file's current contents
```

`kori cron list --json` prints the same table as one JSON document; set
`ui.cron_list_json: true` to make that the default output.

### The jobs folder

Each job is one YAML file in `~/.kori/jobs/` — `news.yml`, `healthchecks.yml`,
one file per job, the job's own fields at the top level of the file rather than
wrapped in a list. The job's name defaults to the filename stem, so `news.yml`
defines the job `news` and `kori cron run news` runs it; a `name:` key
overrides it for the rare file whose stem is not the name you want (a name
must be unit-safe: letters, digits and `. _ -` only).

The folder is scanned on every `cron` invocation, list included, so there is no
daemon and no reload step: a file dropped in appears in `cron list` on the next
invocation, a file edited changes the job, and a file deleted takes the job
with it. Deleting the file is the off switch.

Job files decode strictly, the way `~/.kori.yml` itself does: a typo in a
job file is a hard error naming the file, not a silent half-job. Loud is the
right setting for something that fires at 2am.

### Trust

A job file is instructions an agent will follow unattended with your provider
keys in reach, so it is gated the way project skills and hooks are: a human
must approve the exact bytes before they run.

- A new file shows up in `kori cron list` immediately, marked untrusted.
- `kori cron run` and `kori cron install` refuse an untrusted job until
  someone runs `kori cron trust <name>`, which prints the file's path, its
  name and its prompt, and asks before recording anything.
- Trusting records a SHA-256 hash of the file's contents against its path in
  `~/.kori/trust.json`, the same store that remembers approved skill
  directories. Editing the file changes the hash, and a changed file is
  refused again — trust re-arms on every edit, so no one-line change to a
  trusted job slips through unreviewed.
- Trust is per machine. A `~/.kori/jobs/` folder synced across machines
  carries the files but not the approvals, so each machine needs its own
  `kori cron trust` — the person trusting is a person at that machine's
  keyboard, and that is the point.

Trusting makes a job **runnable**, not **scheduled**: it clears `cron run` and
unblocks `cron install`, and `cron install` still only prints the systemd
service and timer units for you to save and enable yourself. Nothing kori
does creates the timer.

### A job file

```yaml
# ~/.kori/jobs/news.yml — one file, one job; the job's name is the stem.
when: daily
prompt: |-
  Summarise what changed on the pages in my feed list since yesterday.
  Keep it under 200 words and name the two most important changes.
workdir: ~/Code/Facile/notes
delivery: file:~/notes/cron
timeout: 600
enabled: true

provider:
  backend: openai
  model: gpt-5.4

security:
  deny_elevation: true

tools:
  run_command: true
  web_fetch: true

reasoning:
  effort: low

limits:
  max_iterations: 5
```

| Field | Default | What it does |
|---|---|---|
| `when` | empty (`install` writes `daily`) | The `OnCalendar=` line of the generated timer — any systemd calendar expression. kori never parses it; systemd does |
| `prompt` | — required | The question the run starts from |
| `workdir` | — install requires one | The run's root and the unit's `WorkingDirectory=`. Without one the run would execute wherever the scheduler happens to start the unit, which is never the project |
| `delivery` | empty — no transcript written | `file:<dir>` appends the transcript to `<dir>/<name>-YYYY-MM-DD.md` and one status line per run to `<dir>/<name>.log` |
| `timeout` | `300` | The service's `TimeoutStartSec=` — systemd kills a run outliving it |
| `enabled` | `false` | `cron install` refuses a disabled job, so test-run-first is explicit: `cron run` it once, read the output, then set `enabled: true` |
| `name` | the filename stem | Overrides the job's name |
| `commands` | `false` | The legacy spelling of `tools.run_command`, kept from the inline-list era for files that already use it |

### Per-job overrides

Every group the interactive config has can be overridden per job: a job can run
on a different backend than the interactive one, tighten its own isolation, or
cap its own spend. Members are per-field — mention a group with one key set and
every key you leave out keeps the config's value; a job saying nothing runs
exactly on the config's settings.

| Group | Keys | Notes |
|---|---|---|
| `provider:` | `backend`, `model`, `base_url`, `api_key` | A different backend, endpoint or key for this job — the cheap model for the dawn summary, the local gateway for the nightly audit. `provider.model` wins over the flat `model:` when both are set |
| `security:` | `approve_tools`, `path_isolation`, `deny_elevation`, `env_isolation` | `approve_tools` is forced off for every cron run whatever the job or the config says — a run nobody watches cannot answer a prompt, and a prompt nobody answers is a run that never starts. The other three apply as written |
| `tools:` | `run_command`, `parallel_agents`, `web_fetch`, `tasks`, `diagnostics` | `run_command` defaults to **false** for cron, the reverse of the interactive default: opt in with `tools.run_command: true` or the legacy `commands: true`, and `tools:` wins when both are spelled |
| `reasoning:` | `effort`, `thinking`, `budget` | Per-job reasoning spend — `low` for the summariser, `high` for the auditor |
| `limits:` | `max_iterations`, `compact_at` | Iteration and compaction caps that only make sense per job |
| `hooks:` | hook specs, same shape as the config's own | **Append** to the config's hooks — a job adds chains on top of the ones every run gets, it never replaces them |
| `gates:` | gate specs, same shape as the config's own | **Append** to the config's gates, same rule |

### Migration

The inline `cron:` list in `~/.kori.yml` is gone. A config still carrying it
is refused at load with an error naming `~/.kori/jobs/` — the same refusal an
unknown key gets, because a silently ignored job list is a scheduler that
quietly stopped existing. Each list entry becomes its own file in the folder:
the entry's `name` names the file (`news` becomes `news.yml`), the rest of the
entry moves across unchanged, and `commands: true` keeps working as the legacy
spelling of `tools.run_command: true`.

## Slash commands

Typing `/` at the start of a line names one of the client's own commands instead of a
question:

| Command | Does |
|---|---|
| `/clear` | Reset the transcript, the conversation sent to the model, and the running cost total. Same client, new session. Never starts a run. |
| `/cost` | Display cumulative token usage, cost, and tools executed so far. Never starts a run. |
| `/help` | List the commands above and the keybindings (esc, ctrl+c/ctrl+\, ctrl+t, ctrl+g). Never starts a run. |
| `/model [name]` | Switch active model or profile mid-session, or open model picker. Never starts a run. |
| `/quit` | Quit. Never starts a run. |
| `/resume` | Resume the most recent recorded session for the current project. Never starts a run. |
| `/sessions` | List recorded sessions for the current project with timestamps and models. Never starts a run. |
| `/status` | Show runtime status, context usage, cost, and active configuration. Never starts a run. |
| `/skill:name [what to do]` | Run a loaded skill directly. Starts a run, unlike the seven above. |

An unrecognised command or skill (a typo like `/clera`, or a `/skill:name` that names nothing
loaded) is reported back rather than sent to the model as a literal question — the same
trade-off every peer client with slash commands makes, on the same reasoning: a real question is
far less likely to start with a slash than a mistyped command is.

`/skill:name` sends that skill's own `SKILL.md` content as the question, instead of waiting for
the model to decide on its own that the skill applies and read it with `read_file` — the same
shortcut pi's own `/skill:name` offers, for the same reason: "models don't always" make that
call themselves. Anything typed after the name is appended as `User: <that text>`, the same
convention pi uses, so `/skill:pdf-tools extract the tables` tells the skill what to do with
itself rather than only that it applies. Every loaded skill counts, from `~/.agents/skills/`,
a trusted project `.agents/skills/`, or a `-skill-dir` — one namespace, `skill:`, so a skill
named `clear` or `help` can never collide with the three commands above.

Typing `/` alone opens a dropdown listing every command and skill, each skill with its own
description; more typed narrows it, ranked best match first — a prefix beats a plain substring
beats a fuzzy, order-preserving scatter of the same letters, so `/skill:rev` finds `review`,
`facile-review` **and** `hunk-review`, not only a name that starts with what was typed. While
it's open, `up`/`down` move the selection instead of scrolling the transcript, `tab`/`enter` fill
the prompt with the highlighted entry (plus a trailing space, and without submitting — most
useful with `/skill:name`'s own trailing argument) and `esc` closes it, leaving what was typed
alone. A second, ordinary `enter` is what actually sends it.

### The prompt

The prompt wraps and grows as you type, up to ten rows, then scrolls inside itself. Every row it
gains is a row the transcript gives up, and gets back the moment the question is sent. `alt+enter`
(or `ctrl+j`) starts a new line without sending; `enter` always sends.

It used to be a single-line input, which scrolled sideways instead of wrapping — a question wider
than the terminal slid out of view a character at a time and read as though it were typing over
itself.

`up`/`down` belong to the prompt at every height — there is no transcript here to scroll, so
nothing competes for them and a wrapped question is editable whatever size it has grown to.

### Scrolling

The client renders inline, on the terminal's own screen, so scrolling is the
terminal's job — wheel, scrollbar, `tmux` copy-mode, your terminal's search, all of it, on the
whole session rather than on a window this client owns. There are no scroll keys here because
there is nothing here to scroll.

That is a deliberate reversal. It ran on the alternate screen once, which hands a program a
blank page with no scrollback of its own, so the client had to own scrolling: capture the wheel,
which takes click-drag selection with it, leave copy-mode reaching nothing, and un-draw the whole
session on quit. Those arrived as four separate complaints and were one decision. Giving the page
back answers all of them, and costs one thing — a printed line belongs to the terminal, so a
resize reflows it the terminal's way rather than this client's, and a theme change applies from
then on rather than retroactively.

### While a run is going

The status line spins for as long as the run lasts, and names what it is waiting on — `waiting for
a response` before the model answers, `running read_file` between a tool's call and its result,
`running 3 tools` when the backend runs a turn's tools at once. It is on the one row that is
always drawn, so it stays visible while scrolled back through the transcript.

This replaced a spinner that only covered the gap before the first event. Every gap after it — a
tool running, the model called again with its result — left the screen still, and a client that has
stopped moving reads exactly like one that has stopped working.

**Enter still works while a run is going**: the line is queued, shown above the prompt, and sent
once the run finishes. A queued `/help` is still a command.

**`esc` stops the run**, and that is all it ever does. `ctrl+c` stops one too, but it is also the
key that quits an idle client, so the press that abandons an answer is a press that has to be
thought about first; `esc` is the one that never is. Either way the queue goes with the run rather
than being started by it, and the client says how many it dropped — the alternative is a message
that silently never gets asked. Idle, `esc` does nothing at all, and with the `/` dropdown open it
closes that first: one visible thing per press.

The question is on the screen before either of them is worth pressing. It used to appear only when
the answer did: the client printed what it had said only after running what that message started,
and what a question starts is a wait on the model. The prompt emptied and nothing else happened,
which reads as a client that swallowed it. The `⏺` line naming a tool had the same fault and was
worse for it — it waited on the tool it was announcing, so a six-second command was announced six
seconds late. The CHANGELOG carries the measurements.

### What a tool call reads as

A tool that worked is **one line**, carrying its own duration:

```
⏺ read_file(view.go) · 12ms
⏺ run_command(go test ./...) · 2.34s
```

The name, the single argument that says which thing it acted on, and how long it took. The
argument is picked by key (`path`, `file_path`, `file`, `command`, `pattern`, `query`, `url`,
`name`), falling back to the sole key when a call has exactly one and to `name()` when nothing
there is worth printing. That beats a table of tool-name to argument-name, because this client
does not know which tools an agent was built with and half of them arrive over MCP under a
server's own names.

The line is held until the result arrives, which is the only way the duration can be on it: a
printed line belongs to the terminal's scrollback and can never be rewritten. Nothing is hidden
while it waits, because the status line names the running tool for exactly as long as the line is
held. A run that ends with calls still in flight (the iteration limit, or `esc`) still prints
them, without a duration.

**A failure stays two lines and stays loud**, because it is the thing a reader must not scroll
past:

```
⏺ edit_file(transcript.go)
  ⤷ edit_file failed after 3ms: no such file
```

What this replaced printed the raw JSON input and then a second `done in 12ms` line under it,
so every transcript was twice as tall in order to report that nothing had gone wrong.

**A file edit also draws its diff.** When `edit_file` or `write_file` changes a file, the
one-line report is followed by what changed, git-style — removals in the terminal's red,
additions in its green, with three unchanged lines around each block so the eye can find where
in the file it is looking:

```
⏺ edit_file(toolline.go) · 4ms
   1   func (m *model) finished(tool *nacelle.ToolEvent) {
   2 -   line := m.run.running[tool.ID]
   2 +   line, held := m.run.running[tool.ID]
   3       delete(m.run.running, tool.ID)
```

Line numbers run down the left gutter — green on additions, red on removals, muted on context.
Added and removed lines each sit on a faint wash (the pure green or red blended to ~20% onto the
pane's backdrop) rather than a solid block, so a long change stays readable without shouting.
The diff is cut to the window width so nothing wraps in scrollback, and capped at four hundred
lines with a `… more` marker saying it was cut rather than pretending it wasn't. A wholesale
rewrite too large to align line by line reads as one block of removals against additions.

`edit_file` carries both sides of the change in its own input, so nothing is read from disk to
draw it; `write_file` carries only the new contents, so what the file held when the call was seen
becomes the before side — empty for a file being created, which shows as all additions. Only
these two tools get a diff: search, commands and MCP tools render exactly as they did before.
Turn it off with `diffs: false` (`KORI_DIFFS`, `-diffs`) to restore the bare one-line report.

**Input with a repeated key is refused rather than summarised.** `encoding/json` keeps the last
value of a duplicate key silently, so `{"command":"ls","command":"rm -rf /"}` can be shown as
`run_command(ls)` over a call that ran something else. The line says `input has a duplicate key`
instead, and with `-approve-tools` on the gate denies the call outright without ever drawing a
prompt: an approval prompt that offers `y` for a call nobody can render is the same lie with a
warning label.

### Reasoning

With `thinking: true`, a turn's reasoning collapses to one quiet line:

```
· thought for 4.2s · ctrl+t to expand
```

The hint appears once per session, not under every answer. **`ctrl+t` expands**: it prints the
retained reasoning below, and sticks, so later turns print theirs in full until it is pressed
again. Expanding cannot replace the collapsed line already on screen for the same reason the
tool line carries its own duration, so a second copy underneath is what expanding means here.
Only the most recent turn's reasoning is retained. `ctrl+t` takes the press whatever is on
screen, which costs the prompt its `transpose-character-backward` binding.

### Quitting

`ctrl+c` when nothing is running, `ctrl+\` at any time, and `/quit` all end the session, and the
session simply stays where it is. Nothing is printed on the way out and nothing needs to be: what
was said went into the terminal's scrollback as it happened, so quitting leaves it exactly as
running left it.

That used to need a deliberate dump at exit, because the alternate screen hands the old page back
untouched and quitting un-drew the whole conversation. Inline rendering removed the problem rather
than the workaround.

## Sandboxes & Remote VMs

The `sandbox:` section in `~/.kori.yml` configures microVM and remote SSH execution targets:

```yaml
sandbox:
  default: ""                     # default target name when invoked as 'kori sandbox'
  user: boite                     # default SSH user
  port: 2226                      # default port
  ssh_key_path: ~/.ssh/id_ed25519 # default identity file
  root: /workspace                # default remote workspace
  auto_sync: true                 # automatically sync kori binary
  auto_snapshot: false            # snapshot overlay disk on completion (boite only)
  targets:
    staging:
      backend: ssh
      host: staging.example.com
      port: 22
      user: deploy
      workdir: /srv/app
    dev-vm:
      backend: boite
      vm_name: dev-box
```

Commands:
- `kori sandbox list`: lists configured targets and local Boite VMs.
- `kori sandbox <target> [prompt]`: launches interactive or headless session inside the target.
- Direct SSH targets (e.g. `kori sandbox user@host:port` or `~/.ssh/config` host alias) are supported directly even when Boite is not installed.

