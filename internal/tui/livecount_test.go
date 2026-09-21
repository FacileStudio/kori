package tui

import (
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

// The stream reports real usage only when a turn ends, so while the model is
// writing the output and context counters tick on an estimate of the deltas
// themselves: ~4 characters per token. This is what makes the footer live.
func TestStreamedTextTicksTheLiveOutputAndContextCounters(t *testing.T) {
	m := sized()
	m.run.busy = true
	m.size = 4000

	m.absorb(nacelle.Event{Kind: nacelle.KindThinking, Text: strings.Repeat("a", 40)})
	m.absorb(nacelle.Event{Kind: nacelle.KindText, Text: strings.Repeat("b", 40)})

	if m.run.liveOut != 20 {
		t.Fatalf("liveOut = %d, want 20 estimated output tokens", m.run.liveOut)
	}
	got := visible(strings.Join(m.footer(), " "))
	if !strings.Contains(got, "↓20") {
		t.Errorf("footer = %q, want the output counter ticking as the answer streams", got)
	}
	if !strings.Contains(got, "↕4.0k") {
		t.Errorf("footer = %q, want the context grown by the streamed output", got)
	}
}

// The estimate gives way the moment the turn reports its authoritative usage:
// the estimate is dropped, not added on top of the real count, or the two
// would double-count the same generation.
func TestTheLiveEstimateIsReplacedAtTheTurnBoundary(t *testing.T) {
	m := sized()
	m.run.busy = true
	m.size = 4000
	m.absorb(nacelle.Event{Kind: nacelle.KindText, Text: strings.Repeat("b", 40)})

	m.absorb(nacelle.Event{Kind: nacelle.KindTurn, Usage: nacelle.Usage{InputTokens: 400, OutputTokens: 30}})

	if m.run.liveOut != 0 {
		t.Errorf("liveOut = %d, want cleared when the turn reports its real usage", m.run.liveOut)
	}
	got := visible(strings.Join(m.footer(), " "))
	if !strings.Contains(got, "↓30") {
		t.Errorf("footer = %q, want the authoritative output count", got)
	}
	if strings.Contains(got, "↓50") {
		t.Errorf("footer = %q, want no estimate stacked on the real count", got)
	}
}

// The size both automatic triggers read is only ever written from a usage
// event, so a usage that reports no input at all must not erase it: that is what
// a backend which reports usage on some events but not others looks like, and
// writing the zero through would switch compaction off silently — the same
// stand-down the pre-send guard had when it read an uncountable backend as
// nothing to do. The last real reading is kept until a real one replaces it.
func TestAZeroUsageDoesNotEraseTheMeasuredSize(t *testing.T) {
	m := sized()
	m.size = 130_000

	m.sized(nacelle.Usage{})

	if m.size != 130_000 {
		t.Errorf("size = %d, want the last real reading kept", m.size)
	}

	m.sized(nacelle.Usage{InputTokens: 125_000})

	if m.size != 125_000 {
		t.Errorf("size = %d, want a real reading to replace it", m.size)
	}
}

// The dollar figure is live too, on the same honesty as the token counter: no
// turn has reported a Cost yet means no price at all; the first turn's real
// Cost seeds a realised rate (Cost per total billed token); and while the next
// turn streams, the footer shows the accumulated real Cost plus that rate
// scaled by the live output-token estimate, so the $ ticks up as tokens
// arrive. A backend that never reports a Cost keeps the rate at zero and no
// price is invented, and the authoritative Cost replaces the estimate the
// moment the next turn ends.
func TestLivePriceTicksFromTheRealisedRateAndLandsOnTheRealCost(t *testing.T) {
	m := sized()

	m.absorb(nacelle.Event{Kind: nacelle.KindText, Text: strings.Repeat("b", 40)})
	if got := visible(strings.Join(m.footer(), " ")); strings.Contains(got, "$") {
		t.Errorf("footer = %q, want no price before a turn has reported a Cost", got)
	}

	m.absorb(nacelle.Event{Kind: nacelle.KindTurn, Usage: nacelle.Usage{InputTokens: 400, OutputTokens: 30, Cost: 0.003}})
	if got := visible(strings.Join(m.footer(), " ")); !strings.Contains(got, "$0.0030") {
		t.Errorf("footer = %q, want the first turn's authoritative Cost", got)
	}

	m.absorb(nacelle.Event{Kind: nacelle.KindText, Text: strings.Repeat("b", 40)})
	streaming := visible(strings.Join(m.footer(), " "))
	if !strings.Contains(streaming, "$0.0031") {
		t.Errorf("footer = %q, want the live estimate on top of the real Cost", streaming)
	}
	if !strings.Contains(streaming, "↓40") {
		t.Errorf("footer = %q, want the live output estimate still streaming", streaming)
	}

	m.absorb(nacelle.Event{Kind: nacelle.KindTurn, Usage: nacelle.Usage{InputTokens: 300, OutputTokens: 10, Cost: 0.004}})
	landed := visible(strings.Join(m.footer(), " "))
	if !strings.Contains(landed, "$0.0070") {
		t.Errorf("footer = %q, want the authoritative Cost to replace the estimate at the turn end", landed)
	}
	if strings.Contains(landed, "$0.0031") {
		t.Errorf("footer = %q, want no estimate stacked on the real cost", landed)
	}
}

// A /parallel fan-out's live spend joins the session total as each nested turn
// streams, so the footer's own counters move in real time instead of only when
// the last task completes.
func TestDetachedLiveSpendJoinsTheSessionTotalAsItStreams(t *testing.T) {
	m := sized()
	batch := "detach7"
	m.parallelTasks = map[string][]parallelTaskInfo{batch: {{Task: "one", Active: true}}}

	m.recordUpdate(subagentUpdate{batch: batch, idx: 0, usage: nacelle.Usage{OutputTokens: 50, Cost: 0.005}, spend: true})

	if m.spent.OutputTokens != 50 || m.spent.Cost != 0.005 {
		t.Fatalf("spent = %+v, want the live spend already folded into the session total", m.spent)
	}
	pt := &m.parallelTasks[batch][0]
	if pt.Ledgered.OutputTokens != 50 {
		t.Errorf("ledgered = %+v, want the folded amount remembered against double-counting", pt.Ledgered)
	}

	m.finishDetached(pt, detachedResult{batch: batch, idx: 0, result: "r", usage: nacelle.Usage{OutputTokens: 80, Cost: 0.009}})
	if m.spent.OutputTokens != 80 || m.spent.Cost != 0.009 {
		t.Errorf("spent = %+v, want the residual, not the whole figure, added on completion", m.spent)
	}
	if pt.Active {
		t.Error("task still active after finishing")
	}
}

// A model-called parallel_agents already reaches the session total through
// nacelle's Usage hook (the delegations channel), so folding its live spend in
// here too would double-count it. Only detached batches fold live.
func TestModelPathLiveSpendStaysOutOfTheSessionTotal(t *testing.T) {
	m := sized()
	m.parallelTasks = map[string][]parallelTaskInfo{"tool-1": {{Task: "one", Active: true}}}

	m.recordUpdate(subagentUpdate{batch: "tool-1", idx: 0, usage: nacelle.Usage{OutputTokens: 50}, spend: true})

	if m.spent.OutputTokens != 0 {
		t.Errorf("spent = %+v, want the model path's live spend kept out (it reaches spent via delegations)", m.spent)
	}
}
