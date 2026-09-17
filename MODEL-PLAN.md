# Plan: Dynamic Model Switching and Profiles in Kori

## Goal

Add a new `/model` slash command to Kori enabling interactive mid-session model and profile switching, paired with a declarative `~/.kori/profiles/` directory for reusable model configurations.

## Architecture & Design

### 1. Profile Store (`~/.kori/profiles/*.yml`)

Profiles represent complete provider configurations stored as YAML files under `~/.kori/profiles/`.

- **Schema**:
  ```yaml
  name: fast
  provider:
    backend: google
    model: gemini-2.5-flash
    base_url: ""
    api_key: ""
  reasoning:
    effort: low
    thinking: true
    budget: 2048
  ```
- **Discovery**: Automatically scans `~/.kori/profiles/` on startup and on `/model` invocation. Any new `.yml` file placed there is discovered immediately without restarting Kori.
- **Config integration**:
  - `~/.kori.yml` gains an optional `profile: <name>` field.
  - `-profile <name>` CLI flag and `KORI_PROFILE` environment variable.
  - When specified, the profile serves as the baseline configuration for `provider` and `reasoning`, overridable by explicit flags/env vars.

### 2. Provider Catalog & Detection

- Inspects environment variables (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`, `OPENROUTER_API_KEY`) and existing config credentials to detect active providers.
- Maintains a curated catalog of premier coding/reasoning models per backend.
- Provides fallback model listing when offline or without external network calls to maintain instant TUI responsiveness.

### 3. Interactive `/model` Slash Command

- **Autocomplete**: Register `/model` in the slash command dropdown with syntax `/model [profile|model]`.
- **Argument invocation (`/model <name>`)**:
  - Exact match against profile names in `~/.kori/profiles/`.
  - Fallback match against known model names or provider-qualified slugs (`openai/gpt-5.4`, `anthropic/claude-opus-5`).
  - Hot-swaps the active agent immediately and announces the change.
- **No-argument invocation (`/model`)**:
  - Opens the interactive Bubble Tea selection menu populated with:
    1. Discovered profiles (`~/.kori/profiles/*.yml`).
    2. Detected providers and their curated models.
  - Selecting an entry applies the profile/model instantly for the next turn.

### 4. Mid-Session Re-instantiation Mechanics

1. Preserves `m.conversation` (the message history slice) in `tui.Model`.
2. Assembles a new `nacelle.Backend` and `nacelle.Config` with the updated provider and reasoning settings.
3. Constructs a new `*nacelle.Agent` via `nacelle.New(...)` and updates `m.agent`.
4. Records a model change entry into the active session JSONL log (`~/.kori/sessions/*.jsonl`) for accurate token and cost tracking.
5. Updates the active model indicator in the status bar and banner.

## File Changes

1. `internal/settings/profiles.go` & `profiles_test.go`:
   - Profile struct, `ProfilesDir()`, `LoadProfiles()`, `LoadProfile()`.
2. `internal/settings/config.go`, `flags.go`, `env.go`, `merge.go`:
   - Support `profile` field in `~/.kori.yml`, `-profile` flag, and `KORI_PROFILE` env var.
3. `internal/agent/catalog.go` & `catalog_test.go`:
   - Curated model catalog and active provider detection.
4. `internal/agent/agent.go`:
   - Support re-building an agent on configuration change.
5. `internal/tui/command.go`, `command_model.go`, `command_model_test.go`:
   - Handler for `/model` and `/model <name>`.
   - Interaction with `internal/menu` for interactive model/profile picker.
6. Documentation in `docs/configuration.md`:
   - Document `~/.kori/profiles/`, `profile:` in `~/.kori.yml`, and `/model` slash command.

## Verification & Gates

- `filet check .` must pass with 0 errors and 0 warnings.
- `go test ./...` all unit and integration tests green.
- Manual verification of profile discovery and hot-swapping.
