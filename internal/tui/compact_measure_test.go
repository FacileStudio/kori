package tui

// What the next send would carry, split from the guard that acts on it: the
// measure has its own regressions — an uncountable backend, a zero that means
// "unknown" — and they are worth reading without the pass machinery around them.

import (
	"context"
	"testing"
)

// The live count wins when the backend offers one: it sees the conversation as
// it will be sent, not as it last was.
func TestSendSizePrefersTheBackendCount(t *testing.T) {
	m := sized()
	m.agent = agentOver(t, counting{tokens: 123_456})
	m.size = 90_000

	size, ok := m.sendSize(context.Background())

	if !ok || size != 123_456 {
		t.Errorf("sendSize = %d, %v, want the backend's own count", size, ok)
	}
}

// The regression the pre-send guard exists for. A backend that cannot count must
// fall back to the last size the provider reported, not stand the guard down.
func TestSendSizeFallsBackToTheLastReportedSize(t *testing.T) {
	m := sized()
	m.agent = agentOver(t, blind{})
	m.size = 90_000

	size, ok := m.sendSize(context.Background())

	if !ok || size != 90_000 {
		t.Errorf("sendSize = %d, %v, want the last usage-reported size", size, ok)
	}
}

// Zero from a backend that claims to count is not a measure — it is the shape a
// stub backend answers with, and reading it as one would make the guard quieter
// than the unknown it is.
func TestSendSizeIgnoresAZeroCount(t *testing.T) {
	m := sized()
	m.agent = agentOver(t, counting{})
	m.size = 90_000

	size, ok := m.sendSize(context.Background())

	if !ok || size != 90_000 {
		t.Errorf("sendSize = %d, %v, want the fallback rather than a zero count", size, ok)
	}
}

// With neither a count nor a remembered size there is nothing to measure
// against, and compacting on a guess would be worse than not compacting.
func TestSendSizeReportsNoMeasureWhenThereIsNone(t *testing.T) {
	m := sized()
	m.agent = agentOver(t, blind{})

	if size, ok := m.sendSize(context.Background()); ok {
		t.Errorf("sendSize = %d, true, want no measure from an empty session", size)
	}
}
