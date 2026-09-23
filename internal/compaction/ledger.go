package compaction

import (
	"strings"

	"github.com/FacileStudio/nacelle"
)

// Sentinel opens the one message that accumulates what a pass must not forget.
// It is the ledger's identity: IsLedger reads it, and a later pass folds into
// the message carrying it rather than ever summarizing it again.
//
// It ends in the private-use codepoint U+E000, and that codepoint is the whole
// discriminator. The identity it grants is the strongest this package hands out:
// a ledger is folded into and never pruned. The text it is read from is text a
// reader supplies, so a turn that spells the marker the way a document prints it,
// or the way a person types it, has to read as the ordinary history it is. No
// keyboard produces a private-use character, so an accidental paste and a quoted
// marker both fail to carry one. That is the bar this meets, and no text marker
// can meet a higher one: a reader who copies this constant out of the source can
// forge the identity, which is a deliberate act rather than the paste the marker
// exists to refuse.
const Sentinel = "[state ledger]\uE000"

// sentinelLine splits a text that opens with the sentinel from the body behind
// it, with ok false for a text that does not. The sentinel has to be the whole
// first line and it has to be the one this package writes, codepoint included:
// a ledger is what a pass folds into and never prunes, so whatever matches the
// marker stops being history, and a turn quoting it, or a paste opening with the
// typed form of it, would take that identity and leave the real ledger behind it
// reading as a prune candidate. Nothing the package writes stops matching, since
// every ledger it builds opens with the sentinel alone on its first line.
func sentinelLine(text string) (string, bool) {
	line, body, _ := strings.Cut(strings.TrimSpace(text), "\n")
	if strings.TrimSpace(line) != Sentinel {
		return "", false
	}
	return strings.TrimSpace(body), true
}

// ledgerBody is the first sentinel-bearing text part's body, with ok false for a
// message that carries no ledger. The marker is the whole identity and nothing
// else is: the role is not part of it, because assembly picks the ledger's role
// as the opposite of its neighbour, so a conversation anchored on an assistant
// turn is folded into a user-role ledger and a later pass has to recognise that
// one too. Nor can the identity be recorded outside the text, which is why the
// marker is what a reader cannot type: nacelle.Message is a role and a list of
// parts, and Part cannot be joined from this package, so there is no field to
// carry a marker on and no part type to carry one in.
func ledgerBody(m nacelle.Message) (string, bool) {
	for _, part := range m.Parts {
		text, ok := part.(nacelle.Text)
		if !ok {
			continue
		}
		if body, ok := sentinelLine(text.Text); ok {
			return body, true
		}
	}
	return "", false
}

// IsLedger reports whether a message carries the state-ledger sentinel.
func IsLedger(m nacelle.Message) bool {
	_, ok := ledgerBody(m)
	return ok
}

// Body is a ledger message's content with the sentinel stripped, empty for a
// message that carries none.
func Body(m nacelle.Message) string {
	body, _ := ledgerBody(m)
	return body
}

// BuildLedger returns the single message that carries the ledger: the sentinel,
// then the previous ledger merged with the new summary. The merge is monotone — a
// ledger built from an earlier one always keeps every line it carried — so a
// later pass can never quietly forget a fact an earlier one recorded. It is also
// duplicate-free, which is what stops the fold doubling the body every pass when
// a summarizer restates what the earlier ledger already said. The default role is
// the assistant's, which alternates with the user turn that anchors the head.
func BuildLedger(previous, summary string) nacelle.Message {
	return newLedgerMessage(MergeLedger(previous, summary))
}

// ExtraParts is a ledger message's parts past the sentinel text: what a
// same-role neighbour contributed when the assembly folded it in. Carrying them
// forward is what keeps that neighbour from being lost when the ledger is
// rebuilt from its text alone; they are appended after the fresh sentinel text.
func ExtraParts(m nacelle.Message) []nacelle.Part {
	if len(m.Parts) == 0 {
		return nil
	}
	text, ok := m.Parts[0].(nacelle.Text)
	if !ok {
		return nil
	}
	if _, ok := sentinelLine(text.Text); !ok {
		return nil
	}
	return m.Parts[1:]
}

// opposite is the role a message can sit next to without colliding. It belongs
// to the ledger's construction because that is the only reason this package
// needs it: the sentinel's role is chosen as the opposite of its neighbour, and
// BuildLedger's own default is the one that alternates with the user turn that
// anchors the head.
func opposite(role nacelle.Role) nacelle.Role {
	if role == nacelle.RoleUser {
		return nacelle.RoleAssistant
	}
	return nacelle.RoleUser
}
