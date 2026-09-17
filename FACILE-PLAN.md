# Plan: Background Session Management and Parallel Agent Concurrency

## Goal
1. Allow users to detach or close a kori session and let it run in the background, view running/completed sessions via `kori sessions` (or `kori list`), and resume or attach back to them.
2. Fix parallel agent concurrency limit so that 7+ subagents run simultaneously instead of being throttled to 4.

## Evidence and Root Causes
1. **Parallel Concurrency Limit**: In `nacelle` (`parallel_agents.go:230-238`), `clampConcurrency(n)` defaulted to 4 when `n == 0` (unspecified) and clamped at 8. In `kori` (`internal/agent/delegate.go` and `internal/tui/parallel_detached.go`), `MaxConcurrency` was left at default 0, throttling all parallel fan-outs to 4 workers at a time.
2. **Session Backgrounding & Management**: `kori` recorded session JSONL logs under `~/.kori/sessions/<timestamp>-<pid>.jsonl`, but had no CLI command to list running/past sessions with live process status, no subcommand to attach or kill background sessions, and no `/detach` TUI command or `--detach` flag for hands-free background runs.

## Architecture & Implementation

### 1. Nacelle SDK Concurrency Upgrade
- Update `nacelle/parallel_agents.go`:
  - `clampConcurrency`: default `MaxConcurrency == 0` to 16, clamp range up to 32.
  - Update `nacelle/parallel_agents_test.go` test cases to verify 16+ parallel workers.
- Update `kori/internal/agent/delegate.go` and `kori/internal/tui/parallel_detached.go`:
  - Pass `MaxConcurrency: 16` (or configured `MaxConcurrency`).
  - Wire `MaxConcurrency` into `settings.Limits`.

### 2. Session Background Execution & Process Tracking (`internal/sessions`)
- Enhance `internal/sessions`:
  - `SessionInfo` struct: ID, Path, PID, Started, ModTime, Backend, Model, Root, Status (`running`, `idle`, `completed`, `failed`), LastMessage, ActiveTools.
  - `ListSessions(projectRoot string) []SessionInfo`: reads JSONL headers + tail entries, queries OS process table to verify whether PID is live.
  - `GetSession(idOrPath string) (*SessionInfo, error)`: resolves partial or full session ID.
  - `KillSession(idOrPath string) error`: gracefully terminates a background session PID.
  - Session status updates: write session state markers (`status: completed`, `status: detached`) into session logs.

### 3. CLI Commands (`cmd/`)
- Add `cmd/sessions.go` and `cmd/sessions_list.go`:
  - `kori sessions` (aliases: `kori list`, `kori session`):
    - `kori sessions [list|ls] [--json] [--all]`: lists active and recent sessions with styled status badges, PID, duration, root, model, and last query.
    - `kori sessions attach <id>` / `kori sessions resume <id>`: attaches to or resumes a session.
    - `kori sessions kill <id>` / `kori sessions stop <id>`: kills a running background session.
  - Add root alias `kori list` pointing to session listing.
  - Support `--detach` / `-d` on root command: launches prompt headlessly in background daemonized/detached process, printing session ID and PID.

### 4. TUI Detach Support (`internal/tui`)
- Add `/detach` (and `/bg`, `/background`) command and `Ctrl+D` prompt handling when running.
- In `internal/tui/command.go`: `/detach` unmounts TUI, keeps any active background run going, and exits with a clear instruction message showing `kori resume <id>`.

## Ordered Steps
1. Update `nacelle` concurrency defaults and tests.
2. Update `kori` settings limits and parallel delegate options for concurrency.
3. Implement `internal/sessions` session metadata, liveness detection, and management functions.
4. Implement `cmd/sessions.go`, `cmd/sessions_list.go`, and root `--detach` / `kori list` wiring.
5. Implement TUI `/detach` command and clean detachment handling.
6. Verify with `go test ./...` in both `nacelle` and `kori`, and run `filet check`.

## Exit Criteria
- Disagreeing tests pass.
- 7+ parallel agents run concurrently.
- `kori sessions` / `kori list` accurately displays active background sessions and their status.
- `kori sessions attach <id>` and `kori resume <id>` resume detached sessions.
- `filet check` reports clean with zero violations.
