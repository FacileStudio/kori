# Changelog

## [Unreleased]

### Added
- feat(compaction): ratio-based context compaction with a soft/mid/hard ladder, a pinned anchor and a persistent `[state ledger]` that folds new facts into itself instead of re-summarizing (`limits.compaction`, `KORI_COMPACTION_*`)
- feat(compaction): an opt-in TypeSafe System One judge (`limits.compaction.judge`, `TYPESAFE_API_KEY`) that classifies each history block keep/prune/ledger before the ledger is written; off by default because enabling it sends conversation history to a third party
- feat(tui): the status line shows the live context ratio and tier (`↕120k/200k · 0.60 · soft`), and `/status` reports the accumulated ledger's size and the last pass's tier

### Changed
- feat(settings): `limits.compact_at` is unset by default and the ceiling now derives from the compaction ratios against the backend's context window; an absolute value still overrides and `0` still disables compaction

### Fixed
- fix(compaction): a pass can no longer orphan a tool pair the ledger absorbed — the ledger zone claims the replies to its own calls and carries an absorbed block forward, so a later prune cannot split a call from its result and a later rebuild cannot shed what a fold put in (`internal/compaction`)

## [0.70.2] - 2026-09-20

### Fixed
- fix(sandbox): `kori remote` passes `-p` only when a port is configured; a bare `user@host` or `~/.ssh/config` alias keeps ssh's own `Port` instead of being overridden with 22, and `remote.port` defaults to 0 so the group no longer forces one
- fix(sandbox): a preflight host-key failure names the host and the command that resolves it — `ssh` for an unknown key (`ssh-keyscan` when the port is literal), `ssh-keygen -R` for a rotated one — instead of only the ssh exit status, and points a loopback target at `kori sandbox`

## [0.70.1] - 2026-09-20

### Fixed
- fix(sandbox): a boite target starts in the guest's `/workspace` — the directory boite bakes and syncs — not the host directory an instance records; the preflight guard refused every VM session with `project directory "/home/yann" does not exist in target`
- fix(sandbox): a target session with no configured workdir records `target:<name>` as its root, so the banner, system prompt and session header name the target instead of the directory kori was launched from
- fix(sessions): a `target:<name>` root groups and resumes by target identity, never folded into the launch directory
- fix(usage): usage records for a target session carry the target as the project and no branch, instead of the local repository kori ran inside

### Changed
- refactor(sandbox): `kori sandbox list` prints the directory a session's tools run in, rather than the instance's host-side workspace

## [0.70.0] - 2026-09-19

### Added
- feat(sandbox): split sandbox into local boite VMs and kori remote execution backends

### Fixed
- fix: remove duplicate `session.additional_prompt` definition

## [0.69.0] - 2026-09-18

### Added
- feat(settings): `session.additional_prompt` (`KORI_ADDITIONAL_PROMPT`, `-additional-prompt`), appended after the base prompt and skills catalog so a specialist persona layers on without replacing the harness guidance
- feat(settings): profiles carry `limits` and `additional_prompt`, keeping identity and model-aware tuning together while tools and security stay in `~/.kori.yml`

### Fixed
- fix(settings): a profile's identity now beats `~/.kori.yml` however the profile is selected — before, the file's own `profile:` key lost to the file's other keys while `-profile` and `KORI_PROFILE` won, so one profile meant two different things and a scaffolded config could name a profile and never see its backend
- fix(docs): `example.kori.yml` is byte-identical to the first-boot scaffold again, and a test now fails when they drift; the documented `ui.rendering_mode` and `ui.transparent_blocks` defaults follow the code (`tui`, `true`)

## [0.68.0] - 2026-09-18

### Added
- feat(sandbox): host-side agent execution with remote SSH tool remoting for microVM sandboxes
- feat(sandbox): remote implementations of `run_command`, `read_file`, `write_file`, `edit_file`, `list_directory`, `find_files`, and `search_content`
- feat(agent): `RunSessionWithTools` and `RunHeadlessWithTools` for custom tool set mounting

### Changed
- refactor(sandbox): remove guest binary synchronization; guest VMs require zero Kori binaries and zero API keys
- refactor(sandbox): update preflight check to verify SSH connectivity and workspace readiness without requiring guest binaries

## [0.67.1] - 2026-09-18

### Fixed
- fix(tui): isolate model command unit tests from environment API keys

## [0.67.0] - 2026-09-18

### Added
- feat(tui): `/model` command for dynamic profile and model switching mid-session
- feat(settings): `~/.kori/profiles/` directory for reusable provider and reasoning configs
- feat(cmd): `--profile` flag and `KORI_PROFILE` env var for profile selection
- feat(sessions): resume last session by default and generate short 8-char hex session IDs

### Fixed
- fix(tui): preserve slash command autocomplete menu when model picker opens
- fix(provider): add provider catalog inspection and backend construction

## [0.66.0] - 2026-09-17

### Added
- feat(sessions): background session management with `kori sessions` (alias `kori list`), attach/resume, and kill
- feat(sessions): detached execution via `--detach` (`-d`) flag, `/detach` TUI command, and `ctrl+d`
- feat(settings): configurable `limits.max_concurrency` and `limits.max_parallel_agents` in `~/.kori.yml` (default 16)

### Fixed
- fix(parallel): upgrade default subagent concurrency limit to 16 concurrent workers

## [0.65.9] - 2026-09-17

### Added
- feat(sandbox): multi-target sandbox support with ssh and boite backends
- feat(cron): display enabled and crontab installation status in cron list

### Fixed
- fix(editor): use ctrl+g as default keybinding for external prompt editor
- fix(sandbox): direct ssh fallback when boite is not installed

## [0.65.8] - 2026-09-17

### Fixed
- fix(editor): use ctrl+o and alt+e for opening external editor
- fix(sandbox): add list subcommand and styled cobra/fang help for boite VMs
- fix(sandbox): fix guest workdir resolution, shell quoting, and auto binary sync
- fix(cron): improve legacy crontab job line stripping with boundary checks

## [0.65.7] - 2026-09-17

### Fixed
- fix(cron): direct crontab install and uninstall with atomic blocks and log redirection
- fix(cron): automatic PATH augmentation and secret retrieval for unattended runs
- fix(cron): context timeout enforcement and error reporting in delivery logs
- feat(sandbox): isolated boite sandbox environment support

## [0.65.6] - 2026-09-17

### Fixed
- fix(editor): update tests for ctrl+shift+u default and gofmt
- fix: restore sandbox subcommand and suppress linter for unusedfunc
- fix(cmd): remove orphaned newSandboxCmd call
- fix: use markdown for editor prompt file
- chore: rebuild sandbox system per SANDBOX-PLAN.md

## [0.65.5] - 2026-09-16

## [0.65.4] - 2026-09-16

### Fixed
- fix: cron install command installs to crontab
