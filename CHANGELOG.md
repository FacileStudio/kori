# Changelog

## [Unreleased]

### Added
- feat(compaction): a labeled-example calibration harness for the judge — one batched call over a corpus of labeled history blocks, re-scored across a sweep of prune thresholds, printing where the model's confidence and its accuracy part company (`TYPESAFE_API_KEY=... go test ./internal/compaction -run JudgeCalibration -v`; skipped without a key, so CI never pays for it)

### Changed
- feat(compaction): `limits.compaction.judge.prune_threshold` defaults to `0.75`, not `0.85`. The old number had never been measured against the model; the harness shows it recovering 17% of the blocks a careful operator would drop where 0.75 recovers 50%, at the same zero false prunes, and lifts agreement with the labels from 65% to 76%

### Fixed
- fix(compaction): the pre-send guard did nothing at all on a backend that cannot count tokens. It read `if count, err := CountTokens(...); err == nil && count > trigger` — and the OpenAI-compatible runner answers with `*Unsupported`, so the error branch skipped the whole check while the post-turn trigger kept running and compaction still looked healthy. A conversation already past the ceiling was sent anyway and the overshoot was absorbed into the next turn instead of prevented; the guard now falls back to the last usage-reported size, so the two automatic triggers agree on what the context costs (`internal/tui`)
- fix(compaction): a usage event reporting no input tokens erased `m.size`, the only measure both automatic triggers read, silently standing compaction down until a real usage arrived — a backend that reports usage on some events but not others was enough to switch it off. A zero is now read as "unknown" and the last real reading kept (`internal/tui`)
- fix(compaction): the judge's choice questions were rejected by the live endpoint — TypeSafe takes a choice's `criteria` as an object keyed by option, not a list, so every classification was a 422 and fell back to the mask. The judge had never successfully classified anything, and no test could see it because they all talk to a stub (`internal/compaction`)
- fix(compaction): every question in a batch was identical, so the model had no way to tell which block it was being asked about and answered the same way for all of them; each question now names its own block in its instructions (`internal/compaction`)
- fix(compaction): the criteria described a dead end under both `ledger` and `prune`, leaving the model no confident answer — the distribution flattened under the confidence floor and no verdict was ever acted on. The three descriptions are now disjoint, and confidence rises from 0.2–0.3 to 0.8–0.9
- fix(compaction): the soft tier tombstoned however little it found, rewriting the prompt-cache prefix for a few hundred bytes; it now stands down below `MinCleared`, and `DroppableBytes` is the pre-flight a caller measures a pass with before paying for the cache it invalidates
- fix(compaction): the judge's request was bounded in block count but not in bytes, so a handful of large tool results produced a multi-hundred-KB call to a small decision model; each block is now cut to `maxBlockText` and the batch to `defaultMaxState`, folding the oldest blocks that do not fit rather than asking about none of them
- fix(compaction): a tombstoned result's placeholder never closed its bracket — `[dropped 40000 bytes. Re-run the tool…` where the documented shape is `[dropped N bytes]`
- fix(compaction): the versioned model id that answered was decoded and thrown away, and the setting defaults to the drifting `jev-latest` alias; `/status` now names the model that answered and what it billed, so the setting can be pinned to the build the thresholds were tuned against (`internal/compaction`, `internal/tui`)
- refactor(compaction): a pass runs on a snapshot taken on the update loop instead of reading the model off its own goroutine, so its read safety is local rather than an invariant spread across every command that might mutate the session mid-pass (`internal/tui`)

### Removed
- chore(compaction): the dead `alignedEvictCut` wrapper, whose only caller in the repository was its own test, and the superseded `compaction-upgrade.md` spec at the repository root

## [0.71.0] - 2026-09-21

### Added
- feat(compaction): ratio-based context compaction with a soft/mid/hard ladder, a pinned anchor and a persistent `[state ledger]` that folds new facts into itself instead of re-summarizing (`limits.compaction`, `KORI_COMPACTION_*`)
- feat(compaction): an opt-in TypeSafe System One judge (`limits.compaction.judge`, `TYPESAFE_API_KEY`) that classifies each history block keep/prune/ledger before the ledger is written; off by default because enabling it sends conversation history to a third party
- feat(tui): the status line shows the live context ratio and tier (`↕120k/200k · 0.60 · soft`), and `/status` reports the accumulated ledger's size, the last pass's tier, and whether the judge is on — the one setting that sends history off the machine is otherwise invisible once enabled

### Changed
- feat(settings): `limits.compact_at` is unset by default and the ceiling now derives from the compaction ratios against the backend's context window; an absolute value still overrides and `0` still disables compaction
- feat(settings): `limits.compaction` ratios and `judge.prune_threshold` are validated at load — a ratio outside `(0,1]`, a ladder with its rungs out of order, or an unusable prune threshold is now a startup error instead of a trigger that silently never fires

### Fixed
- fix(compaction): a pass can no longer orphan a tool pair the ledger absorbed — the ledger zone claims the replies to its own calls and carries an absorbed block forward, so a later prune cannot split a call from its result and a later rebuild cannot shed what a fold put in (`internal/compaction`)
- fix(compaction): a judged pass that comes back with an empty summary masks instead of folding away the turns the judge tagged for the ledger, so a provider that streams no text can no longer drop history silently and call it a summary
- fix(compaction): a rebuild heavier than the conversation it replaces is refused and the context left unchanged, so a pass never grows the context it exists to shrink (folding a two-byte turn can no longer install a verbose ledger)
- fix(compaction): the pinned head is extended over the replies answering the calls it carries, and a pass that folds nothing still stands a ledger between the head and the active window — the two cases where the assembly could merge a kept turn into the anchor or emit two messages of one role
- fix(jev): a transport error (a dropped connection or a per-attempt timeout) is retried like a 429/529, bounded by the caller's context rather than spending the attempt budget on requests that are already out of time
- fix(compaction): the soft tier tombstones tool results only — every backend drops a `Reasoning` block when it builds a request, so stubbing one freed no context while taking the chain of thought out of a transcript the reader can still scroll back to, and the pass reported the saved bytes as if it had bought headroom
- fix(compaction): a judge built without a prune threshold falls back to the shipped `0.85` instead of pruning on any probability at all, and a block the model kept can no longer be dropped by a threshold of zero
- fix(compaction): `soft_ratio: 0` no longer turns compaction off while the tier ladder still reports it on — the derived ceiling falls back to `75000` instead of zero, which every trigger reads as "disabled", and the setting is refused at load
- fix(compaction): a pass refuses a plan that no longer covers the conversation instead of indexing past the end of it, so a `/clear` or `/resume` landing mid-pass cannot take the session down with it, and the mask fallback refuses one too rather than promising a fallback it did not run
- fix(compaction): a tool call's own arguments (abbreviated) reach the judge, which was classifying every block from the tool's name alone
- fix(jev): a gateway's 502/503/504 is retried like a 429/529, and a zero-value client makes one attempt instead of returning an empty response with no error

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
