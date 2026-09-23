package compaction

import (
	"strings"

	"github.com/FacileStudio/nacelle"
)

// Sentinel opens the one message that accumulates what a pass must not forget.
// It is the ledger's identity: IsLedger reads it, and a later pass folds into
// the message carrying it rather than ever summarizing it again.
const Sentinel = "[state ledger]"

// sentinelLine splits a text that opens with the sentinel from the body behind
// it, with ok false for a text that does not. The sentinel has to be the whole
// first line rather than a prefix of it. The identity it grants is the strongest
// one this package has — a ledger is what a pass folds into and never prunes —
// and a prefix is cheap to hit by accident: a turn quoting the marker, or a
// paste that happens to open with it, would claim the identity, and the ledger
// behind it would then read as history and become a prune candidate. The marker
// itself never changes, so nothing the package writes stops matching: every
// ledger it builds opens with the sentinel alone on its first line.
func sentinelLine(text string) (string, bool) {
	line, body, _ := strings.Cut(strings.TrimSpace(text), "\n")
	if strings.TrimSpace(line) != Sentinel {
		return "", false
	}
	return strings.TrimSpace(body), true
}

// ledgerBody is the first sentinel-bearing text part's body, with ok false for a
// message that carries no ledger. The role is not part of the identity:
// assembly picks the ledger's role as the opposite of its neighbour, so a
// conversation anchored on an assistant turn is folded into a user-role ledger
// and a later pass has to recognise that one too.
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
