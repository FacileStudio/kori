// Package tui — the snapshot one compaction pass runs on.
package tui

import (
	"slices"

	"github.com/FacileStudio/kori/internal/compaction"
	"github.com/FacileStudio/nacelle"
)

// compactPass is everything one pass needs, snapshotted on the update loop
// before the pass's goroutine starts. A pass is the one thing in this package
// that reads a conversation off the update loop's goroutine, so it is handed its
// own view instead of reaching into the model: a conversation the reader's next
// command cannot shorten under it, and a size the next sized() cannot move.
//
// That keeps the read safety local to this struct. It used to be an invariant
// spread across every command that might mutate the session mid-pass — ask
// queues while compacting, the pre-send path leaves the run busy — which held but
// was invisible at the point it mattered. What is left for the runtime to catch is
// the case the invariant never covered: the live conversation really did move on,
// and Apply's Covers check is what refuses a plan measured against it.
type compactPass struct {
	conv  []nacelle.Message
	size  int64
	plan  []compaction.Span
	tier  compaction.Tier
	judge compaction.Judge
	agent *nacelle.Agent
}

// pass snapshots the session for one pass. The conversation is cloned so the
// goroutine owns its own slice header and backing array; the messages it points
// at stay read-only for the duration, because Tombstone — the one thing here that
// edits a message in place — runs on the update loop and never inside a pass.
//
// The summarizer is built here rather than in the goroutine for the same reason:
// it reads m.agent, and a pass must not touch the model at all.
func (m *Model) pass(plan []compaction.Span, tier compaction.Tier) compactPass {
	return compactPass{
		conv:  slices.Clone(m.conversation),
		size:  m.size,
		plan:  plan,
		tier:  tier,
		judge: m.judge,
		agent: m.summarizer(),
	}
}
