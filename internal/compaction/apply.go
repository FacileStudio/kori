package compaction

import "github.com/FacileStudio/nacelle"

// Stats is what one pass did, in the two units a report speaks: estimated tokens
// before and after, how many messages were tombstoned, dropped from the history,
// and the tier that ran.
type Stats struct {
	Before, After int64
	Masked        int
	Summarized    int
	Tier          Tier
}

// Apply reassembles a conversation after a pass: the pinned anchor verbatim, one
// ledger message in place of what was folded, the surviving history blocks, and
// the active window untouched. It is the deterministic half of a pass, so a test
// can drive it without a model. The anchor is copied byte for byte (I2), roles
// alternate in the result (I4), and the estimate never grows (I5).
//
// keep selects the history indices that survive in place; nil drops the whole
// history, which is what an unclassified pass does. A degenerate call with
// neither a new ledger nor a previous one and nothing surviving hands the ends
// back untouched rather than merging an active turn into the anchor to close a
// boundary no pass created.
func Apply(conv []nacelle.Message, policy Policy, plan []Span, ledger string, keep func(int) bool) ([]nacelle.Message, Stats) {
	anchor := Section(conv, plan, ZoneAnchor)
	active := Section(conv, plan, ZoneActive)
	surviving := survivingHistory(conv, plan, keep)
	previous := LedgerText(conv, plan)
	carryParts, carryMsgs := ledgerCarry(conv, plan)
	hasLedger := ledger != "" || previous != "" || len(carryParts) > 0 || len(carryMsgs) > 0

	out := make([]nacelle.Message, 0, len(anchor)+len(carryMsgs)+len(surviving)+len(active)+1)
	out = append(out, anchor...)
	if !hasLedger && len(surviving) == 0 {
		out = append(out, active...)
		return out, measure(conv, out, policy, 0)
	}
	if hasLedger {
		built := BuildLedger(previous, ledger)
		built.Parts = append(built.Parts, carryParts...)
		built.Role = ledgerRole(anchor, following(carryMsgs, surviving, active))
		out = append(out, built)
	}
	out = append(out, carryMsgs...)
	out = append(out, surviving...)
	out = append(out, active...)

	out = alternateFrom(out, len(anchor))
	return out, measure(conv, out, policy, len(HistoryMessages(conv, plan))-len(surviving))
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

func opposite(role nacelle.Role) nacelle.Role {
	if role == nacelle.RoleUser {
		return nacelle.RoleAssistant
	}
	return nacelle.RoleUser
}

// alternateFrom folds any same-role neighbours starting at index from, moving
// the later message's parts into the earlier one. Starting at from keeps the
// anchor untouched whatever the ledger looks like, which is what makes the fold
// safe: only the ledger, the surviving blocks and the active window can merge.
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

func measure(conv, out []nacelle.Message, policy Policy, summarized int) Stats {
	before := EstTokens(Bytes(conv))
	return Stats{
		Before:     before,
		After:      EstTokens(Bytes(out)),
		Summarized: summarized,
		Tier:       policy.Tier(before),
	}
}
