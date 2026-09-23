# Plan: robust compaction & automatic context management (JEV-assisted)

**Status:** Phases 0–4 implemented 2026-09-21 — see the notes at the end of each for the
deviations. Three review rounds followed the same day, recorded at the end as *Review
follow-ups*: the first closed four holes, the second six, the third seven (answered by the
calibration harness the earlier rounds' claims were checked with). Two sections dated
2026-09-22 — *Tail budget and reserve* and *Ledger growth and duplication* — close what those
reviews left open. Written 2026-09-21 for a cold-start handoff.
**Audience:** an implementer agent with no prior conversation. Everything needed is below;
nothing here depends on the conversation that produced it.
**Repos:** `github.com/FacileStudio/kori` (this repo, the harness) and its pinned dependency
`github.com/FacileStudio/nacelle` (the agent SDK). **No nacelle change is required**; if one turns
out to be, it ships as a core tag then a TUI adoption commit (nacelle's standing release rule).

**Definition of done, one line:** kori stops compacting at a single absolute token count and
decides instead, per history block and with a calibrated confidence, what to keep, prune, or fold
into a persistent state ledger — while the original task and the system prompt survive every pass
byte-for-byte and every `tool_use`/`tool_result` pair stays valid.

---

## 0. How to use this document

1. Read §1 (current state) and §2 (gaps) once, so the change is grounded in the code as it is.
2. Read §3–§5 (target design, JEV facts, locked decisions). Do not relitigate §5.
3. Execute §6 phase by phase. Each step names exact files, the symbol to add or move, and the test
   that proves it. A phase's **Exit** line is the gate; do not start the next phase before it holds.
4. Run the gate in §7 before declaring any phase done. `filet` is at `failOn: info` — a warning is
   a failure here.
5. If reality contradicts this document, trust the code and update this file in the same commit.

---

## 1. Current state (read this before touching anything)

### 1.1 Where compaction lives

Strategy lives in `internal/compaction`; the TypeSafe client in `internal/jev`; `internal/tui` keeps
only the wiring. This is the **implemented** layout (as of Phase 4); the pre-implementation state this
plan was written against is gone.

| File | Owns |
|---|---|
| `internal/compaction/policy.go` | `Ratios`, `Policy`, `Tier`, `Zone`, `Span`, `Plan`, `Tier`/`Trigger`/`reaches`, `activeStart`, `ledgerIndex` |
| `internal/compaction/spans.go` | `Section`, `HistoryMessages`, `HistoryRange`, `LedgerText`, `LedgerEnd`, `answersAny`, `clamp` |
| `internal/compaction/blocks.go` | `Block`, `Blocks`, atomic pairing (`blockEnd`/`opensToolPair`/`foldOrphanResults`), `renderBlock` |
| `internal/compaction/pair.go` | `AlignedCut`, `Covers`, `toolResultIDs`, `toolCallIDs` |
| `internal/compaction/ledger.go` | `Sentinel`, `IsLedger`, `Body`, `BuildLedger`, `ExtraParts`, `opposite` |
| `internal/compaction/ledgerbody.go` | `MaxLedgerTokens`, `MergeLedger`, `LedgerOverBudget`, `NextLedger`, `newLedgerMessage` |
| `internal/compaction/ledgersection.go` | `section`, `sections`, `sectionNames`/`isHead`, `mergeSections`, `appendUnique`, `render`, `normalize` |
| `internal/compaction/ledgerident.go` | `Identifiers`, `MissingIdentifiers`, `isPathLike`, `identTrim`, `dedupe` |
| `internal/compaction/micro.go` | the package's numbers (`MinResult`/`MinCleared`, `DroppedNotice`, `maxBlockInput`/`maxBlockText`), `EstTokens`, `Bytes`/`MsgBytes`/`PartBytes`, `measure`, `callText`/`clampText` |
| `internal/compaction/tombstone.go` | `MicroStats`, `droppable`, `DroppableBytes`/`spanDroppableBytes`, `Tombstone`, `tombstoneResults` |
| `internal/compaction/macro.go` | `Fold`, `Classify`, `JudgeRequest`, `GoalText`, `foldVerdicts` |
| `internal/compaction/judge.go` | `Decision`, `Verdict`, `Judge`, `Reporter`/`Answer`, `JudgeConfig`, `ConfidenceFloor`, `decide`, `keepAll` |
| `internal/compaction/jevjudge.go` | `NewJevJudge` (the System One adapter), `overflow`, `record`/`LastAnswer`, `defaultMaxState` |
| `internal/compaction/jevquestion.go` | `state`, `questions`, `choiceQuestion` |
| `internal/compaction/apply.go` | `Stats`, `Apply`, `assemble`, `unchanged`, `ledgerCarry`, `following`, `ledgerRole`, `alternateFrom` |
| `internal/jev/` | the HTTP client (`client.go`), wire types (`types.go`), typed errors (`errors.go`) |
| `internal/agent/compact.go` | `Budget`, `ResolveBudget` (replaces `resolveCompactAt` and its `==100000` hack) |
| `internal/agent/policy.go` | `Policy`, `CompactionConfig`, `Judge` — settings → the TUI's policy/judge |
| `internal/settings/compaction.go` | `Compaction`, `Judge`, their merge, `compactionEnv` |
| `internal/tui/compact.go` | `beginCompaction`, `runCompaction`, `settleCompaction`, `installFold`, `waitForCompact`, `resolvedPolicy`, `plan` |
| `internal/tui/compact_light.go` | `compactTiered`, `softPass`, `maskOnlyPass`, `evictionCanLandUnder`, `summarizer`, `thrashed`/`checkThrash`, `thrashLimit=3` |
| `internal/tui/compact_mask.go` | the package aliases and `maskHistory`/`applyMaskFallback` |
| `internal/tui/compact_summary.go` | `compacted`, `compactOutcome`, `installs`, `compactReport`, `summarizeInto`, `compactTimeout=120s`, `compactJudgeTimeout=30s` |
| `internal/tui/compact_prompt.go` | `compactPrompt`, `compactAskWith`, `withAsk`, `compactSystem`/`compactAsk`/`keepAsk`/`consolidateAsk` |
| `internal/tui/overflow.go` | `armRecovery`, `recoverOverflow`, `retryRun` — context-overflow recovery |
| `internal/overflow/` | `Detect` — the provider context-length refusal classifier |
| `internal/tui/settle_tasks.go` | `taskReminder`, `tasksUnfinished`, `tasksSummary` |
| `internal/tui/compact_idle.go` | `shouldCompactIdle`, `maybeCompactIdle`, `compactCmd` (`/compact`) |
| `internal/tui/compact_pass.go` | `compactPass` and `Model.pass` — the snapshot one pass goroutine runs on |
| `internal/tui/account.go` | the `account` struct (`size`/`trimmed`/`compactBegan`) and `sized` |
| `internal/tui/context_line.go` | `contextLoad`, `Model.compactionLines`, `judgeModelLine`, `lastAnswer` — the footer and `/status` |
| `internal/tui/types.go` | `policy`, `judge`, `compacting`, `compactAt`, `last`, `thrashCount` on `transcript` |
| `internal/compaction/calibration_test.go` | the opt-in calibration harness: the corpus loader and the threshold sweep |
| `internal/compaction/calibration_{score,report}_test.go` | classification, scoring and the report the harness prints |
| `internal/compaction/testdata/judge_labels.json` | the labeled corpus — the file a human reviews and extends |
| tests | `internal/compaction/*_test.go`, `internal/jev/client_test.go`, `internal/tui/compact*_test.go`, `internal/tui/context_line_test.go`, `internal/agent/compact_test.go`, `internal/settings/compaction_test.go` |

### 1.2 The three triggers, as wired today

- **Pre-send** — `internal/tui/compact_idle.go` (`compactBeforeSend`, called from `send` in
  `run.go`): the measure is `m.agent.CountTokens(ctx, m.conversation)` when the backend offers one
  and the last usage-reported `m.size` when it does not; if
  `size > m.policy.Trigger()` and not thrashed, `m.compactTiered(ctx)` runs. The
  fallback is load-bearing: `CountTokens` returns an `*Unsupported` error on the
  OpenAI-compatible runner, and a guard written as `err == nil && …` silently does nothing there.
- **Post-turn** — `internal/tui/settle.go` calls `maybeCompactIdle()`; it fires only when idle, no
  queued line, not compacting, not thrashed, and `m.compactAt > 0 && m.size > m.policy.Trigger()`.
- **Manual** — `/compact` → `compactCmd` in `internal/tui/compact_idle.go` (registered in
  `command.go`).
- **Recovery** — `internal/tui/overflow.go`, called from `consume`/`settle`. The ladder is measured
  against an estimate, so a provider can still refuse a request: `armRecovery` holds a
  context-length rejection (`internal/overflow.Detect`) instead of printing it, and `settle` answers
  it with `recoverOverflow` → a *forced hard* pass → `retryRun`, which starts the run again on the
  compacted conversation. Once per turn, and not at all when `compact_at` is 0.

### 1.3 The pass, as it behaves today

`compactTiered` selects by `Policy.Tier(m.size)`. **Soft** runs `softPass`: a synchronous
`compaction.Tombstone` of the history spans, no model call, no goroutine. **Mid/Hard** run
`beginCompaction`, which falls back to `maskOnlyPass` (no model call, reports that the cost sits in
the kept tail) when `evictionCanLandUnder` is false, and otherwise starts `runCompaction`. That
goroutine runs `compaction.Classify` — one batched judge call when the judge is on, whole-history
fold when it is off — feeds only the ledger-tagged blocks to the tool-free summarizer
(`summarizer` + `compactPrompt` + `summarizeInto`, 120 s deadline; the judge is bounded at 30 s),
and sends back a `compactOutcome`. `settleCompaction` installs it with `compaction.Apply` (pinned
anchor + rebuilt one-message ledger + surviving blocks + active window), falls back to the
synchronous mask on any error, then runs `checkThrash`. `checkThrash` counts consecutive passes that
fail to land under the threshold; at 3 the automatic triggers stand down. The `m.size` debit the
mask applies is an estimate the next `sized()` overwrites (G6) — and a `sized()` that reports no
input at all is ignored rather than written through, because `m.size` is the only measure both
automatic triggers read and a zero would erase it.

Two refinements landed after Phase 4. A pass whose ledger body is past `MaxLedgerTokens` is a
*consolidating* pass: it asks for one rewritten block (`consolidateAsk`) and `Apply` replaces the
body rather than merging into it, provided the rewrite still names every identifier the old body
named. And a run the provider refuses for context length compacts once and retries, as above — the
retry is the only caller of `beginCompaction` that forces the hard tier, because a size the ladder
would have caught cannot be what the provider just refused.

`compaction.Tombstone` replaces oversized history `ToolResult`s (≥ 1024 bytes) with
`[dropped N bytes]` stubs that say to re-run the tool, keeping `ID`/`Name`; it is idempotent and mutates the conversation in
place. Reasoning is deliberately not tombstoned — see the second round of review follow-ups. `LedgerEnd` extends a ledger span over the replies answering the calls the
ledger carries, and `Apply`'s `ledgerCarry` re-emits them, so the ledger zone can be more than one
message — see the Phase 4 note.

### 1.4 Ground truth from nacelle (do not duplicate it)

- `nacelle.Trim(conversation, keep) (kept []Message, dropped int)` — boundary-safe truncation; never
  returns a slice whose first message opens with a `ToolResult`. `nacelle/trim.go`. It exists and is
  not called: the hard tier already lands at anchor+ledger+active through `Apply` (see the review
  follow-ups), so nothing is left for it to trim.
- `Agent.CountTokens(ctx, conversation) (int64, error)` — real request size (system + tools + MCP +
  messages). `nacelle/stream.go`. **Not every backend implements it**: `Capabilities.TokenCounting
  == false` means the call returns an `*Unsupported` error rather than a guess, so anything
  branching on it must have an answer for the error and not merely for the count.
- `Agent.CompactConversation(...)` + `BeforeCompact`/`AfterCompact` hooks — binary-search trim to a
  budget, hooks fire with pre/post token counts as strings. `nacelle/stream.go`, `nacelle/hooks.go`.
- `Backend.Capabilities().ContextWindow int64` — total window; **may be 0** for some backends.
- `nacelle.Message{Role, Parts}` with parts `Text`, `Reasoning`, `ToolCall{ID,Name,Input,Finished}`,
  `ToolResult{ID,Name,Result,Failed}`, `Finish`. `nacelle/message.go`. Roles are only
  `RoleUser`/`RoleAssistant`; a `ToolResult` rides on a user message.
- **Strategy stays in kori.** nacelle's roadmap says so explicitly ("Compaction in `tui/`, not the
  core", item 17) and `trim.go` repeats it. Do not move summarization into nacelle.

---

## 2. Problem statement (the gaps this plan closes)

| # | Gap | Evidence |
|---|---|---|
| G1 | Trigger is one absolute token count; ratio awareness is a magic value. | `defaults.go:7` (`75000`); `agent/compact.go:5-10` (`== 100000`) |
| G2 | Micro-cleanup is neither continuous nor turn-aware — it runs only inside a triggered pass. | `compact_mask.go:81-105` |
| G3 | The first user turn is evictable; the root task can be summarized away. | `compact.go:122-129`, `compact_summary.go:52-65` |
| G4 | The summary replaces the middle; a second pass re-summarizes it → summary-of-summary decay. | `compact_summary.go:52-96` |
| G5 | Pairing safety covers only the one computed cut; there is no per-block delete. | `internal/tui/compact_pair.go:11-30`, since deleted — the cut now lives in `internal/compaction/pair.go` |
| G6 | Threshold decisions mix authoritative `CountTokens` with a `bytes/4` debit, so local state drifts until the next `sized()`. | `run.go:126`, `compact_mask.go:99-104`, `compact.go:113-115` |
| G7 | No semantic classification exists at all: whole-middle-or-nothing. | the whole `compact*` set |

---

## 3. Target architecture

### 3.1 Four zones

The conversation is partitioned **by index** into spans, in this order:

1. **Anchor** — the pinned head. The **system prompt is already outside the conversation**
   (`nacelle.Config.System`; `m.delegate.System`) — nothing to move, only to state as an invariant.
   The pinned in-conversation anchor is the **first user turn**, i.e. `anchor_messages` messages
   from the front. Never rewritten, never summarized, never pruned. A head that carries a tool call
   is extended over the reply answering it (`anchorEnd`), so the pinned boundary never splits a pair.
2. **Ledger** — exactly one message carrying the `[state ledger]` sentinel, which ends in the
   private-use codepoint U+E000. That codepoint is the discriminator: a reader cannot type one, so
   the marker as a document prints it is ordinary history rather than a claim on the zone. It is *rebuilt* across
   passes, and it is rebuilt by *merging*: a later pass folds new facts into the existing ledger
   line by line, so a summarizer that restates what the ledger already holds adds nothing
   (`MergeLedger`). The one exception is a consolidation: a body past `MaxLedgerTokens` is rewritten
   by a call that also carries turns no earlier pass had, and only when the rewrite still names
   every identifier the old body named (`NextLedger`, `MissingIdentifiers`). The zone may also own
   the tool replies answering calls the ledger absorbed (`LedgerEnd`), which keeps such a pair
   atomic; the sentinel itself still rides exactly one message.
3. **History** — everything between the ledger and the active window. The **only** region eligible
   for tombstoning and for JEV classification.
4. **Active** — the newest turns, sized by `keep_tokens` and floored at `keep_turns`
   messages. Kept verbatim, always. (Superseded 2026-09-22: it began as a count of
   messages, which is the one bound no tier can move — see the note at the end of §6.)

`Plan(conv, policy) []Span` computes the partition; `Apply` reassembles it.

### 3.2 Three tiers

| Tier | Trigger | Action | Model calls |
|---|---|---|---|
| **Soft** | `size ≥ soft_ratio × usable window` | Deterministic `Tombstone` of oversized history tool results older than the active window. | 0 |
| **Mid** | `size ≥ mid_ratio × usable window` | One batched JEV `Classify` over history blocks → prune (atomic) + ledger blocks + keep the rest. | 1 JEV call + 1 LLM call (ledger) |
| **Hard** | `size ≥ hard_ratio × usable window` | Mid, **plus** force-summarize the whole history (ignore `Keep` verdicts). Every `Keep` becomes a fold and every `Prune` still drops, so the rebuild lands at anchor+ledger+active by construction; there is no separate trim step to take. | 1 JEV + 1 LLM |

Defaults (locked in §5): soft `0.65`, mid `0.80`, hard `0.90`. Absolute `compact_at`, when set,
wins over the ratios and remains the fallback when `ContextWindow == 0`.

### 3.3 Invariants (each must be a test, not a comment)

- **I1 Pairing:** every `ToolResult` in the outgoing conversation is preceded by the assistant
  message carrying the matching `ToolCall`. Prune removes whole atomic blocks only.
- **I2 Anchor:** the first `anchor_messages` messages are byte-identical after any number of passes.
- **I3 Ledger monotonicity:** stated in three parts, because the old single sentence was already
  violated in letter while the property it protected held.
  - **I3a** the message carrying the ledger's sentinel is never a judge block, never history, and
    never a prune candidate (`Blocks` chunks only `ZoneHistory`; `LedgerEnd` claims the ledger's own
    replies). The sentinel is matched as the whole first line *including* the private-use codepoint
    U+E000 it ends in, because identity is decided by a text a reader supplies: with a typeable
    marker, a turn whose first line spells it (the ASCII form a document prints) matched, and
    `ledgerIndex` takes the first match in the `anchor..active` range, so the paste took the ledger's
    zone while the real ledger behind it read as history and became a prune candidate. Identity
    cannot be bound to the assistant role (assembly picks the ledger's role as the opposite of its
    neighbour, so an assistant-anchored session folds into a user-role ledger) and cannot be recorded
    outside the message text (`nacelle.Message` is a role and a list of parts, and `Part` is sealed to
    this package), so the marker being untypeable is the discriminator.
  - **I3b** no summarizer call is ever handed the ledger *alone*: every call that rewrites it also
    receives turns no earlier pass compressed, so a rewrite is measured against fresh material
    rather than being a summary of a summary.
  - **I3c** a rewrite of the ledger's body cannot *silently* drop what the earlier body named:
    it is granted only when every identifier the earlier body carried survives, or refused and the
    merge stands (`Identifiers`, `MissingIdentifiers`, `Stats.LedgerKept`/`LedgerReplaced`). The
    guard is **lexical and no more**: "identifier" means a backticked span or a path/flag/extension-
    shaped token and nothing else. A body whose facts are prose — a constraint, a decision, a number
    — carries no such token, so it has no rewrite guard at all: a consolidation that dropped one is
    granted and `Stats.LedgerReplaced` reports the success.
  - Monotonicity itself is now line-granular rather than byte-granular: a merge keeps every line the
    earlier body carried and drops only the ones the new one repeats. Growth is bounded by
    consolidation; duplication is bounded by the merge and by `keepAsk`.
- **I4 Role alternation:** the assembled conversation never has two consecutive messages of the
  same role (`Apply` runs `alternateFrom` after assembling, folding same-role neighbours together
  and carrying the merged parts forward so nothing is lost — see the Phase 4 note).
- **I5 Never grow:** a pass never increases `m.size`.
- **I6 Idle-only:** an automatic pass never races a live run or a queued line; a line typed during a
  pass is queued and delivered by `settleCompaction` (this already works — keep it working).
- **I7 Degrade, never corrupt:** a failed/absent judge, a 429, a timeout, or a malformed answer
  results in *nothing pruned*, never in a broken conversation.

---

## 4. JEV: what it is and the exact contract

JEV is TypeSafe AI's **System One** model (launched 2026-09-18; by Diogo Almeida, ex-OpenAI). It is
**not an LLM**: it generates no text. Facts verified against `docs.typesafe.ai` (API, Confidence,
Choice) and the TypeSafe blog:

- **Endpoint:** `POST https://api.typesafe.ai/v1/systemone`, header `Authorization: Bearer <KEY>`.
- **Request:** `{"state": <text | structured | messages>, "model": "jev-latest", "questions": {...}}`.
- **Primitives:** `choice` → `{choice, probabilities, confidence}`; `score` →
  `{score, legend, probabilities, confidence}`; `noul` → `{noul: 0..1}` (no confidence).
- **All questions are evaluated in parallel against the same `state`, in one call**, and "adding
  more questions does not create context-rot". → **one batched call, one `choice` question per
  history block.** This is the single most important fact for the design.
- **`confidence` is calibrated**, derived from the probability distribution; the docs explicitly
  recommend thresholding it by stakes and reading raw `probabilities` when the default statistic
  does not fit. Use `probabilities["prune"]` for the prune gate and `confidence` as the "model
  isn't sure" guard.
- **Cost/latency:** $0.042/MTok input, output free, 70–500 ms.
- **Limits:** ≤ 255 options per question; errors 401 / 422 / 429 / 529, with exponential backoff
  expected on 429/529.
- **Early access.** Availability and model names can move; treat the whole feature as opt-in.

### 4.1 Division of labour — JEV classifies, the LLM writes

JEV cannot emit the ledger's prose. So: **JEV decides `keep` / `prune` / `ledger` per block; the
existing summarizer writes the ledger, fed only the blocks JEV tagged `ledger` plus the previous
ledger.** This is a smaller, cheaper, better-scoped LLM call than today's whole-middle summary.

### 4.2 Privacy — the sharpest risk, decided here

Enabling the judge sends conversation history (source code, possibly secrets) to a third party.
Therefore: **judge off by default; opt-in only; never send the system prompt; never log the API
key; document the exposure in `docs/configuration.md`.** A user who does not opt in keeps today's
behaviour exactly.

---

## 5. Locked decisions (do not relitigate)

1. **Ratios:** soft `0.65`, mid `0.80`, hard `0.90`. Absolute `compact_at` overrides; window `0`
   falls back to `compact_at`.
2. **Package layout:** policy in a new `internal/compaction`; the TypeSafe HTTP client in a new
   `internal/jev` (one integration per package, mirroring `internal/diagnostics` / `internal/usage`).
   Logic must **not** grow inside `internal/tui` (filet: `funcsPerFile: 8`, `fileLines: 250`).
3. **JEV classifies, the LLM writes the ledger** (§4.1).
4. **Judge is opt-in and off by default** (§4.2).
5. **Anchor = the first user turn**; the system prompt is already out of band and stays there.
6. **`compact_at` is not removed** — it stays valid and is the fallback.
7. **No nacelle change.** If one becomes unavoidable, core tag then TUI adoption.
8. **The deterministic mask and the thrash guard stay.** They are the fallback path, not legacy.

---

## 6. Work plan

Each step is one session of work. Signatures are indicative of shape, not gospel — but the file
placement and the invariant each step protects are.

### Phase 0 — configuration surface (independent of Phase 1; can run in parallel)

**Step 0.1 — settings schema** (`internal/settings/config.go`)
Add under `Limits`:
```go
type Compaction struct {
    SoftRatio      *float64 `yaml:"soft_ratio"`
    MidRatio       *float64 `yaml:"mid_ratio"`
    HardRatio      *float64 `yaml:"hard_ratio"`
    KeepTurns      *int     `yaml:"keep_turns"`
    AnchorMessages *int     `yaml:"anchor_messages"`
    Judge          Judge    `yaml:"judge"`
}
type Judge struct {
    Enabled        *bool    `yaml:"enabled"`
    Model          string   `yaml:"model"`
    BaseURL        string   `yaml:"base_url"`
    APIKey         string   `yaml:"api_key"`   // prefer TYPESAFE_API_KEY
    PruneThreshold *float64 `yaml:"prune_threshold"`
    MaxBlocks      *int     `yaml:"max_blocks_per_call"`
}
```
Add a field named `Compaction` (YAML key `compaction`) to the `Limits` struct.

**Step 0.2 — defaults** (`internal/settings/defaults.go`): soft `0.65`, mid `0.80`, hard `0.90`,
`keep_turns: 3`, `anchor_messages: 1`, `prune_threshold: 0.75`, `max_blocks_per_call: 64`, judge
`enabled: false`. Leave `DefaultCompactAt` at 75 000 untouched.

**Step 0.3 — precedence chain** (`merge.go`, `env.go`, `setters.go`, `flags.go`, `scaffold.go`):
thread every key through file → env → flag, `KORI_COMPACTION_*` and `TYPESAFE_API_KEY`. Prefer
env+file over new flags if flags would push the collector past `params: 5`.

**Step 0.4 — budget resolver** (`internal/agent/compact.go`): replace `resolveCompactAt` with
```go
type Budget struct{ Ceiling int64; TierRatio struct{ Soft, Mid, Hard float64 }; Window int64 }
func ResolveBudget(compactAt int64, c settings.Compaction, backend nacelle.Backend) Budget
```
Ceiling = `compact_at` when non-zero, else `int64(ratio × ContextWindow)` per tier when the window
is known, else `settings.DefaultCompactAt`. **Delete the `== 100000` branch.**

**Step 0.5 — doc sync:** update the `ResolveCompactAt` reference in `docs/plan-model-command.md`
(that plan is unimplemented and must not drift).

**Exit 0:** `go build ./...` + `go test ./internal/settings/... ./internal/agent/...` clean;
`example.kori.yml` and `docs/configuration.md` show the new keys.

**Phase 0 as implemented (2026-09-21).** Three deviations, recorded per §0.5:

1. The schema lives in a new `internal/settings/compaction.go`, not `config.go`: `config.go` sat at
   243 of filet's 250 lines, so the types, their merge and the env readers went next door and
   `Limits` gained only the field.
2. `compact_at` is **unset by default** — `Defaults` no longer fills it — because a non-zero
   default would have made the ratio ladder dead on arrival. The setting is now tri-state: unset
   derives the ceiling from the ratios, `0` disables, any other value is an absolute override.
   `DefaultCompactAt` stays 75 000 as the windowless-backend fallback, and the existing `compact_at`
   default tests were updated to match.
3. `ResolveBudget` takes `compactAt *int64`, not `int64`, because the tri-state above needs
   "unset" distinguished from "0". It carries `TierRatio` even when `compact_at` overrode the
   ceiling, so a session that pins its trigger still tiers its passes.

Also folded in: `limits.compaction` merges per member for cron jobs too, so a job file cannot
carry a silently ignored compaction block.

### Phase 1 — zones, blocks, ledger, micro (no model call)

**Step 1.1 — `internal/compaction/policy.go`** (new): `Ratios`, `Policy`, `Tier`
(`Below|Soft|Mid|Hard`), `Zone`, `Span`, `Plan(conv, policy) []Span`, `Policy.Tier(size int64) Tier`.
Pure; imports `nacelle` only.

**Step 1.2 — `internal/compaction/blocks.go`** (new): move `AlignedCut` (+ `toolResultIDs`,
`toolCallIDs`) from `internal/tui/compact_pair.go`; add
`Blocks(conv []nacelle.Message, spans []Span) []Block` where a `Block` is an atomic span (assistant
`ToolCall` message + the user message answering it, or one standalone turn) and never opens on a
`ToolResult`.

**Step 1.3 — `internal/compaction/ledger.go`** (new):
```go
const Sentinel = "[state ledger]"
func IsLedger(m nacelle.Message) bool
func BuildLedger(previous, summary string) nacelle.Message  // keeps the sentinel, folds previous
```
Replaces `compactedHeader` / `compactedHistory` (moved from `compact_summary.go`).

**Step 1.4 — `internal/compaction/micro.go`** (new): move `estTokens`, `msgBytes`, `partBytes` and
the replacement logic of `trimResults`/`trimThinking`; expose
`Tombstone(conv []nacelle.Message, spans []Span) MicroStats` (`Results`, `Bytes`).
Idempotent: a stub is never re-stubbed or re-counted.

**Step 1.5 — `internal/compaction/apply.go`** (new): `Apply(conv, plan, ledgerText) ([]nacelle.Message, Stats)`
assembling anchor + ledger + surviving blocks + active window, preserving role alternation (I4) and
returning `Stats{Before, After, Masked, Pruned, Summarized, Tier}`.

**Step 1.6 — shrink `internal/tui`.** `compact_mask.go`, `compact_pair.go`, `compact_summary.go`
become wrappers that call the new package; `alignedEvictCut` stays as a one-line delegate so the
existing tests compile. Keep `compactOutcome`, the goroutine contract and `waitForCompact`.

**Step 1.7 — tiered trigger.** `run.go` `send` and `compact_idle.go` select the tier via
`Policy.Tier`; soft returns after `Tombstone` with no goroutine. Stop applying the `bytes/4` debit
as authoritative (G6): record the estimate, let the next `sized()` overwrite it.

**Step 1.8 — tests.** New `internal/compaction/*_test.go` for `Plan`, `AlignedCut`, `Blocks`,
`IsLedger`/`BuildLedger` (idempotence + monotonicity), `Tombstone` (idempotence). Port the
guarantees the existing seven `compact*_test.go` files assert, so nothing regresses silently.
`compact_test.go`'s literal `compactAt = 100_000` and the `100_000` in `helpers_test.go` /
`mode_test.go` must move to whatever `ResolveBudget` now returns.

**Exit 1:** I1–I6 hold in tests; soft tier does zero model calls; the seven legacy test files pass
unchanged in spirit; `filet check` clean.

**Phase 1 as implemented (2026-09-21).** Six deviations, recorded per §0.5:

1. `internal/compaction` gained `spans.go` (span selectors: `Section`, `HistoryMessages`,
   `HistoryRange`, `LedgerText`) and `AlignedCut` lives in `pair.go`, not `blocks.go`: filet's
   `funcsPerFile: 8` put the atomic-block chunker over on its own once the pair helpers were
   counted with it. `internal/agent/policy.go` folds a `Budget` plus the two end sizes into the
   TUI's `compaction.Policy`.
2. The ledger's default role is the assistant's, so it alternates with the user turn that anchors
   the head; `Apply` picks the role opposite its neighbour and folds the ledger into the turn that
   follows when it still must (I4). The sentinel is `[state ledger]`, replacing the old
   `[compacted context]` header; `BuildLedger` folds a previous ledger into a new one without
   duplicating either half.
3. `Policy.Tier` returns **Mid**, not Soft, for a windowless backend whose ceiling is crossed: with
   no window there is no soft ratio to measure a free tombstone against, so the ceiling buys the
   full pass. This preserves the pre-Phase-1 behaviour on backends that report no context window.
   With a window, `compact_at` floors the tier at Soft when it sits below the soft ratio.
4. `Tombstone` dropped the old byte budget: the soft tier tombstones every oversized history
   result and reasoning block, per §3.2. The `bytes/4` debit stays an estimate the next `sized()`
   overwrites (G6); it is no longer used to size a budget.
5. `SessionConfig.CompactAt` was replaced by `SessionConfig.Policy`, whose `Ceiling` carries the
   resolved absolute. A separate field was a 17th member and filet caps `structFields` at 16.
6. The `account` struct moved to `internal/tui/account.go` so `compact.go` stays under
   `fileLines: 250`.

### Phase 2 — the JEV judge

**Step 2.1 — `internal/jev/client.go`** (new): typed client for `/v1/systemone`.
```go
type Question struct{ Type string `json:"type"`; Instructions any `json:"instructions"`; Criteria any `json:"criteria,omitempty"` }
type Answer struct{ Type, Choice string; Confidence float64; Probabilities map[string]float64; Noul float64 }
type Response struct{ Model string; Answers map[string]Answer; Usage Usage }
func (c *Client) Evaluate(ctx context.Context, state any, qs map[string]Question) (Response, error)
```
Bearer auth (config, else `TYPESAFE_API_KEY`), bounded timeout, exponential backoff on 429/529,
typed errors on 401/422. No SDK dependency. Test with `httptest`.

**Step 2.2 — `internal/compaction/judge.go`** (new):
```go
type Decision uint8            // Keep | Prune | Ledger
type Block struct{ Key string; Start, End int; Text string }
type Verdict struct{ Decision Decision; PruneProb, Confidence float64 }
type Judge interface{ Classify(ctx context.Context, goal string, blocks []Block) ([]Verdict, error) }
```
Asymmetric rule lives here: `Prune` only when `PruneProb ≥ PruneThreshold` **and** `Confidence`
clears a floor; anything ambiguous, missing or errored → `Keep`.

**Step 2.3 — `internal/compaction/jevjudge.go`** (new): adapter rendering goal + blocks into one
`state`, building **one `choice` question per block** (`keep` / `prune` / `ledger`, criteria spelled
out: a decision, a constraint, a dead end), one **batched** `Evaluate` call, mapping answers back.
Respect ≤ 255 options, cap `max_blocks_per_call`, treat a failed call as all-`Keep`.

**Step 2.4 — `internal/compaction/macro.go`** (new): `Classify` → apply. `Prune` drops a whole
atomic span; `Ledger` collects the block; `Keep` leaves it. Runs off the UI thread inside the
existing compaction goroutine.

**Step 2.5 — wire into `internal/tui/compact.go`:** mid tier runs the batched JEV call, then sends
only the ledger-marked blocks to the summarizer, then one `Apply`. Extend `compactReport` to name
pruned / summarized / ledgered and the tier used. Keep the failure path: any error → mask fallback
+ a clear notice (today's wording is a good model).

**Step 2.6 — config/docs:** document `limits.compaction.judge` as opt-in and state the privacy
exposure plainly.

**Step 2.7 — tests:** `internal/jev/client_test.go` (success fixture, 401 typed error, 422, 429
retry then success, timeout); `internal/compaction/judge_test.go` (prune only at ≥ threshold;
ambiguous → Keep; malformed → Keep; prune atomic → no orphan `ToolResult`); a 40-turn load test
asserting the effective size settles under the hard ratio.

**Exit 2:** judge off ⇒ byte-for-byte today's behaviour; judge on ⇒ exactly one JEV request per pass
and one ledger call; a forced 429 prunes nothing and the session continues.

**Phase 2 as implemented (2026-09-21).** Seven deviations, recorded per §0.5:

1. `internal/jev` gained `types.go` and `errors.go` beside `client.go`: the wire types and the typed
   errors each deserved their own file, and filet's `funcsPerFile: 8` wanted the split.
2. `JudgeRequest` carries `Goal` and `Force` only. The prune threshold lives on `JudgeConfig`, where
   the adapter that reads it is built, so a request cannot silently re-threshold a judge.
3. `Stats` dropped its `Pruned` field: `Apply` is handed a survive predicate rather than the verdicts,
   so it cannot tell a prune from a fold. The counts come from the returned `Fold`, which the TUI
   report reads instead of inventing a number.
4. `Apply` gained a fifth parameter, `keep func(int) bool` (`Fold.Survives`). It is how a classified
   pass keeps the blocks the judge tagged Keep; `nil` is the pre-judge pass that drops the history.
5. The judge call is bounded by its own `compactJudgeTimeout` (30 s) inside the pass, on top of the
   client's per-attempt timeout and retry budget, so a wedged endpoint cannot hold "compacting".
6. `SessionConfig.Policy` became `SessionConfig.Compaction{Policy, Judge}` to stay under filet's
   `structFields: 16`. The judge is built in `internal/agent` and passed as a `compaction.Judge`
   interface, so the TUI never sees the HTTP client.
7. Blocks past `max_blocks_per_call` are folded (`Ledger`), not kept: the summarizer can compress
   what the judge never saw, where a keep would leave the ceiling to be crossed again.

Hard still calls the judge and then upgrades every `Keep` to `Ledger` (§3.2's force-summarize);
`Prune` verdicts survive that upgrade, so a hard pass prunes what mid would have kept.

### Phase 3 — surface

**Step 3.1** — `internal/tui/status.go` (`footer`) and `internal/tui/command.go` (`statusCmd`):
show the ratio (e.g. `↕ 120k/200k · 0.60`) and the active tier; `/status` reports ledger size and
the last pass's tier.

**Step 3.2** — `docs/configuration.md`, `example.kori.yml`, `README.md`: document ratios, tiers,
the judge, and the privacy note.

**Step 3.3** — `CHANGELOG.md` and `ROADMAP.md` Track H: record the upgrade and close the
"summarize near the limit" item.

**Exit 3:** options and their defaults are documented; `ROADMAP.md` reflects reality.

**Phase 3 as implemented (2026-09-21).** Three deviations, recorded per §0.5:

1. The status surface lives in a new `internal/tui/context_line.go`. `status.go` and `command.go`
   were both already at filet's `funcsPerFile: 8`, so the two formatters (`contextLoad` and
   `Model.compactionLines`) went next door rather than pushing either file over the cap.
2. The footer carries the tier inside the context item — `↕120k/200k · 0.60 · soft` — instead of as
   a separate token, so the ratio and the tier it selected cannot be read apart. The tier is left
   off below the soft ratio, the same "nothing active to name" the `Below` rung already means.
3. `/status` reports the ledger's **live** measured size, read from the conversation's own
   `[state ledger]` message, and the last pass's tier from a new `transcript.last` field. The
   ledger accumulates across passes, so the last pass's own fold size would understate it; the
   tier, unlike the size, is a property of the pass and not of the ledger, so it comes from the
   recorded outcome.

`example.kori.yml` needed no edit: Phase 0 already wrote the compaction block there, and the
scaffold-parity test forbids changing one without the other.

### Phase 4 — hardening sweep

**Step 4.1** — re-run §7 with judge on and judge off; run `filet check`; fix every finding inside the
file it belongs to (never raise a limit).

**Step 4.2** — update this document's status line and note any deviation from the plan.

**Phase 4 as implemented (2026-09-21).** The sweep ran the whole §7 matrix with the judge on and off
and the four gate commands; all four were already clean. The one finding was a hole in I1 the matrix
did not reach, found by exercising the assembled conversation rather than a single pass:

`Apply` reassembles by folding same-role neighbours together, and the ledger sits immediately after
the anchor with the *opposite* role — so when the first surviving block, or the first active turn,
is an assistant `ToolCall`, it was folded into the ledger while its `ToolResult` stayed behind. Two
failures followed. On the next pass `Blocks` saw that result as an orphan block (a prune could drop it
while the call stayed in the ledger, orphaning the call instead), and because the ledger is rebuilt
from its text alone the absorbed call — and any assistant text a merge folded in — was silently
dropped. The judge-off path hits it too: with `keep_turns` odd, the active window opens on an
assistant turn and the ledger swallows it.

Three changes close it, all in `internal/compaction`:

1. `Plan` now extends a ledger span over the replies answering the calls the ledger itself carries
   (`LedgerEnd`), so an absorbed result is never an orphan in history and is never a prune candidate;
   and `activeStart` no longer pulls the active boundary back onto a ledger (the ledger is not
   dropped with the cut, so the result may safely open the window instead of the ledger being
   swallowed into it).
2. `Apply` carries the ledger zone forward — the extra parts a merge folded into the ledger
   (`ExtraParts`) and its reply messages — and re-emits them with the rebuilt ledger, so nothing an
   absorption touched is lost when the ledger is next rebuilt from its text.
3. `ledgerRole` now alternates against the first message that will actually follow the ledger
   (carried replies included), not just the surviving history.

A consequence worth stating: the ledger zone is no longer necessarily one message. When the ledger
carries a tool call it also owns that call's reply, which is exactly what keeps the pair atomic. Four
tests pin it — `TestApplyKeepsAnAbsorbedToolPairWhole`, `TestApplyCarriesWhatTheLedgerAbsorbed`,
`TestPlanExtendsTheLedgerOverItsReplies`, `TestPlanDoesNotSwallowTheLedgerIntoTheActiveWindow` — and
all four fail against the pre-sweep code.

### Review follow-ups (2026-09-21)

A review of the unpushed commit found four more holes, all closed in the same commit. Recorded here
per §0.5.

1. **An empty summary with the judge on deleted history and reported it as a summary.**
   `settleCompaction` installed on `outcome.judged` alone, so a pass whose summarizer streamed no text
   (a failure mode measured in this repo) dropped the turns the judge had tagged for the ledger and
   built no ledger at all, while the card claimed "summarized N turns". The condition is now
   `outcome.installs()`: a non-empty summary, or a judged pass that tagged nothing and so had nothing
   to summarize. An empty summary falls back to the mask the unjudged path already used.
2. **I5 was asserted, not enforced.** `Apply` measured its own rebuild and installed it regardless,
   and a judged pass folding a two-byte turn into a verbose ledger does grow the conversation
   (measured: 3 → 3206 estimated tokens). A rebuild heavier than the conversation it replaces is now
   refused — `Stats.Refused`, the original stands, the card says the context is unchanged.
3. **The pinned head could split a tool pair, and with no ledger the assembly could not stay
   role-legal.** `Plan` extends the head over the replies answering the calls it carries (`anchorEnd`,
   the rule `LedgerEnd` already applied to the ledger). `Apply` now installs a ledger on *every* pass
   that changes anything, even with no text to fold, because that message is the buffer that keeps
   the head and the turn after it from colliding: without it the only options were merging a kept
   turn into the anchor (I2 broken) or emitting two same-role messages (I4 broken), and nacelle does
   not merge roles on the way out (`anthropic/conversation.go` passes every message through as-is).
   A call that changes nothing is still handed back untouched.
4. **`nacelle.Trim` is not used.** §3.2 and §12.1 promised it as the hard tier's last resort; the
   force path already lands at anchor+ledger+active, so the text is corrected rather than the code.

Also folded in: `/status` names an opted-in judge, so the one setting that ships history off the
machine is visible where the reader already looks for the ladder; and the JEV client retries a dropped
connection or a per-attempt timeout as well as a 429/529, bounded by the caller's own context.

### Review follow-ups, second round (2026-09-21)

A second review of the same unpushed work, run against a tree where all four gates were already
green, found five more things worth closing. Recorded here per §0.5. No invariant was weakened:
I1–I7 hold as before, and two of these close holes *inside* them.

1. **The soft tier credited itself the wrong thing.** Reasoning was tombstoned alongside tool
   results and counted as bytes freed. No backend ever sends a `Reasoning` part back — `anthropic`'s
   `blocksOf` and the shared `internal/oairunner` `sift` both switch on `Text`/`ToolCall`/`ToolResult`
   and nothing else, because a thinking block needs the signature the stream never carries. So the
   stub freed no context at all, `maskHistory` debited `m.size` by bytes that were never in the
   request, and the report said "masked N thinking blocks" as if it had bought headroom — while
   taking the chain of thought out of a transcript the reader can still scroll back to. `Tombstone`
   now touches tool results only, `MicroStats` lost `Thinking`, and one dropped notice is left.
2. **A zero prune threshold pruned everything.** `decide` compares `pruneProb >= threshold`, and a
   config that never mentions the key arrives as a zero — so a block the model voted 80% *keep* would
   be dropped, inverting the one destructive verdict. The threshold now has a floor
   (`DefaultPruneThreshold`, aliased from settings), `NewJevJudge` fills an unusable one, `decide`
   refuses to prune below a positive threshold, and settings refuses a value outside `(0,1]`.
3. **`soft_ratio: 0` turned compaction off while reporting it on.** The derived ceiling is
   `soft_ratio × window`, so a zero produced a zero ceiling — which is how every gate spells
   "disabled" — while `Policy.Tier` still reported `hard`. `validateCompaction` now rejects an
   out-of-range or out-of-order ladder at load, `ResolveBudget` floors a degenerate derived ceiling at
   `DefaultCompactAt`, and the `/compact` refusal names the setting that did it.
4. **`Apply` indexed the conversation with the pass's plan.** A plan is measured against one
   conversation and installed against whatever is in the field, and `Section` sliced
   `conv[span.Start:span.End]` unguarded: a `/clear` or `/resume` landing mid-pass was an
   index-out-of-range panic. `Covers` is now the precondition, `Stats.Stale` reports the refusal, and
   the selectors trim a span past the end rather than trusting it.
5. **Two smaller ones.** A tool call's arguments now reach the judge, abbreviated to
   `maxBlockInput`, since a block judged on `tool call read` alone cannot tell two reads of different
   files apart; and the JEV client retries a gateway's 502/503/504 like a 429/529, while a zero-value
   `Client` makes one attempt through the default HTTP client instead of returning an empty success
   that a caller cannot tell from "nothing to prune".

6. **The mask fallback guarded the conversation it touches, and a dead parameter went.**
   `applyMaskFallback` was the one remaining place a pass's plan could be applied to a conversation
   it never measured; it now checks `Covers` and reports whether it masked, so the "masked instead"
   notice cannot claim a fallback that did not land. And `Apply`'s `policy` parameter was unused —
   the plan already carries the two ends the assembly needs — so it is gone rather than left as a
   signature that implies a decision the function does not make.

`nacelle.Trim` remains unused, for the reason recorded in the first round.

### Review follow-ups, third round (2026-09-21)

A third review — this one asking whether the design matches published practice rather than whether
its own invariants hold — found two things this plan had asserted and never checked. The calibration
harness built to check them found a third within minutes of being pointed at the live endpoint, and
a fourth once the judge started answering at all.

1. **The judge had never worked.** TypeSafe takes a choice's `criteria` as an object keyed by option;
   `choiceQuestion` sent a list of sentences. The endpoint answers 422, `InvalidRequestError` is
   correctly non-retryable, and the pass fell back to the mask — so the whole judge, the one setting
   that ships conversation history to a third party, had classified nothing since it shipped. Every
   test passed because every test talks to a stub, and a stub accepts any shape. The criteria are a
   map now, and `TestChoiceQuestionCriteriaEncodeAsAnObject` pins the *encoded* shape, which is the
   assertion a stub cannot make on your behalf.
2. **Every question was the same question.** A batch is one shared state and one question per block,
   and the questions were byte-identical: nothing said which block a given question was about. The
   model answered `keep` for all seventeen blocks at 0.9 confidence, which reads as a decisive verdict
   over a whole history and is really the absence of one. Each question now names its block;
   agreement with the labels went from 24% to 65%.
3. **The criteria overlapped, so confidence collapsed.** "A dead end worth keeping in compressed
   form" (ledger) and "a dead end with no lasting value" (prune) describe the same block — exactly
   the ambiguity `ConfidenceFloor` exists to refuse. Confidences sat at 0.2–0.3, under the 0.6 floor,
   so no verdict was ever acted on and the mid tier was decorative. Disjoint descriptions took them
   to 0.8–0.9.
4. **`prune_threshold` had never been measured.** With the judge answering, the sweep gives a curve:
   0.85 recovers 17% of the prunable blocks a careful operator would drop, 0.75 recovers 50% at the
   same zero false prunes, and agreement goes 65% → 76%. The default moves to 0.75 with the
   measurement in the constant's own comment. `ConfidenceFloor` stays at 0.6 on the bands' evidence:
   every verdict at or above it was right, and every one below it was wrong.
5. **The soft tier had no `clear_at_least`.** Anthropic's context-editing guidance for its own
   tool-result clearing is explicit that clearing invalidates the cached prompt prefix and should be
   held to a minimum tokens-cleared budget. The tombstone ran on any one result over `MinResult`
   (1 KB). `MinCleared` is now the floor under the pass and `DroppableBytes` is the pre-flight that
   measures it, so a marginal overshoot waits until the history is worth clearing.
6. **The judge's request was unbounded.** `max_blocks_per_call` bounds the block count and
   `maxBlockInput` bounds one call's arguments, but each block carries a whole tool result and the
   state concatenates them — so a handful of large results is a multi-hundred-KB request to a small
   decision model, billed, with no ceiling. `maxBlockText` cuts one block and the goal — the
   anchor's own text, which a pasted first message can make larger than every block together —
   `defaultMaxState` bounds the batch, and the newest block is always admitted, so a byte cap can
   never silently disable the judge.
7. **Two smaller ones.** The answering model id was decoded and discarded while the setting defaults
   to the drifting `jev-latest` alias the vendor says to log and then pin, so `Reporter`/`LastAnswer`
   carry it and `/status` shows it with the bill. And a pass read the conversation off the update
   loop's goroutine under an invariant that held but was invisible at the point it mattered; it now
   runs on a `compactPass` snapshot, so the read safety is local. `alignedEvictCut` went with it — its
   only caller in the repository was its own test — and so did the superseded `compaction-upgrade.md`.

### Tail budget and reserve (2026-09-22)

Two things §3.1 and §5 left as magic values, closed after a review that asked whether the design
matched published practice rather than whether its own invariants held. Recorded here per §0.5.

1. **The active window is sized by a budget, not a count.** `keep_turns: 3` was a *message* bound,
   and a message bound is the one figure neither tier can move: the newest turns are exactly where
   a session's largest tool results land, and the active window is never touched. Two heavy reads
   could therefore hold the conversation over the trigger with nothing any pass could do, which is
   the shape the thrash guard was quietly absorbing. `keep_tokens` (40k) is now the budget, walked
   back over whole messages; `keep_turns` drops to 1 and becomes the *floor* — the live turn alone,
   which is also the one turn no summary may stand in for. The **token budget** is capped at half of
   what a pass may fill (`min(usable window, ceiling)/2`), since a tail that size could not land
   under the trigger it fires. The **message floor carries no such cap**: `walkBack` starts at the
   floor's cut and only ever widens it, so a `keep_turns` big enough to take more than half the fill
   is honoured anyway. It is the one bound a pass cannot move.
2. **The ladder is read against a usable window.** The ratios were fractions of the raw window, so
   `hard_ratio: 0.90` left a tenth of the window for the response — thin on a reasoning model. A
   `reserve_tokens` (a fifth of the window within 8k–64k when unset) is now held back for the turn's
   own answer and subtracted *before* the ladder is read, so `hard` leaves the reserve plus a tenth
   of the usable window. A `window_tokens` override was added in the same breath, because the
   reserve needs a window to derive from and an OpenAI-compatible runner reports none.

Two consequences worth stating. The pre-send guard now fires at `Trigger()` rather than
`Trigger() + compactSlack`: the trigger is already the soft ratio of the usable window, so waiting
only changes which tier answers, and upward — and the thrash guard's "did it land" test moved to
the same figure, since a margin between the two made every pass stopping inside it a silent reset.
`compactSlack` therefore has no production role left and survives only as a fixture offset in the
tests. And package `tui`'s fixture session pins its tail explicitly
(`KeepTurns: 3, KeepTokens: 1`), because a fixture conversation is small enough that the shipped
40k budget swallows it whole and leaves the pass tests with no history to fold.

The invariant that §3.3's I3 states survives untouched: nothing here re-summarizes the ledger, and
the new bounds only decide *which messages stay verbatim*.

### Ledger growth and duplication (2026-09-22)

Two defects, one cause: the fold handed the previous ledger to the summarizer and asked it to "fold
it into the new one", then concatenated whatever came back unless it textually contained the old
body. A model that *reworded* what it was shown produced `previous + reworded previous + new`, so
the body doubled on every pass and nothing bounded it — the fold is monotone by design, and I5
compares whole-conversation estimates, so a body adding kilobytes a pass is invisible until it
dominates the window.

What changed:

1. **The merge is structural, not textual.** `MergeLedger` parses both bodies into headed sections
   (`sections` — the schema's own section names, or a short label ending in a colon, or markdown
   emphasis) and merges line by line with a normalized dedup key, so a restatement costs nothing.
2. **The prompt says why the ledger is shown.** `keepAsk` tells the summarizer not to repeat what
   the earlier ledger holds. The merge catches exact restatement; the ask is what stops the reworded
   half, which no deterministic check can see.
3. **The body is bounded.** `MaxLedgerTokens` (aliased by the TUI's `compactMaxTokens`, so the
   ledger's budget and the summarizer's own ceiling are one number) is the size above which a pass
   consolidates: `beginCompaction` decides it on the update loop from the ledger the conversation
   actually holds (`LedgerOverBudget`), `compactPass.consolidate` carries it to the goroutine, and
   `consolidateAsk` asks for one rewritten block instead of an addition. A consolidating pass also
   forces the fold, the way the hard tier does, because the turns it folds are what carry the earlier
   ledger into the ask: a keep-heavy judge used to leave the pass with no ledger block at all, so the
   summarizer was never called and the body stayed over budget.
4. **The rewrite is gated.** `NextLedger` grants a replacement only when `MissingIdentifiers` is
   empty — a lexical check over backticked spans and path/flag/extension-shaped tokens. A rewrite
   that would lose one is refused and the merge stands, which costs growth rather than a fact. The
   refusal is reported (`Stats.LedgerKept`, "kept the ledger as written" in the report card).
5. **A provider refusal is recovered.** `internal/overflow.Detect` recognises the context-length
   vocabulary every backend words differently, `armRecovery` holds it instead of printing it,
   `settle`→`recoverOverflow` compacts with a forced hard pass and `retryRun` starts the run again.
   Once per turn (`run.overflowTried`), and never with `compact_at: 0`.

I3 is restated above rather than weakened: what protected the ledger was never that the summarizer
could not see it, but that it was never the *only* thing the summarizer could see, and that is
I3b — now explicit — with I3c added as the guard the original design lacked.

---

## 7. Test & validation matrix

| Invariant / behaviour | Test | Where |
|---|---|---|
| I1 pairing after prune | no `ToolResult` without a preceding `ToolCall`; property over random conversations | `internal/compaction/blocks_test.go` |
| I1 pairing when the ledger absorbs a kept block | no orphan result, pair survives a second pass and a prune | `internal/compaction/apply_test.go`, `policy_test.go` |
| I2 anchor preserved | first `anchor_messages` byte-identical after 5 passes | `internal/compaction/apply_test.go` |
| I3 ledger monotonic | re-pass does not re-summarize; the merge keeps every line and repeats none | `internal/compaction/ledger_test.go`, `ledgerbody_test.go` |
| I3 duplication is bounded | 8 passes whose summarizer echoes its own section list leave each fact recorded once | `internal/compaction/ledgerbody_test.go` |
| I3c a rewrite never loses an identifier | a consolidated body is granted only with every identifier kept, and refused otherwise | `internal/compaction/ledgerident_test.go`, `ledgerbody_test.go` |
| Ledger budget | `LedgerOverBudget` at one summary; the pass consolidates rather than adds, and the report names it | `internal/compaction/ledgerbody_test.go`, `internal/tui/compact_ledger_test.go` |
| Overflow recovery | the rejection is held, answered once, forced to hard, and reported verbatim when there is no history to fold | `internal/overflow/overflow_test.go`, `internal/tui/overflow_test.go` |
| I4 role alternation | assembled slice has no same-role neighbours | `internal/compaction/apply_test.go` |
| I5 never grow | `After ≤ Before` for every tier, and a rebuild that would grow is refused with the original handed back | `internal/compaction/apply_test.go` |
| I1 the pinned head never splits a pair | a head that is a `ToolCall` claims its reply; history never opens on an orphan result | `internal/compaction/policy_test.go` |
| I4 both ends legal without a ledger | a pass that folds nothing still stands a ledger buffer between the head and the active window | `internal/compaction/apply_test.go` |
| I7 empty summary with the judge on | masks instead of folding turns it cannot summarize | `internal/tui/compact_live_test.go` |
| I6 idle-only + queued delivery | existing `compact_idle_test.go` cases still pass | `internal/tui` |
| I7 degrade safely | 429 / timeout ⇒ a failed call keeps every block; a malformed answer has **no test** | `internal/jev/retry_test.go`, `internal/compaction/jevjudge_test.go` |
| Tier boundaries | `Tier(size)` at 0.649/0.65/0.799/0.80/0.899/0.90 of a known window | `internal/compaction/policy_test.go` |
| Window unknown | `ContextWindow == 0` ⇒ falls back to `compact_at` | `internal/compaction/policy_test.go` |
| Tombstone idempotence | second pass adds no stub and no debit | `internal/compaction/micro_test.go` |
| Load | 40 turns of large reads stay under hard ratio | `internal/compaction/load_test.go` |
| Asymmetric judge | `prune` at 0.70 ignored; at 0.86 applied | `internal/compaction/judge_test.go` |
| Batching | N blocks ⇒ one HTTP request | `internal/compaction/jevjudge_test.go` |
| Reasoning is never tombstoned | a stub on it frees nothing and is not credited | `internal/compaction/micro_test.go`, `internal/tui/compact_mask_test.go` |
| The prune threshold has a floor | an unusable threshold prunes nothing | `internal/compaction/judge_test.go`, `jevjudge_test.go` |
| A degenerate ratio cannot disable the ladder | derived ceiling falls back; the value is refused at load | `internal/agent/compact_test.go`, `internal/settings/compaction_test.go` |
| A stale plan is refused, not indexed | `Covers` is exact; `Apply` reports `Stale` with the original | `internal/compaction/apply_test.go`, `pair_test.go` |
| The judge sees what a call did | a block carries the call's arguments, abbreviated | `internal/compaction/blocks_test.go`, `micro_test.go` |
| A gateway failure is transient | 502 retried like a 429 | `internal/jev/retry_test.go` |
| A question is one the live endpoint accepts | the criteria encode as an object, and every question names its own block | `internal/compaction/jevquestion_test.go` |
| The judge request is bounded in bytes | a block is cut to `maxBlockText`, the goal to the same cap, the batch to `defaultMaxState`, and the newest block is always asked about | `internal/compaction/jevjudge_test.go`, `jevquestion_test.go` |
| The answering model is reportable | `LastAnswer` carries the version and the bill, and survives a failed pass | `internal/compaction/jevjudge_test.go`, `internal/tui/context_line_test.go` |
| The soft tier does not clear for a cache-invalidating few bytes | a pass under `MinCleared` is skipped and reports nothing | `internal/tui/compact_light_test.go`, `internal/compaction/micro_test.go` |
| The thresholds are measured, not asserted | one live call over the labeled corpus, swept offline; no `keep` block pruned, accuracy over the floor | `internal/compaction/calibration_test.go` |

**Gate, every phase:**
```
go build ./...
go test ./... -race
golangci-lint run ./...
filet check
```
All four clean. filet is `failOn: info`, so a style warning fails the gate; fix the code, never the
config.

**Gate, whenever a judge threshold moves** (opt-in, needs a key, so it is not part of the four
above — but it is the only evidence those two numbers have):
```
TYPESAFE_API_KEY=... go test ./internal/compaction -run JudgeCalibration -v
```
It fails when a block labeled `keep` is pruned at the shipped threshold, or when agreement with the
labels drops under `minCalibrationAccuracy`. Raise the corpus before raising the number.

---

## 8. Config reference (target)

```yaml
limits:
  compact_at: 75000           # absolute override; unset => ratio-derived; 0 disables
  compaction:
    soft_ratio: 0.65          # tombstone only, no model call
    mid_ratio: 0.80           # + one batched JEV pass + one ledger summary
    hard_ratio: 0.90          # + force-summarize history; lands at anchor+ledger+active
    window_tokens: 200000     # overrides the backend's reported window (unset = the backend's)
    reserve_tokens: 40000     # runway held back for the answer; the ratios are read
                              # against the window less it (unset = a fifth of the window)
    keep_tokens: 40000        # the verbatim tail's budget, in tokens
    keep_turns: 1             # the tail's floor, in messages (the live turn alone)
    anchor_messages: 1        # pinned head (the first user turn)
    judge:
      enabled: false          # OPT-IN: sends conversation history to TypeSafe
      model: jev-latest
      base_url: https://api.typesafe.ai
      api_key: ""             # prefer the TYPESAFE_API_KEY env var
      prune_threshold: 0.75
      max_blocks_per_call: 64
```
Env: `KORI_COMPACTION_*` for the scalars, `TYPESAFE_API_KEY` for the key.

---

## 9. Risks / unknown unknowns

1. **Privacy / exfiltration (highest).** The judge sends history to a third party. Mitigation:
   off by default, docs, never send the system prompt, never log the key. (§4.2)
2. **JEV is early access.** Availability, rate limits and model names can move. Mitigation: the
   `Judge` interface, a nullable judge, retry/backoff, and a 429 degrading to "keep everything".
3. **Cardinality / state size.** One question per block, ≤ 255 options; a very long history may
   exceed a comfortable batch. Mitigation: `max_blocks_per_call` plus the summarizer for the rest.
4. **Prompt-cache busting.** Rewriting the prefix invalidates the provider KV cache; nacelle's own
   roadmap warns and set the shipped micro threshold high on purpose (`nacelle/ROADMAP.md:19-25`).
   Mitigation: soft tier tombstones only, and only on real overshoot.
5. **`ContextWindow == 0`** on some backends ⇒ ratio undefined. `compact_at` is the fallback.
6. **filet ceilings.** `internal/tui` sits near the limits; all growth goes in the new packages, and
   any wrapper that pushes a TUI file over is split in the same step.
7. **`/model` interlock.** Both this plan and `docs/plan-model-command.md` touch the resolver. Land
   Phase 0's rename before `/model`, or implement them together.
8. **Test-fixture drift.** `compactAt = 100_000` literals appear in `compact_test.go`,
   `helpers_test.go`, `mode_test.go`; they must move with the resolver.
9. **Threshold feel.** 0.65 is aggressive by the repo's own history; if a real session thrashes at
   the soft tier, raise `soft_ratio` before touching the mid tier — the soft tier is free and the
   mid tier is not.

---

## 10. Out of scope (YAGNI — do not gold-plate)

- No nacelle core rewrite; strategy stays consumer-side (`nacelle/trim.go`).
- No provider-side / native compaction, no checkpoints, no plan-mode machine.
- No *unbounded* summarization of the ledger; no "summary of a summary" path, ever. The one
  rewrite (a consolidation) is bounded by `MaxLedgerTokens`, gated on identifiers, and always made
  in a call that carries uncompressed turns — see the 2026-09-22 note under §6.
- No JEV-based tool-risk gating (a natural fit for nacelle's `BeforeToolCall`, but a separate
  feature with its own security review).
- No per-tool semantic summaries or tool-specific stubs; one tombstone shape.
- No migrations, no auth/porte, no muse/UI, no events envelope — none apply to this repo.
- Not adopting the source spec's 85/95 ladder; not removing `compact_at`.

---

## 11. Implementer checklist

- [x] Phases 0–4 done in order, each **Exit** verified.
- I1–I7 implemented and carried by tests. I3a is pinned by `TestClassifyIsNeverOfferedTheLedger`
  and `TestPruningEverythingStillCannotReachTheLedger` (`internal/compaction/macro_test.go`) — a
  plan that gives the ledger its own zone, no block overlapping it, and a judge that prunes every
  block it is shown still unable to reach it — alongside the older `TestBlocksIgnoreNonHistorySpans`
  which exercises an anchor span. I3b is pinned by `TestARewriteCallIsNeverHandedTheLedgerAlone` and
  `TestTheI3bCounterSeesALedgerOnlyAsk` (`internal/tui/compact_prompt_test.go`), the second of which
  keeps the first honest by asserting the counter returns 0 on the forbidden ledger-only shape.
- [x] No new logic in `internal/tui` beyond thin wiring.
- [x] `filet check`, `go test ./... -race`, `golangci-lint run ./...`, `go build ./...` all clean.
- [x] Judge off reproduces today's behaviour; judge on makes exactly one JEV call per pass.
- [x] `docs/configuration.md`, `example.kori.yml`, `README.md`, `CHANGELOG.md`, `ROADMAP.md` updated.
- [x] This document's status line updated; deviations recorded.

---

## 12. Appendix

### 12.1 Glossary

- **Anchor** — the pinned head of the conversation (the original task); never rewritten.
- **Ledger / State Ledger** — the single accumulating `[state ledger]` message, its marker ending in the private-use codepoint U+E000; never summarized.
- **Atomic block** — an assistant `ToolCall` message plus the user message answering it, or one
  standalone turn; the unit of prune.
- **Tier** — soft / mid / hard, selected by measured size against the window ratio.
- **JEV** — TypeSafe AI's System One model: typed questions in, calibrated decisions out; no text.

### 12.2 Sources (verified 2026-09-21)

- TypeSafe docs: `https://docs.typesafe.ai/introduction/quickstart`, `.../api`, `.../confidence`,
  `.../primitives/choice`.
- TypeSafe launch post: `https://typesafe.ai/blog/introducing-system-one-models-and-jev`.
- LangChain: `https://www.langchain.com/blog/building-a-harness-with-jev`.
- Press: `https://techcrunch.com/2026/09/18/a-new-kind-of-ai-model-from-a-chatgpt-inventor-is-thrilling-developers/`.
- In-repo: `internal/tui/compact*.go`, `internal/agent/compact.go`, `internal/settings/*`,
  `nacelle/trim.go`, `nacelle/stream.go`, `nacelle/hooks.go`, `nacelle/message.go`,
  `nacelle/ROADMAP.md` (Track H, shipped microcompaction, standing constraints).

### 12.3 Source spec

The TS-shaped original this document supersedes was `compaction-upgrade.md` at the repository root.
It was deleted rather than kept, because it named `src/context/*.ts` paths that do not and will not
exist here and nothing in it said so. Its useful ideas — micro/macro duality, anchors, the state
ledger, the asymmetric confidence gate — are kept in the code; its file layout, its 85/95 ladder, and
its assumption that JEV produces summaries are not.
