package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/FacileStudio/nacelle"

	"github.com/FacileStudio/kori/internal/sessions"
	"github.com/FacileStudio/kori/internal/usage"
)

// account holds what the client knows about what the session has spent: the
// usage total, and what it knows about how much context the conversation is
// carrying.
//
// The two live together because they answer one question from two sides —
// what the session cost, and whether the conversation is now too heavy to
// continue as-is — and both outlive any single run.
type account struct {
	// spent is the session's cumulative usage; the status line adds the run
	// in flight to it into a total that only ever goes up.
	spent nacelle.Usage

	// rate is the realised cost per token of the most recent turn that
	// reported one: its Cost divided by its total billed tokens. The status
	// line multiplies the live output-token estimate by it while a turn
	// streams, so the dollar figure ticks without waiting for the turn to
	// end. It stays zero until a turn reports a Cost — and stays that way on
	// backends that never report one, so no dollars are invented — and the
	// authoritative per-turn Cost replaces the estimate at the next turn
	// boundary.
	rate float64

	// size is the input cost of the most recent finished turn, in tokens.
	// Every turn re-bills the whole conversation as input, so this is also
	// what the conversation would cost to send again right now — measured,
	// by the backend's own accounting, not guessed from bytes.
	size int64

	// trimmed is how many tool results and thinking blocks have been masked.
	// It reaches the status line, because a model whose memory was quietly
	// edited should not be the only one who knows.
	trimmed int

	// compactBegan is when the current compaction pass started, stamped in
	// beginCompaction. The running-tool row draws its elapsed time against
	// it, the same way every other live row draws against its group's start.
	compactBegan time.Time

	// began is when this client started, stamped once in newModel. The
	// status line and the closing recap both measure against it, so the
	// two can never disagree about how long the session ran.
	began time.Time

	// tools and failed are how many tool calls this session finished, and
	// how many of those fell over. They are counted where a call ends
	// rather than derived at the end from anything, because nothing keeps
	// a record of a finished call: the line is printed and forgotten.
	tools  int
	failed int

	// sink appends each finished turn to mycelium's event feed, so a session
	// running here shows up in mycelium's dashboard while it is still going.
	// It is nil when mycelium is not installed on this machine.
	sink *usage.Sink

	// session is this run of the client written down: the questions asked
	// and the answers given, appended to a file under ~/.kori/sessions
	// as they are said. It is nil when the file could not be opened, which
	// is not a reason to refuse to run.
	session *sessions.SessionLog

	// tasks is the plan the model is working to, as it last reported it.
	// It is written only by the routed update, never by the tool that
	// produces it — see tasks.go for why a tool goroutine cannot touch it.
	tasks taskList

	// grind is the per-run minimum spend this session demands, and how many
	// continuations the current run has used of it. With no floor set it is
	// off and nothing here ever fires.
	grind grindBudget
}

// compactFinished says the compaction channel closed and no outcome arrived.
type compactFinished struct{}

const (
	// compactKeepMessages is how many of the newest messages are never
	// touched. What the model is reasoning about right now lives here;
	// dropping it would save tokens and lose the plot.
	compactKeepMessages = 4

	// compactKeepPercent is how much of a long conversation stays verbatim.
	// Compaction is a hybrid: the newest this fraction of turns is kept
	// intact and the older middle is summarized, so the model keeps its feet
	// in the recent thread while the distant past is compressed.
	compactKeepPercent = 25

	// compactMaxTokens is the ceiling a compaction summary is asked to stay
	// under. Small on purpose: the summary replaces a large middle with a
	// stub, so a long summary would buy almost nothing.
	compactMaxTokens = 2000
)

// sized records what a finished turn cost on the input side.
//
// Cache reads and cache creations are billed input like any other, and both
// backends report them here, so leaving either out would understate the
// conversation by most of an agentic session's real size.
func (m *Model) sized(usage nacelle.Usage) {
	m.size = usage.InputTokens + usage.CacheReadTokens + usage.CacheCreationTokens
}

// keepCount is how many of the newest messages survive a pass verbatim: at
// least compactKeepMessages, and a share of a long conversation beyond that.
// A conversation no longer than the floor is kept whole, so a short session is
// never compacted. evictCut, the number of oldest messages a pass may touch,
// is len minus it.
func keepCount(length int) int {
	if length <= compactKeepMessages {
		return length
	}
	n := length * compactKeepPercent / 100
	n = min(max(n, compactKeepMessages), length-1)
	return n
}

// beginCompaction runs one pass. It is called from send (right before the
// model needs the freed context), from settle when a turn ends with the
// context over threshold and nothing queued, and from the manual /compact
// command. It is where the running-tool row and the purple status come from:
// a pass is an LLM call now, so it must not block the update loop. When the
// evicted middle is too small to plausibly land the conversation under the
// threshold, the pass skips the summarizer and masks what little old turns
// hold instead — see evictionCanLandUnder/maskOnlyPass in compact_light.go.
//
// The pass is two-stage and tiered, the way the research on long-running
// agents lands. The summarization stage runs first, on its own goroutine,
// against the raw evicted middle — it must read the actual tool output and
// decisions, not the mask's placeholders, so the summary can preserve exactly
// what compaction would otherwise discard. The mask stage is the fallback: if
// there is no backend, or the summary call fails, or comes back empty, the UI
// thread masks the oversized tool results and thinking blocks in the evicted
// middle as a cheap headroom lever. Either way a pass never hands a model more
// context than it started with, and only the newest share of messages is kept
// verbatim in both stages.
//
// Nothing is recorded to the session file — this is the client talking about
// itself, the same reason the banner is not.
func (m *Model) beginCompaction(ctx context.Context) tea.Cmd {
	evictCut := alignedEvictCut(m.conversation, len(m.conversation)-keepCount(len(m.conversation)))
	if evictCut <= 0 || m.compacting {
		return nil
	}
	if !m.evictionCanLandUnder(evictCut) {
		return m.maskOnlyPass(evictCut)
	}

	m.compacting = true
	m.compactBegan = time.Now()

	resultsChan := make(chan compactOutcome)
	m.run.compactChan = resultsChan

	go runCompaction(m, ctx, resultsChan, evictCut)
	return tea.Batch(waitForCompact(resultsChan), m.spin.Tick)
}

// runCompaction is the pass's own goroutine. It asks the backend for a
// summary of the raw evicted middle and sends back just that result; it never
// mutates the conversation. Because the conversation is stable while a pass is
// in flight — send holds the run busy — reading it here is safe.
//
// The summarizer runs inside a deadline set by summarizeInto, so a wedged
// backend cannot hold the session at "compacting" forever: whichever way the
// stream winds down once the deadline fires, the outcome still arrives and
// the pass falls back to the mask.
func runCompaction(m *Model, ctx context.Context, results chan compactOutcome, evictCut int) {
	defer close(results)
	conv := m.conversation
	outcome := compactOutcome{before: m.size, evictCut: evictCut}

	if agent := m.summarizer(); agent != nil {
		summary, err := summarizeInto(ctx, agent, compactPrompt(conv, evictCut))
		if err != nil {
			outcome.err = err
		} else {
			outcome.summary = summary
		}
	}

	results <- outcome
}

// summarizer is in compact_light.go, beside the light lever and the pass-skip
// gate that decide whether it is spent.

// settleCompaction installs a finished pass and starts the run that was
// waiting on the freed context, or on the idle path sends the lines the
// reader typed during the pass. A summary replaces the evicted middle or the
// mask stands, and it never grows the conversation. Delivery lives here, not
// chained in settle, where a detached sequence would race this install.
func (m *Model) settleCompaction(outcome compactOutcome) tea.Cmd {
	m.compacting = false
	m.run.compactChan = nil

	evictCut := outcome.evictCut

	if outcome.err != nil {
		m.applyMaskFallback(outcome)
		m.say(fromCompact, "compaction summary failed · "+outcome.err.Error()+" — masked instead")
	} else if outcome.summary != "" {
		done := compactApply(m.conversation, evictCut, outcome.summary, outcome.before)
		m.conversation = done.conversation
		m.size = done.after
		m.say(fromCompact, compactReport(done))
	} else {
		m.applyMaskFallback(outcome)
		m.say(fromCompact, "compaction summary came back empty — masked instead")
	}

	m.checkThrash()

	if m.run.busy && m.agent != nil {
		return m.startRun(m.run.bgCtx)
	}
	return m.deliver()
}

// waitForCompact takes exactly one outcome and re-arms itself from Update,
// the same contract waitFor holds for run results.
func waitForCompact(results <-chan compactOutcome) tea.Cmd {
	return func() tea.Msg {
		next, open := <-results
		if !open {
			return compactFinished{}
		}
		return next
	}
}
