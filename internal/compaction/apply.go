package compaction

import (
	"slices"

	"github.com/FacileStudio/nacelle"
)

// Stats is what one pass did, in the units a report speaks: estimated tokens
// before and after, how many history turns the rebuild dropped, and whether the
// rebuild was refused. Tombstoning is not counted here — the mask reports its own
// numbers back to the caller that ran it.
type Stats struct {
	Before, After int64
	Summarized    int
	// Refused reports that the assembly came out heavier than the conversation it
	// would have replaced, so the original was handed back untouched. It is I5 in
	// the one place the estimate can move the wrong way: a judged pass can fold a
	// two-byte turn and get a verbose ledger back, and a pass that adds weight is
	// not a pass. A refused pass reports Before == After and no work.
	Refused bool
	// Stale reports that the plan no longer described the conversation it was
	// measured against, so nothing was assembled and the original stands. A plan
	// and a conversation travel together but are installed at different moments,
	// so this is the one way a pass can be handed a plan that does not bound it —
	// and indexing one conversation with another's spans would either run off its
	// end or silently replace it with nothing.
	Stale bool
	// LedgerReplaced reports that the ledger was rewritten wholesale, which only
	// a consolidating pass is allowed to do: the body it was asked to rewrite had
	// outgrown its budget and came back carrying every identifier the old one
	// named. It is how a ledger that only ever grew gets smaller again.
	LedgerReplaced bool
	// LedgerKept reports that a consolidating pass was asked for and refused,
	// because the rewrite had stopped carrying identifiers the ledger held. The
	// merge stands instead, so the body grows rather than losing a fact the next
	// pass could not recover. Growth is the failure mode this leaves open, and it
	// is the deliberate one: the next pass can attack a longer ledger, nobody can
	// attack a forgotten one.
	LedgerKept bool
}

// Apply reassembles a conversation after a pass: the pinned anchor verbatim, one
// ledger message in place of what was folded, the surviving history blocks, and
// the active window untouched. It is the deterministic half of a pass, so a test
// can drive it without a model. The anchor is copied byte for byte (I2), roles
// alternate in the result (I4), and a rebuild that would grow the conversation is
// refused outright (I5), handing the original back with Stats.Refused set.
//
// replace is the one bit of the summarizer's intent the caller has to pass in:
// a pass that asked for a consolidated ledger may rewrite the body it was
// written from, a pass that only added to it may not. Even then the rewrite is
// conditional — NextLedger grants it only when the new body still carries every
// identifier the old one named — so the flag selects the attempt, never the
// outcome.
//
// The one precondition is that the plan still covers conv: a plan measured
// against a conversation that is no longer there cannot be applied to it, so it
// comes back as Stats.Stale with the original untouched. That check lives here,
// at the boundary, so no caller has to remember it — the assembly below assumes
// it has already held.
func Apply(conv []nacelle.Message, plan []Span, ledger string, keep func(int) bool, replace bool) ([]nacelle.Message, Stats) {
	if !Covers(conv, plan) {
		before := EstTokens(Bytes(conv))
		return conv, Stats{Before: before, After: before, Stale: true}
	}
	return assemble(conv, plan, ledger, keep, replace)
}

// assemble is Apply's body: the plan is known to cover conv. keep selects the
// history indices that survive in place; nil drops the whole history, which is
// what an unclassified pass does. A call that changes nothing — no history to
// drop, nothing kept, no ledger text and no previous ledger — is handed back
// untouched rather than rewritten to close a boundary it never created.
//
// Every other call installs a ledger, even with no word to put in one, because
// that message is the buffer that keeps the pinned head and whatever follows it
// role-legal: the head's last turn and the next turn can share a role, and the
// only ways out of that are merging the next turn into the head — which would
// rewrite the anchor (I2) — or standing a ledger between them even when it has
// nothing to say.
func assemble(conv []nacelle.Message, plan []Span, ledger string, keep func(int) bool, replace bool) ([]nacelle.Message, Stats) {
	anchor := Section(conv, plan, ZoneAnchor)
	active := Section(conv, plan, ZoneActive)
	surviving := survivingHistory(conv, plan, keep)
	dropped := len(HistoryMessages(conv, plan)) - len(surviving)
	previous := LedgerText(conv, plan)
	carryParts, carryMsgs := ledgerCarry(conv, plan)

	if dropped == 0 && len(surviving) == 0 && ledger == "" && len(carryParts) == 0 && len(carryMsgs) == 0 &&
		!slices.ContainsFunc(plan, func(span Span) bool { return span.Zone == ZoneLedger }) {
		return unchanged(conv, anchor, active)
	}

	body, replaced := NextLedger(previous, ledger, replace)
	built := newLedgerMessage(body)
	built.Parts = append(built.Parts, carryParts...)
	built.Role = ledgerRole(anchor, following(carryMsgs, surviving, active))

	out := make([]nacelle.Message, 0, len(anchor)+len(carryMsgs)+len(surviving)+len(active)+1)
	out = append(out, anchor...)
	out = append(out, built)
	out = append(out, carryMsgs...)
	out = append(out, surviving...)
	out = append(out, active...)

	out = alternateFrom(out, len(anchor))

	stats := measure(conv, out, dropped)
	stats.LedgerReplaced, stats.LedgerKept = replaced, replace && !replaced && ledger != ""
	if stats.After > stats.Before {
		return conv, Stats{Before: stats.Before, After: stats.Before, Refused: true}
	}
	return out, stats
}

// unchanged is the conversation handed back when a call has nothing to do: the
// pinned head and the active window, with no ledger standing between them. It is
// the one shape that gets no buffer, and the reason is in assemble's own doc
// comment — nothing was dropped, so there is no boundary to close.
func unchanged(conv, anchor, active []nacelle.Message) ([]nacelle.Message, Stats) {
	out := make([]nacelle.Message, 0, len(anchor)+len(active))
	out = append(out, anchor...)
	out = append(out, active...)
	return out, measure(conv, out, 0)
}

// ledgerCarry is what the ledger zone holds beyond its own text: the extra parts
// a same-role neighbour folded into the ledger message, and the reply messages
// answering the ledger's own tool calls. Re-emitting both with the rebuilt
// ledger is what stops an absorbed call — or an absorbed fact — from vanishing
// on the next pass, when the ledger is otherwise rebuilt from its text alone.
func ledgerCarry(conv []nacelle.Message, plan []Span) ([]nacelle.Part, []nacelle.Message) {
	var parts []nacelle.Part
	var msgs []nacelle.Message
	for _, span := range plan {
		if span.Zone != ZoneLedger || span.Start >= len(conv) {
			continue
		}
		parts = ExtraParts(conv[span.Start])
		for i := span.Start + 1; i < span.End && i < len(conv); i++ {
			msgs = append(msgs, conv[i])
		}
	}
	return parts, msgs
}

// following is every message that will sit after the ledger, in order, so the
// role pick can alternate with the first of them.
func following(groups ...[]nacelle.Message) []nacelle.Message {
	var out []nacelle.Message
	for _, group := range groups {
		out = append(out, group...)
	}
	return out
}

// survivingHistory is the history messages keep selects, in order. A nil
// predicate keeps nothing.
func survivingHistory(conv []nacelle.Message, plan []Span, keep func(int) bool) []nacelle.Message {
	if keep == nil {
		return nil
	}
	var out []nacelle.Message
	for _, span := range plan {
		if span.Zone != ZoneHistory {
			continue
		}
		for i := span.Start; i < span.End && i < len(conv); i++ {
			if keep(i) {
				out = append(out, conv[i])
			}
		}
	}
	return out
}

// ledgerRole picks the ledger's role so it never collides with the message
// before it: the anchor's last turn when there is one, otherwise the first
// message that will follow it. With neither neighbour a user role keeps the
// conversation starting legally.
func ledgerRole(anchor, following []nacelle.Message) nacelle.Role {
	if len(anchor) > 0 {
		return opposite(anchor[len(anchor)-1].Role)
	}
	if len(following) > 0 {
		return opposite(following[0].Role)
	}
	return nacelle.RoleUser
}

// alternateFrom folds any same-role neighbours starting at index from, moving
// the later message's parts into the earlier one. Starting at from keeps the
// anchor untouched, and the ledger is what makes that hold: its role is chosen
// opposite the anchor's last turn, so the boundary above the anchor can never
// collide and only the ledger, the surviving blocks and the active window can
// merge with each other.
func alternateFrom(msgs []nacelle.Message, from int) []nacelle.Message {
	out := make([]nacelle.Message, 0, len(msgs))
	out = append(out, msgs[:from]...)
	for _, msg := range msgs[from:] {
		if len(out) > 0 && out[len(out)-1].Role == msg.Role {
			last := &out[len(out)-1]
			last.Parts = append(last.Parts, msg.Parts...)
			continue
		}
		out = append(out, msg)
	}
	return out
}
