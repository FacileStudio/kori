package tui

import (
	"fmt"

	"github.com/FacileStudio/kori/internal/compaction"
)

// contextLoad is the status surface's name for the conversation's size against
// the window it is measured on: `↕120k/200k · 0.60` once the backend reports a
// window, a plain `↕120k` when it does not, with the tier appended once the size
// has crossed one. One shape, so the footer and /status cannot disagree about
// how full the context is.
func contextLoad(size int64, policy compaction.Policy) string {
	load := "↕" + shortTokens(size)
	if policy.Window > 0 {
		load += "/" + shortTokens(policy.Window)
		load += fmt.Sprintf(" · %.2f", float64(size)/float64(policy.Window))
	}
	if tier := policy.Tier(size); tier != compaction.Below {
		load += " · " + tier.String()
	}
	return load
}

// compactionLines is what /status adds about the ladder beyond the footer's own
// figure: the size against the window with the tier it has reached, the
// accumulated ledger with the tier of the last pass that wrote it, and the judge
// when it is on. A session with nothing to say — no size measured, no ledger yet,
// no judge — adds no lines.
//
// The judge gets a line of its own because it is the one setting that sends the
// conversation off the machine, and it is otherwise invisible once enabled: a
// reader who opted in months ago has nothing to remind them that every pass
// ships the history somewhere. It sits with the ladder because it is part of it —
// the judge is what decides which turns the fold is allowed to take.
func (m *Model) compactionLines() []string {
	var lines []string
	if m.size > 0 {
		lines = append(lines, contextLoad(m.size, m.policy))
	}
	if ledger := compaction.LedgerText(m.conversation, m.plan()); ledger != "" {
		line := "ledger · ~" + shortTokens(compaction.EstTokens(len(ledger))) + " tokens"
		if m.last.tier != compaction.Below {
			line += " · last pass " + m.last.tier.String()
		}
		lines = append(lines, line)
	}
	if m.judge != nil {
		lines = append(lines, "judge · on — each pass sends the history off the machine")
	}
	return lines
}
