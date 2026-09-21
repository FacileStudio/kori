# Plan: mid-session `/model` switch

Status: planned. Written 2026-09-14.
Survey artifact: https://mycelium.facile.studio/artifacts/2026-09-14-mid-session-model-switching-in-terminal-agent-harnesses-f93e3e

## Goal

Add a `/model` slash command that switches the active model for the rest of an interactive session without dropping conversation history, usage metrics, or compaction state.

## Problem and evidence

`m.agent` is assigned once during initialization in `internal/tui/model.go` and never changes during a session. Switching models currently requires quitting and restarting `kori`.

Every major harness except goose provides an in-session model switch. Claude Code, Codex, opencode, Crush, gemini-cli, and aider all support mid-session switching while keeping the conversation intact. Compaction pressure and cost management are the primary operational reasons to switch between fast and deep models mid-turn.

## Technical approach

The SDK's `nacelle.Agent` is immutable after construction. It provides no setter for its backend, exposing only the `Backend()` getter. A model switch must therefore reconstruct a new `nacelle.Agent` and reassign it to `m.agent`, following the precedent set by `/parallel` in `internal/tui/parallel_detached.go`.

The switch must stay session-scoped in v1. It affects only the current interactive session and does not write back to `~/.kori.yml`.

### State changes on swap

1. `m.agent`: reconstructed via `agent.Swap` using the updated backend and wrapped with `nacelle.Retry`.
2. `m.delegate`: updated so that nested agents dispatched by `/parallel` use the new backend.
3. `m.sink`: rebuilt as `usage.NewSink(m.run.root, newModel)` so future usage events log the correct model name.
4. `m.backend` and `m.model`: stored on `m.core` so `/status` and the UI reflect the active model without unsafe interface assertions.
5. `m.banner`: updated so `/clear` echoes the new model name.
6. `m.policy` and `m.compactAt`: re-resolved wholesale with `agent.Policy` if the new model has a different context window. Updating `m.compactAt` alone is not enough — it is only the gate; `Policy.Tier` and `Policy.Trigger` read `m.policy.Window` and its ratios, so a stale window would tier every later pass against the previous model's denominator and print the wrong ratio in the footer.
7. `m.session`: left untouched on disk. `sessions.OpenSession` generates a new timestamped file and breaks Herdr tracking. The single existing session file continues logging questions and answers for the entire run.

### Invariants

1. Conversation messages and token counters (`m.conversation`, `m.spent`, `m.size`, `m.trimmed`) remain untouched.
2. Compaction follows automatically: `m.summarizer()` in `compact_light.go` reads `m.agent.Backend()`, so subsequent compaction runs use the new backend immediately.
3. Cold prompt cache notice: switching models invalidates provider KV caches. The command card prints an explicit note in the transcript.
4. Idle-only gate: switches are refused when `m.run.busy || m.compacting`.
5. Error safety: an invalid model or construction error aborts the swap and leaves the existing agent and state intact.

## Implementation steps

### 1. Export swap helpers in `internal/agent`

Modify `internal/agent/agent.go` and `internal/agent/compact.go`:

- `internal/agent/compact.go` already exports `ResolveBudget(compactAt *int64, c settings.Compaction, backend nacelle.Backend) Budget`, and `internal/agent/policy.go` folds that budget plus the two end sizes into a `compaction.Policy`; the swap re-runs both, assigns `m.policy`, and takes `m.compactAt = m.policy.Ceiling` from it.
- In `internal/agent/agent.go`, define `SwapResult`:
  ```go
  type SwapResult struct {
      Agent   *nacelle.Agent
      Backend nacelle.Backend
      Config  nacelle.Config
  }
  ```
- In `internal/agent/agent.go`, export `Swap`:
  ```go
  func Swap(cfg nacelle.Config, provider settings.Provider) (SwapResult, error)
  ```
  `Swap` constructs the backend via `chosen(Config{Provider: provider})`, wraps it in `nacelle.Retry`, updates `cfg.Backend`, and constructs a new agent via `nacelle.New(cfg)`. This reuses the existing toolset and system prompt without rebuilding file descriptors or MCP connections.

### 2. Add backend and model fields to `internal/tui/types.go`

- Add `backend string` and `model string` to `core` in `internal/tui/types.go`.
- Set these fields during `NewModel` in `internal/tui/model.go` and `boot` in `internal/tui/launch.go` using `SessionConfig`.

### 3. Implement swap command and execution in `internal/tui/model_swap.go`

Create `internal/tui/model_swap.go` to keep `command.go` under the filet function limit:

- `modelCmd(args string) tea.Cmd`:
  - If `m.run.busy || m.compacting`, print a refusal notice and return nil.
  - If `args == ""`, open the model picker.
  - If `args != ""`, resolve aliases, validate against the current backend, and run `m.performSwap(targetModel)`.
- `performSwap(targetModel string) tea.Cmd`:
  - Build `settings.Provider` with `m.backend` and `targetModel`.
  - Call `agent.Swap(m.delegate, provider)`.
  - On error, display the error and retain the current agent.
  - On success, reassign `m.agent = res.Agent`, `m.delegate = res.Config`, `m.model = targetModel`.
  - Rebuild `m.sink = usage.NewSink(m.run.root, targetModel)`.
  - Update `m.banner` to show the new model.
  - Re-run `agent.ResolveBudget` against `res.Backend` with the session's resolved compaction settings, rebuild the policy with `agent.Policy(budget, m.policy.KeepTurns, m.policy.AnchorMessages)`, and assign it to `m.policy` (taking `m.compactAt = m.policy.Ceiling`), so the ratios, the window and the ceiling all move together and the ladder re-derives from the new model's window.
  - Print confirmation card: `→ switched to <backend>/<model> · context re-reads cold from here (no prompt-cache hits)`.

### 4. Implement interactive picker in `internal/tui/model_picker.go`

Create `internal/tui/model_picker.go`:

- Maintain static catalogues and aliases for supported backends:
  - Anthropic: `claude-opus-5` (alias: `opus`), `claude-sonnet-5` (alias: `sonnet`), `claude-haiku-5` (alias: `haiku`), `claude-3-7-sonnet-20250219`, `claude-3-5-haiku-20241022`.
  - Google: `gemini-3.7-flash` (alias: `flash`), `gemini-3.7-pro` (alias: `pro`), `gemini-2.5-pro`, `gemini-2.5-flash`.
  - OpenAI: `gpt-5.4` (alias: `gpt-5`), `gpt-5-mini`, `o3`, `o3-mini`, `o1`, `gpt-4o`.
- Manage modal picker state via `m.modelPicker`.
- Support keyboard navigation: Up/Down to navigate, Enter to confirm and swap, Esc to dismiss.

### 5. Wire command dispatch in `internal/tui/command.go`

- Add `/model` to `commands` map: `"model": func(m *Model) tea.Cmd { return m.modelCmd("") }`.
- Add arg handling in `parseCommand`:
  ```go
  if name == "model" {
      return func(m *Model) tea.Cmd { return m.modelCmd(rest) }, true
  }
  ```
- In `statusCmd`, append active model information:
  ```go
  lines = append(lines, fmt.Sprintf("model · %s/%s", m.backend, m.model))
  ```
- `command.go` stays at exactly 8 functions, complying with filet's `funcsPerFile: 8`.

### 6. Tests in `internal/tui/model_swap_test.go`

Create unit tests covering:

- Successful swap updates `m.agent`, `m.model`, and `m.delegate.Backend`.
- `summarizer()` uses the new backend.
- Refusal when `m.run.busy` or `m.compacting`.
- Failure on invalid model preserves previous agent and banner.
- Compaction threshold re-resolves when context window changes.
- `/status` output reflects the swapped model.
- Conversation and spend totals are preserved across swaps.

## File checklist

| File | Action | Purpose |
|---|---|---|
| `internal/agent/compact.go`, `internal/agent/policy.go` | Reference | Re-run the exported `ResolveBudget` and `Policy` for the new backend |
| `internal/agent/agent.go` | Modify | Add `SwapResult` and `Swap` |
| `internal/tui/types.go` | Modify | Add `backend` and `model` fields to `core` |
| `internal/tui/model.go` | Modify | Set initial `backend` and `model` |
| `internal/tui/launch.go` | Modify | Set `backend` and `model` in `boot` |
| `internal/tui/command.go` | Modify | Wire `/model` command and update `/status` |
| `internal/tui/model_swap.go` | New | Implement swap execution and validation |
| `internal/tui/model_picker.go` | New | Implement interactive modal picker |
| `internal/tui/model_swap_test.go` | New | Comprehensive unit tests |
| `CHANGELOG.md` | Modify | Record new feature |
| `README.md` | Modify | Document `/model` command |

## Quality gates

- `go test ./...` passes.
- `golangci-lint run ./...` passes.
- `filet check` reports 0 errors and 0 warnings.
- All filet limits respected:
  - `funcsPerFile: 8`
  - `fileLines: 250`
  - `funcLines: 35`
  - `funcStatements: 25`
  - `params: 5`
  - `returns: 3`
  - `structFields: 16`
