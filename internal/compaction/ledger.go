package compaction

import (
	"strings"

	"github.com/FacileStudio/nacelle"
)

// Sentinel opens the one message that accumulates what a pass must not forget.
// It is the ledger's identity: IsLedger reads it, and a later pass folds into
// the message carrying it rather than ever summarizing it again.
const Sentinel = "[state ledger]"

// IsLedger reports whether a message carries the state-ledger sentinel.
func IsLedger(m nacelle.Message) bool {
	for _, part := range m.Parts {
		if text, ok := part.(nacelle.Text); ok && strings.HasPrefix(strings.TrimSpace(text.Text), Sentinel) {
			return true
		}
	}
	return false
}

// Body is a ledger message's content with the sentinel stripped, empty for a
// message that carries none.
func Body(m nacelle.Message) string {
	for _, part := range m.Parts {
		text, ok := part.(nacelle.Text)
		if !ok || !strings.HasPrefix(strings.TrimSpace(text.Text), Sentinel) {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text.Text), Sentinel))
	}
	return ""
}

// BuildLedger returns the single message that carries the ledger: the sentinel,
// then the previous ledger folded into the new summary. The fold is monotone — a
// ledger built from an earlier one always contains it — so a later pass can
// never quietly forget a fact an earlier one recorded. The default role is the
// assistant's, which alternates with the user turn that anchors the head.
func BuildLedger(previous, summary string) nacelle.Message {
	body := fold(strings.TrimSpace(previous), strings.TrimSpace(summary))
	text := Sentinel
	if body != "" {
		text += "\n\n" + body
	}
	return nacelle.Message{
		Role:  nacelle.RoleAssistant,
		Parts: []nacelle.Part{nacelle.Text{Text: text}},
	}
}

// ExtraParts is a ledger message's parts past the sentinel text: what a
// same-role neighbour contributed when the assembly folded it in. Carrying them
// forward is what keeps that neighbour from being lost when the ledger is
// rebuilt from its text alone; they are appended after the fresh sentinel text.
func ExtraParts(m nacelle.Message) []nacelle.Part {
	if len(m.Parts) == 0 {
		return nil
	}
	if text, ok := m.Parts[0].(nacelle.Text); !ok || !strings.HasPrefix(strings.TrimSpace(text.Text), Sentinel) {
		return nil
	}
	return m.Parts[1:]
}

// fold merges a previous ledger into a new one without duplicating either: an
// empty side yields the other, an already-contained side is not repeated, and
// two genuinely new halves are concatenated.
func fold(previous, summary string) string {
	switch {
	case previous == "":
		return summary
	case summary == "" || strings.Contains(summary, previous):
		return summary
	case strings.Contains(previous, summary):
		return previous
	default:
		return previous + "\n\n" + summary
	}
}
