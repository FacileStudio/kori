# Kori Configuration Plan

Status: implemented on main (unreleased), revised 2026-09-18.
This revision replaces the first draft, whose executive summary described a sandbox/remote
configuration split the document never contained, whose Problem Statement argued against its own
recommendation, and whose open questions the code had already answered. That sandbox design is its
own document: [SANDBOX-PLAN.md](SANDBOX-PLAN.md).

## Goal

Close the gap between the built-in harness prompt and a personal one, and let a reusable identity
carry the settings that belong to the model rather than to the checkout. Three changes, all
shipped together because they only make sense together:

1. `session.additional_prompt` — text appended on top of whatever base prompt is active.
2. Profiles carry `limits` and `additional_prompt` beside provider and reasoning.
3. One precedence chain that says where a profile sits, whichever layer names it.

## Problem and evidence

- **`system_prompt` replaces, with no middle ground.** It is a string overwrite, so setting it
  costs kori's own tool descriptions, how-to guidance and safety scope. A user who wants one
  standing rule in a specialist session had to choose between the rule and the harness.
- **A persona cannot live in a skill.** Skills use progressive disclosure: only `name` and
  `description` reach the system prompt, and the body is read on demand. Anything that must steer
  every turn has to be in the prompt itself.
- **The model-aware settings had nowhere to live.** `~/.kori/profiles/<name>.yml` carried provider
  and reasoning only, while the two settings that genuinely depend on which model is running — the
  iteration ceiling and the compaction threshold — could only be set for the whole checkout.
- **Profile resolution was inconsistent.** A profile was applied at whichever layer named it, so
  the file's own `profile:` key lost to the file's other keys (`limits`, `provider`) while
  `-profile` and `KORI_PROFILE` beat them. The same profile meant two different things depending
  on how it was selected, and the scaffolded `~/.kori.yml` always names a backend, so a
  file-named profile could never move one.

## Decisions

### 1. `additional_prompt` composes, `system_prompt` still replaces

| `system_prompt` | `additional_prompt` | result |
|---|---|---|
| unset | unset | the built-in harness prompt alone |
| unset | set | harness prompt, then the extra text |
| set | unset | the replacement, verbatim — unchanged behaviour |
| set | set | the replacement, then the extra text |

Nothing about `system_prompt` changes. The two are not alternatives; the useful case is replacing
the base *and* still layering a rule over it.

### 2. Appended last: after the environment block, the project context and the skills catalog

In a transformer prompt later text carries more weight, so a directive placed at the end exerts
the most influence over the guidance it sits on. It also matches how `CLAUDE.md` and `AGENTS.md`
already work — appended at the end as instructions layered over the base.

Rejected alternative: right after the base prompt, before the session environment block. Reads as
"identity declared up front", but puts the steering text under the environment facts rather than
over them.

### 3. A single string, and one persona at a time

Layers **replace** each other rather than stacking: the highest layer that sets `additional_prompt`
is the one that reaches the prompt. A repeatable list would let a profile, a file and a flag
compose fragments, but multiplying personas is the failure mode this setting exists to avoid, and
the merge rule would have to become list-aware for one key only.

### 4. Profile = identity, `~/.kori.yml` = behaviour

| layer | carries |
|---|---|
| `~/.kori/profiles/<name>.yml` | `provider`, `reasoning`, `limits`, `additional_prompt` |
| `~/.kori.yml` | `tools`, `security`, `sources`, `gates`, `ui`, `session.root`, `session.system_prompt` |

Tools and security are about *this machine and this checkout*; a profile that carried them would
have to be edited per host, which is the thing profiles exist to avoid. `limits` is the opposite:
an iteration ceiling and a compaction threshold are properties of the model, not of the clone.

### 5. Precedence: `defaults < ~/.kori.yml < profile < environment < flags`

A profile is a layer of its own, applied in the same place whichever layer names it — the file's
`profile:` key, `KORI_PROFILE`, or `-profile` (for the name itself: flag beats environment beats
file). The environment and the flags still win over it, field by field, so `KORI_MAX_ITERATIONS`
and `-max-iterations` beat a profile's limit and a machine's own settings survive a profile the
whole suite shares. `-no-config` drops the file layer and nothing else: a profile named by the
flag or the environment still applies.

This is the one decision the first cut got wrong. It applied the profile at the layer that named
it, which made `profile: fast` in the file lose to the file's own `provider.backend` — so a
scaffolded config could name a profile and never see its backend.

### 6. A mid-session `/model <profile>` applies provider and reasoning only

Limits and persona take effect at launch. The system prompt is built once at boot and the run's
ceiling is read then; rebuilding either mid-session would silently change the rules of a run the
user is already in. The picker and the switch message should say so, which is a UI item, not a
settings one.

## Where each setting can be set

| setting | `~/.kori.yml` | profile | env | flag |
|---|---|---|---|---|
| `provider.backend` / `model` | yes | yes | `KORI_BACKEND`, `KORI_MODEL` | `-backend`, `-model` |
| `reasoning.effort` / `thinking` / `budget` | yes | yes | `KORI_EFFORT`, `KORI_THINKING`, `KORI_REASONING_BUDGET` | `-effort`, `-thinking`, `-reasoning-budget` |
| `limits.*` | yes | yes | `KORI_MAX_ITERATIONS`, `KORI_COMPACT_AT`, … | `-max-iterations`, `-compact-at`, … |
| `session.additional_prompt` | yes | yes | `KORI_ADDITIONAL_PROMPT` | `-additional-prompt` |
| `session.system_prompt` | yes | no | `KORI_SYSTEM_PROMPT` | `-system-prompt` |

## Rejected

- **Tools, security, sources and gates in profiles.** Option B of the first draft. It would make a
  profile host-specific and a profile file a second `~/.kori.yml` to keep in sync.
- **A repeatable persona list** (see 3).
- **`kori profile list` / `kori profile validate`.** The `/model` picker already lists profiles and
  the strict decoder already refuses a bad key with its line number; a second surface would be a
  second thing to keep honest.
- **Re-resolving limits on `/model`** (see 6).
- **Sandbox and remote targets in this document.** They are their own design, already written:
  [SANDBOX-PLAN.md](SANDBOX-PLAN.md).

## Documentation contract

- [docs/configuration.md](docs/configuration.md) is the reference: every setting, the precedence
  chain, and the traps. It carries the profile example with `limits` and `additional_prompt`.
- `example.kori.yml` is byte-for-byte the text `settings.Template` writes on first boot, and
  `TestExampleConfigIsTheScaffoldTemplate` fails when the two drift. They had drifted by 53 lines
  before this pass — `rendering_mode`, `transparent_blocks`, the editor keys, the sandbox targets —
  which is why the promise is a test and not a comment. A template line can also never contain a
  backtick: the constant is a Go raw string, and a stray one silently ends it.

## Verification

The suite is the spec for the rules above:

- `internal/settings/prompt_test.go` — `additional_prompt` through the whole chain; a replacement
  base and a persona holding at once; a profile's persona beating the file's and losing to the
  environment's.
- `internal/settings/profiles_test.go` — a profile carries limits and persona; the profile beats
  the file, the environment beats the profile; the file, `KORI_PROFILE` and `-profile` all resolve
  the profile the same way.
- `internal/agent/additional_prompt_test.go` — the persona is the last thing in the prompt, over a
  default and over a replaced base.
- `internal/settings/scaffold_test.go` — the scaffold round-trips to the defaults, and the example
  file matches it byte for byte.
- `filet check .` and `go test ./...` clean.

## Still open

1. **No layer can unset a value.** An empty string reads as "unmentioned", so a persona or a limit
   set by a profile can be overridden but never cleared from `~/.kori.yml`. If a `null` spelling is
   wanted, it has to be one decision across every string setting, not one for the new key.
2. **Should `/model <profile>` say what it does not apply?** The behaviour is documented, the
   picker still describes a profile as `backend/model`.
3. **Is a persona meant to reach delegated subagents?** They share the parent's system prompt, so
   today it does. That is probably right — a persona that stopped applying to delegated work would
   be a surprising exception — but it is unstated.
