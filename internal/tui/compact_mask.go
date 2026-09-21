package tui

import "github.com/FacileStudio/kori/internal/compaction"

const (
	// compactMinResult, droppedNotice and droppedThinkingNotice alias the
	// compaction package's tombstone shape, so the report and the tests read the
	// same strings the pass writes.
	compactMinResult      = compaction.MinResult
	droppedNotice         = compaction.DroppedNotice
	droppedThinkingNotice = compaction.DroppedThinkingNotice

	// compactSlack is how far under the trigger one pass aims to land, so a
	// session that grows between passes does not trim on every turn.
	compactSlack = 20_000
)

// estTokens delegates to the package's bytes-to-tokens estimate, kept as a name
// because the whole TUI reads it.
func estTokens(bytes int) int64 {
	return compaction.EstTokens(bytes)
}

// maskHistory runs the deterministic tombstone over the plan's history spans and
// debits the session size by the byte estimate. The debit is deliberately an
// estimate: the next sized() overwrites it with the backend's own count, so the
// two never drift past the turn boundary.
func (m *Model) maskHistory(plan []compaction.Span) compaction.MicroStats {
	stats := compaction.Tombstone(m.conversation, plan)
	if stats.Bytes == 0 {
		return stats
	}
	m.trimmed += stats.Results + stats.Thinking
	m.size = max(m.size-estTokens(stats.Bytes), 0)
	return stats
}

// applyMaskFallback runs the tombstone in place on the UI thread for the two
// cases where the summary did not happen: the summarizer errored, or produced
// nothing. The mask still frees the bulky tool output, so a failed summary costs
// nothing but the attempt.
func (m *Model) applyMaskFallback(outcome compactOutcome) {
	before := m.size
	stats := m.maskHistory(outcome.plan)
	start, end, _ := compaction.HistoryRange(outcome.plan)
	report := compactOutcome{
		before: before,
		after:  m.size,
		done: compacted{
			evictCut: end - start,
			kept:     len(m.conversation) - end,
			results:  stats.Results,
			thinking: stats.Thinking,
			tier:     outcome.tier,
		},
	}
	m.last = report.done
	m.say(fromCompact, compactReport(report))
}
