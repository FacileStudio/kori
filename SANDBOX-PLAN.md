# Plan: Kori + Boite Sandbox Integration

## Goal
Enable kori agents to start in boite QEMU VM sandboxes via SSH, so users can easily run kori agents in isolated environments without touching their host environment.

## Why (evidence) — one line
The boite tool already manages QEMU VM instances with SSH access (port 2226, overlay disks, env var passthrough via `~/.boite/state.json`), and kori already supports `-root`, `-bash`, and environment-driven API key configuration. Combining them lets agents run in clean sandboxes that persist across sessions.

## Approach
Create a lightweight wrapper/workflow that bridges kori and boite: when a user invokes kori against a boite instance, the wrapper ensures the VM is running, syncs the project directory into the VM's overlay disk, and starts kori with `-root` pointing inside the VM. SSH is the transport layer — boite instances already expose SSH (e.g., `pingu` on port 2226) with API keys available through `~/.boite/state.json` `pinned_env`. The plan reuses boite's existing provisioning and disk layout; no new runtime deps are added.

Conventions checked:
- **[filet]** — ensure any new Go code passes `filet check`; plan the implementation so it is verifiably clean.
- **[muse]** — not applicable (TUI is internal to kori; no UI changes outside kori).
- **[module-path]** — Go module stays `github.com/FacileStudio/kori`; never fork-inherit bare `module api`.
- **[auth/porte]** — not applicable at this layer; SSH key mgmt is boite's concern.

## Steps (ordered)

1. **`/home/yann/Code/Facile/kori/cmd/kori/root.go`** — add a new CLI subcommand `kori sandbox <instance-name>` that:
   - Resolves the boite instance config from `~/.boite/instances/<instance-name>/state.json`
   - Verifies the VM is running (`boite up <name>` equivalent, or `ssh -p <port> <name> true`)
   - Syncs the host project directory into the VM via overlay mount or rsync (the VM's workspace is already `/home/yann` in `state.json`)
   - Starts kori inside the VM context: `kori -root /home/yann/project -bash` (or headless if preferred)
   - Exits and optionally snapshots the overlay disk on cleanup

2. **`/home/yann/Code/Facile/kori/internal/settings/settings.go`** — add an optional `sandbox` config section that, when set, overrides `root` and injects boite‑specific env vars (SSH port, instance name) into the agent's environment. This keeps the config declarative rather than imperative.

3. **`/home/yann/Code/Facile/kori/example.kori.yml`** — add a `sandbox` entry showing how to configure a default instance, e.g.:
   ```yaml
   sandbox:
     instance: pingu
     project: .
   ```

4. **`/home/yann/Code/Facile/kori/scripts/boite-sync.sh`** *(new file)* — optional helper that rsyncs the local project directory into the running boite VM's `/home/yann/project` path. Can be called manually or from the new `kori sandbox` subcommand.

## Files to Modify / New

- **Modified**: `cmd/kori/root.go` — add `sandbox` subcommand
- **Modified**: `internal/settings/settings.go` — add `sandbox` config section
- **Modified**: `example.kori.yml` — add sandbox configuration example
- **New**: `scripts/boite-sync.sh` — rsync helper for project → VM sync

## Exit criteria

- `kori sandbox pingu` successfully starts a kori session inside the running `pingu` boite VM
- The agent's `-root` resolves to a project directory that exists inside the VM (synced from host)
- `filet check` passes on all modified Go files with no new failures
- SSH connection to the boite instance is verified before kori starts
- On `kori` exit, the user is prompted whether to snapshot the overlay disk (preserving state for next time)

## Risks / unknown unknowns

- SSH port forwarding may require local config; the boite instance's SSH key (`~/.ssh/id_ed25519`) must be available locally
- Overlay disk growth: repeated runs without snapshotting may fill the 20 GB disk; consider a max-age or max-size policy
- API key availability: boite passes `OPENROUTER_API_KEY` etc. via `pinned_env`; if the user's keys are not set in the host environment, the agent inside the VM will fail
- Headless vs TUI: the Bubble Tea TUI may not render well over SSH; a `--headless` flag may be needed for purely CI/worker use cases

## Skip (YAGNI)

- Do NOT add Docker container support in this iteration — boite's QEMU VM model is the starting point; Docker can be added later as an alternative sandbox backend.
- Do NOT implement full boite CLI recreation — only the minimal SSH-connect + project sync logic.
- Do NOT change kori's core TUI or approval flow; the sandbox wrapper sits outside those systems.

---

**Generated**: 2026-09-16
**Source**: brainstorming session combining kori (terminal AI agent) + boite (QEMU VM sandbox manager)