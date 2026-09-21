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
// figure: the size against the window with the tier it has reached, and the
// accumulated ledger with the tier of the last pass that wrote it. A session
// with nothing to say — no size measured, no ledger yet — adds no lines.
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
	return lines
}
