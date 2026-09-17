# Changelog

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

## [Unreleased]
