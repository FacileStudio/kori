package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FacileStudio/kori/internal/compaction"
	"github.com/FacileStudio/nacelle"
)

// compacted holds one compaction pass's numbers, so the report and the tests
// that check it share one shape rather than a bag of named arguments. evictCut
// and kept are the history and active sizes the pass measured, which is what the
// "kept N% verbatim" line is drawn from; results, thinking and turns are what it
// actually did.
type compacted struct {
	evictCut int
	turns    int
	kept     int
	pruned   int
	results  int
	thinking int
	tier     compaction.Tier
}

// compactOutcome is what a compaction pass sends back to the update loop. On the
// wire it carries the size before, the plan the pass ran against, the tier it
// selected and the summary the backend produced, plus any error from the
// summarizer call. The install and report functions fill in the after size, the
// rebuilt conversation and the tallies on the UI thread, so the goroutine never
// races a list it is also reading.
type compactOutcome struct {
	before  int64
	after   int64
	done    compacted
	plan    []compaction.Span
	fold    compaction.Fold
	tier    compaction.Tier
	summary string
	judged  bool
	stage   string
	err     error
}

// failedAt names which half of a pass failed, for the notice: the judge that
// classifies, or the summarizer that writes the ledger. An outcome built without
// a stage — a test's — reads as the summary, the half that has always existed.
func (o compactOutcome) failedAt() string {
	if o.stage != "" {
		return o.stage
	}
	return "summary"
}

// compactReport is how the client says a pass went: the context size it carried
// before and after, the share kept verbatim, what the mask and the summary did,
// and the tier that ran. A short block, so it reads as a visible milestone in the
// transcript rather than a one-word aside.
func compactReport(outcome compactOutcome) string {
	d := outcome.done
	freed := outcome.before - outcome.after
	kept := 0
	if d.evictCut+d.kept > 0 {
		kept = d.kept * 100 / (d.evictCut + d.kept)
	}

	var work []string
	if d.results > 0 {
		work = append(work, "masked "+countedNoun(d.results, "result"))
	}
	if d.thinking > 0 {
		work = append(work, "masked "+countedNoun(d.thinking, "thinking block"))
	}
	if d.turns > 0 {
		work = append(work, "summarized "+countedNoun(d.turns, "turn"))
	}
	if d.pruned > 0 {
		work = append(work, "pruned "+countedNoun(d.pruned, "message"))
	}
	if len(work) == 0 {
		work = []string{"kept everything verbatim"}
	}
	return "✂ Compaction summary (" + d.tier.String() + ")\n" +
		fmt.Sprintf("   before → after  %s → %s tokens (freed %s)\n", shortTokens(outcome.before), shortTokens(outcome.after), shortTokens(freed)) +
		fmt.Sprintf("   kept            %d%% verbatim\n", kept) +
		"   work            " + strings.Join(work, ", ")
}

// compactPrompt is what the summarizer is fed: exactly the history the pass may
// touch, with the ask appended and any earlier ledger folded into that ask so a
// later pass builds on it instead of re-deriving it. The anchor and the active
// window are deliberately excluded, so the model cannot leak the present into
// the past's summary.
func compactPrompt(conv []nacelle.Message, plan []compaction.Span, fold compaction.Fold) []nacelle.Message {
	history := fold.LedgerMessages(conv)
	ask := compactAsk
	if previous := compaction.LedgerText(conv, plan); previous != "" {
		ask += "\n\nAn earlier " + compaction.Sentinel + " is shown below. Fold it into the new one:\n" + previous
	}
	return withAsk(history, ask)
}

// withAsk appends the ask as a user turn, folding it into the last history turn
// when that turn is already the user's — a tool-result reply usually is — so the
// summarizer is never handed two user messages in a row, which the backends
// refuse.
func withAsk(history []nacelle.Message, ask string) []nacelle.Message {
	if len(history) == 0 || history[len(history)-1].Role != nacelle.RoleUser {
		return append(history, nacelle.UserText(ask))
	}
	last := history[len(history)-1]
	last.Parts = append(append([]nacelle.Part{}, last.Parts...), nacelle.Text{Text: "\n\n" + ask})
	history[len(history)-1] = last
	return history
}

// compactSystem is the schema the summarizer writes against. Structured on
// purpose — the research on long-horizon agents is blunt that freeform
// summaries drop the load-bearing details — decisions, constraints, dead ends
// and exact state — that stop a model re-treading them. Verbatim identifiers
// survive so a model can still grep for the file or id a compressed summary
// names. The scoping lines are load-bearing too: the history is handed over
// alone, and the newer turns follow the ledger unchanged, so the prompt must
// stop the model reaching past its chunk.
const compactSystem = "You are the compaction engine for a long-running coding agent. Your job " +
	"is to compress the older turns you are shown into a dense block that preserves everything " +
	"the working model still needs, so it can keep going as if those turns had happened — without " +
	"re-deriving them and without re-doing work. " +
	"Compress, do not reduce to a slogan. This is a compressed handoff of working memory, not a " +
	"prose recap, so keep the sharp edges that cause re-work: " +
	"the actual decisions made and the reasons, not just the conclusion; " +
	"constraints that must still hold — invariants, formats, interface contracts, security rules — " +
	"verbatim when short; " +
	"the state of the work — what exists, what is in flight, what was verified vs assumed; " +
	"dead ends and failed approaches, so the model does not re-try them; and " +
	"load-bearing identifiers verbatim — file paths, package and module names, function, class and " +
	"variable names, command invocations, tool and call ids, message ids, exact error strings, " +
	"version pins, and the config keys and values the work depends on. " +
	"Name the artifacts the work produced or touched, with their paths. " +
	"Structure the summary as short bullet sections, in exactly this order and only these: " +
	"Decisions, Constraints, Plan, State, Artifacts, Ruled out, Open questions. " +
	"Leave a section out if it is empty. Never add prose outside the bullets — no preamble, no " +
	"closing line. " +
	"Never invent facts that are not in the source: no guesses, no reconstructed numbers, no " +
	"unstated intentions. If something is genuinely ambiguous, record it under Open questions " +
	"instead of assuming. " +
	"Summarize only the turns shown to you. The newer turns after this chunk are preserved " +
	"verbatim elsewhere and will follow your summary unchanged, so do not anticipate, reference " +
	"or restate them — your summary must hand off the past without overlapping the present. " +
	"Be as short as correctness allows."

// compactAsk is the message tacked onto the history to ask for the summary. It
// repeats the chunk boundary because the model may lose the system's framing and
// try to \"recap the whole conversation\".
const compactAsk = "Above are the older turns to compact, and nothing else. Write the compaction " +
	"summary of exactly those turns, following your instructions. The conversation after this " +
	"chunk is kept intact and is not part of this request. Return only the summary block — no " +
	"preamble, no closing remark."

// compactTimeout bounds one summarizer call. A backend that hangs would
// otherwise hold the session at \"compacting\" forever, because settleCompaction
// runs only when the outcome arrives. A pass past the deadline is a failure:
// its partial summary is dropped and the mask fallback runs, which still
// frees the bulky tool output — so a spurious timeout costs an attempt, never
// correctness. Generous, because the pass replaces a large chunk (possibly
// hundreds of KB) with a short summary.
const compactTimeout = 120 * time.Second

// compactJudgeTimeout bounds the classification half of a pass. The judge is a
// short typed call and the client retries internally, so this is a backstop
// against a wedged endpoint, not a budget: the deadline is read like the
// summarizer's — a pass past it fails and the mask takes over.
const compactJudgeTimeout = 30 * time.Second

// summarizeInto runs one bounded summarizer call. The agent streams the chunk
// on a child context carrying a deadline, so a hung backend frees the session
// rather than holding it. A stream error, or a context whose deadline fired
// while the transport wound down cleanly, both come back as the error and the
// partial text is discarded: a transport that honours cancellation by stopping
// reports nothing through the stream, so the deadline must be read off the
// context itself.
func summarizeInto(parent context.Context, agent *nacelle.Agent, asks []nacelle.Message) (string, error) {
	local, cancel := context.WithTimeout(parent, compactTimeout)
	defer cancel()

	var b strings.Builder
	for event, err := range agent.Stream(local, asks) {
		if err != nil {
			return "", err
		}
		if event.Kind == nacelle.KindText {
			b.WriteString(event.Text)
		}
	}
	if localErr := local.Err(); localErr != nil {
		return "", localErr
	}
	return strings.TrimSpace(b.String()), nil
}
