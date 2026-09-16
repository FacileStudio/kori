# Plan: Kori + Boite Sandbox Integration - Updated for 100% VM Isolation

## Goal
Enable kori agents to start in boite QEMU VM sandboxes via SSH, so users can easily run kori agents in isolated environments without touching their host environment.

## Why (evidence) — one line
The boite tool already manages QEMU VM instances with SSH access (port 2226, overlay disks, env var passthrough via `~/.boite/state.json`), and kori already supports `-root`, `-bash`, and environment-driven API key configuration. Combining them lets agents run in clean sandboxes that persist across sessions.

## Approach
Create a lightweight wrapper/workflow that bridges kori and boite: when a user invokes kori against a boite instance, the wrapper ensures the VM is running, syncs the project directory into the VM's overlay disk, and starts kori with `-root` pointing inside the VM. SSH is the transport layer — boite instances already expose SSH (e.g., `pingu` on port 2226) with API keys available through `~/.boite/state.json` `pinned_env`. The plan reuses boite's existing provisioning and disk layout; no new runtime deps are added.

CRITICAL: This plan updates the sandbox integration to guarantee 100% VM isolation. The agent MUST execute inside the VM, not on the host. The kori instance must be forced to run with its working directory strictly inside the VM's filesystem via the `-bash` flag and SSH connection with a TTY. This ensures no host environment or filesystem access can leak into the agent.

## Conventions checked:
- **[filet]** — ensure any new Go code passes `filet check`; plan the implementation so it is verifiably clean.
- **[muse]** — not applicable (TUI is internal to kori; no UI changes outside kori).
- **[module-path]** — Go module stays `github.com/FacileStudio/kori`; never fork-inherit bare `module api`.
- **[auth/porte]** — not applicable at this layer; SSH key mgmt is boite's concern.

## Steps (ordered)

1. **CRITICAL ISOLATION GUARD**: Before launching kori inside the VM, execute `ssh -tt -p <port> <name> "whoami; pwd; echo VM_HARNESS=$(date)"` and verify:
   - whoami returns the kori user (not root or host user)
   - pwd shows `/home/yann` or `/home/yann/project`
   - Environment is set in VM (CHECK_VM_HARNESS)
   - No fallback to host environment is possible

2. **STRICT VM EXECUTION**: The kori process MUST run through SSH tunnel with TTY allocation:
   ```
   ssh -tt -p <port> <name> "kori -root /home/yann/project -bash"
   ```
   - No direct host execution allowed
   - All stdin/stdout/stderr go through SSH channel
   - Process must be spawned inside VM, not host

3. **VM FILESYSTEM VERIFICATION**: Before any kori start, verify project directory exists in VM:
   - `ssh -p <port> <name> "ls -la /home/yann/project"`
   - Confirm directory structure matches host (excluding .boite metadata)
   - No host files accessible from VM

4. **SSH CONNECTION LIFETIME**: Maintain SSH connection for entire kori session:
   - No reconnection attempts to different host
   - SSH key fingerprint validation before connection
   - Connection timeout set to prevent hanging

5. **VM-SAFE EXECUTION**: After SSH connection established:
   - Execute `kori -root /home/yann/project -bash` inside VM shell
   - No host binary accessible via PATH from VM
   - VM's Go binary and dependencies only

6. **POST-EXECUTION ISOLATION**: After kori exits:
   - Verify process was spawned from VM: `ps aux | grep kori | grep <vm_user>`
   - Clean up SSH session
   - No host memory/process traces left

7. **OVERLAY DISK SNAPSHOT**: Before cleanup:
   - Prompt user about snapshotting current VM state
   - Snapshot preserves isolated environment for next session
   - No host state pollution

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

## Enhanced Isolation Requirements

### VM Isolation Guard
- Verify SSH connection before agent start
- Check VM user identity and working directory
- Validate VM harness environment variables
- Confirm no host access during session

### SSH Connection Enforcement
- Must use SSH tunnel with TTY (`-tt` flag)
- No direct host binary execution allowed
- Connection must remain stable throughout session
- SSH key fingerprint validation required

### Filesystem Isolation
- Project must exist ONLY in VM filesystem
- Verify `ls -la /home/yann/project` in VM
- Confirm no host files accessible via VM
- Ensure no .gitignore overrides host access

### Process Isolation
- kori MUST spawn from VM, not host
- Verify process tree shows VM origin
- Confirm no host process traces
- Validate environment variables only VM-sourced

### Post-Execution Cleanup
- SSH session must terminate cleanly
- No host memory/processes left behind
- Verify snapshot option presented to user
- Ensure overlay disk preserved correctly

## Risks / unknown unknowns

- SSH port forwarding may require local config; the boite instance's SSH key (`~/.ssh/id_ed25519`) must be available locally
- Overlay disk growth: repeated runs without snapshotting may fill the 20 GB disk; consider a max-age or max-size policy
- API key availability: boite passes `OPENROUTER_API_KEY` etc. via `pinned_env`; if the user's keys are not set in the host environment, the agent inside the VM will fail
- Headless vs TUI: the Bubble Tea TUI may not render well over SSH; a `--headless` flag may be needed for purely CI/worker use cases

## Skip (YAGNI)

- Do NOT add Docker container support in this iteration — boite's QEMU VM model is the starting point; Docker can be added later as an alternative sandbox backend.
- Do NOT implement full boite CLI recreation — only the minimal SSH-connect + project sync logic.
- Do NOT change kori's core TUI or approval flow; the sandbox wrapper sits outside those systems.

## Isolation Validation Checklist

[ ] SSH connection verified before kori start
[ ] VM user identity confirmed (not host user)
[ ] VM working directory validated (`/home/yann/project`)
[ ] VM harness environment variables set
[ ] No host environment variables leaked
[ ] SSH tunnel with TTY established
[ ] kori process spawned from VM only
[ ] VM filesystem exclusively accessible
[ ] No host binaries accessible via PATH
[ ] SSH session maintains stability
[ ] Process tree confirms VM origin
[ ] Host memory/processes clean post-exit
[ ] Snapshot option presented to user
[ ] Overlay disk preserved correctly

---

**Generated**: 2026-09-16 (Updated for 100% VM isolation)
**Source**: brainstorming session combining kori (terminal AI agent) + boite (QEMU VM sandbox manager)