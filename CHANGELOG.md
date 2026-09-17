# Changelog

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
