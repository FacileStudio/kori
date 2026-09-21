// Package tui — the light lever, the tier dispatch and the thrash guard of
// context compaction.
package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/FacileStudio/kori/internal/compaction"
	"github.com/FacileStudio/nacelle"
)

// thrashLimit is how many consecutive compaction passes must fail to land the
// conversation under the trigger threshold before the automatic triggers back
// off. A single failed pass is not a reason to stop trying — it can be a
// transient summarizer rejection — so the guard only stands down once the
// pattern is clear that a pass cannot help. Three is Claude Code's own number
// for the same guard.
const thrashLimit = 3

// summarizer builds the small, tool-free agent asked to compact the history,
// billed like the work it protects, and nil when there is no backend — tests and
// offline runs tombstone instead. It sends no reasoning parameter at all: an
// explicit reasoning-off makes endpoints where reasoning is mandatory answer
// 400, while omitting the key keeps reasoning off on models that default off and
// lets a mandatory model spend its own default. The budget is the summary's
// either way.
func (m *Model) summarizer() *nacelle.Agent {
	if m.agent == nil {
		return nil
	}
	agent, err := nacelle.New(nacelle.Config{
		Backend:       m.agent.Backend(),
		System:        compactSystem,
		Thinking:      nacelle.Thinking{Show: false},
		MaxTokens:     compactMaxTokens,
		MaxIterations: 1,
	})
	if err != nil {
		return nil
	}
	return agent
}

// evictionCanLandUnder is whether folding the whole history into a free ledger
// could plausibly land the conversation under the ceiling. When the pinned ends
// alone are already over — one giant active tool result, or several — a pass
// never touches them, so no summarizer can help. Spending a 120-second LLM call
// on a pass that cannot land under is waste; the honest move is to tombstone
// what little the old turns hold and tell the reader the cost is theirs to free
// with /clear or chunked reading.
func (m *Model) evictionCanLandUnder(start, end int) bool {
	return m.size-estTokens(compaction.Bytes(m.conversation[start:end]))+compactMaxTokens <= m.compactAt
}

// maskOnlyPass is the skip for a pass whose history is too small to justify the
// summarizer: it tombstones whatever droppable output the old turns hold,
// reports that the cost lives in the pinned ends, and returns nil so no
// summarizer goroutine is spawned. It counts toward the thrash guard when the
// context stays over the threshold, so the automatic triggers back off instead
// of repeating the near-no-op pass on every turn.
func (m *Model) maskOnlyPass(plan []compaction.Span) tea.Cmd {
	before := m.size
	tier := m.policy.Tier(before)
	stats := m.maskHistory(plan)
	start, end, _ := compaction.HistoryRange(plan)
	report := compactOutcome{
		before: before,
		after:  m.size,
		done:   compacted{evictCut: end - start, kept: len(m.conversation) - end, results: stats.Results, tier: tier},
	}
	m.last = report.done
	m.say(fromCompact, compactReport(report)+"\n   cost sits in the kept tail — compaction protects the newest turns; /clear or read in chunks")
	m.checkThrash()
	return nil
}

// softPass is the soft tier: a deterministic tombstone of the history with no
// model call and no goroutine. It reports only when it actually freed something,
// and it does not count toward the thrash guard — it is free, so repeating it
// costs nothing.
func (m *Model) softPass() tea.Cmd {
	plan := m.plan()
	before := m.size
	stats := m.maskHistory(plan)
	if stats.Results == 0 {
		return nil
	}
	start, end, _ := compaction.HistoryRange(plan)
	report := compactOutcome{
		before: before,
		after:  m.size,
		done: compacted{
			evictCut: end - start,
			kept:     len(m.conversation) - end,
			results:  stats.Results,
			tier:     compaction.Soft,
		},
	}
	m.last = report.done
	m.say(fromCompact, compactReport(report))
	return nil
}

// compactTiered selects the pass a measured size has earned: a free tombstone at
// the soft tier, a full summarize at mid or hard, and nothing below. It is the
// single dispatch both automatic triggers and the pre-send path share.
func (m *Model) compactTiered(ctx context.Context) tea.Cmd {
	switch m.policy.Tier(m.size) {
	case compaction.Soft:
		return m.softPass()
	case compaction.Mid, compaction.Hard:
		return m.beginCompaction(ctx)
	default:
		return nil
	}
}

// thrashed is whether the automatic triggers should back off: enough consecutive
// passes have failed to land the conversation under the threshold that repeating
// a pass is more likely to waste a call than to help. A manual /compact still
// forces a fresh attempt, and a pass that lands under, or a /clear, resets it.
func (m *Model) thrashed() bool {
	return m.thrashCount >= thrashLimit
}

// checkThrash closes a pass that left the conversation still over the trigger
// threshold. Rather than backing off on the first miss, it counts: each pass that
// fails to land under increments, and each one that does resets. Only when the
// count reaches thrashLimit does it warn and stand the automatic triggers down —
// a single enormous result, usually in the pinned ends a pass never touches, is
// the class a pass cannot clear, and repeating it cannot help. What a reader can
// act on is named in the notice: /clear, or reading in chunks, then a manual
// /compact.
func (m *Model) checkThrash() {
	if m.compactAt > 0 && m.size > m.compactAt+compactSlack {
		m.thrashCount++
		if m.thrashCount == thrashLimit {
			m.say(fromClient, "compaction keeps leaving the context over the threshold — one very large result is likeliest; /clear, read in chunks, then /compact")
		}
		return
	}
	m.thrashCount = 0
}
