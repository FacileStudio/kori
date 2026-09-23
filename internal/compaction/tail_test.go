package compaction

import (
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

// The budget does the sizing, and the message floor is only a floor: with the
// shipped floor of one message, a tail of small turns is kept whole, where a
// count of messages would have kept exactly one.
func TestActiveWindowIsSizedByTheBudget(t *testing.T) {
	conv := []nacelle.Message{nacelle.UserText("the task")}
	for range 6 {
		conv = append(conv, nacelle.AssistantText(strings.Repeat("x", 400)))
	}
	policy := Policy{AnchorMessages: 1, KeepTurns: 1, KeepTokens: 1_000}

	if got := activeStart(conv, len(conv), 1, policy); got != 1 {
		t.Errorf("activeStart = %d, want the whole budget's worth of tail kept", got)
	}
}

// A tail heavier than the whole budget still keeps the newest message. It is the
// live turn the run is answering and no summary can stand in for it, so the walk
// stops there rather than trimming into the turn.
func TestActiveWindowKeepsTheNewestMessageWhateverItWeighs(t *testing.T) {
	conv := heavyTail()
	policy := Policy{AnchorMessages: 1, KeepTurns: 1, KeepTokens: 40_000}

	if got := activeStart(conv, 4, 1, policy); got != 3 {
		t.Errorf("activeStart = %d, want the newest message alone", got)
	}
}

// A floor somebody set is honoured even when the turns under it are heavy: the
// floor is the one bound a pass cannot move, because the active window is never
// touched. That is exactly why the shipped floor is one and the budget is what
// sizes the tail.
func TestActiveWindowHonoursAGenerousMessageFloor(t *testing.T) {
	conv := heavyTail()
	policy := Policy{AnchorMessages: 1, KeepTurns: 3, KeepTokens: 40_000}

	if got := activeStart(conv, 4, 1, policy); got != 1 {
		t.Errorf("activeStart = %d, want the floor of three messages kept", got)
	}
}

// Without a budget the floor is the whole of it, which is how a session that
// never mentions keep_tokens keeps behaving exactly as it did.
func TestActiveWindowFallsBackToTheMessageFloor(t *testing.T) {
	conv := []nacelle.Message{
		nacelle.UserText("the task"),
		nacelle.AssistantText("working"),
		nacelle.UserText("newest"),
	}
	policy := Policy{AnchorMessages: 1, KeepTurns: 1}

	if got := activeStart(conv, 3, 1, policy); got != 2 {
		t.Errorf("activeStart = %d, want the newest message only", got)
	}
}

// The budget can never take more than half of what a pass may fill: a tail that
// size leaves nothing for the ladder to fold, so the pass it triggers could not
// land under its own trigger whatever it summarized. Both bounds that cap it are
// covered — the usable window when one is known, and the ceiling, which is the
// only one a backend reporting no window has. Without the second, a flat 40k tail
// against a 75k ceiling would keep most of the context verbatim and no pass could
// ever land under it.
func TestKeepBudgetNeverTakesHalfOfWhatAPassMayFill(t *testing.T) {
	tests := []struct {
		name   string
		policy Policy
		want   int
	}{
		{"half the usable window", Policy{Window: 200_000, Reserve: 40_000, KeepTokens: 100_000}, 80_000 * bytesPerToken},
		{"a budget under the cap left alone", Policy{Window: 200_000, Reserve: 40_000, KeepTokens: 10_000}, 10_000 * bytesPerToken},
		{"half the ceiling when no window is reported", Policy{Ceiling: 75_000, KeepTokens: 40_000}, 37_500 * bytesPerToken},
		{"the tighter of the two bounds wins", Policy{Window: 200_000, Reserve: 40_000, Ceiling: 50_000, KeepTokens: 100_000}, 25_000 * bytesPerToken},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.policy.keepBudget(); got != tc.want {
				t.Errorf("keepBudget = %d, want %d", got, tc.want)
			}
		})
	}
}

// heavyTail is an anchor followed by three turns too large to share a budget.
func heavyTail() []nacelle.Message {
	heavy := strings.Repeat("x", 200_000)
	return []nacelle.Message{
		nacelle.UserText("the task"),
		nacelle.AssistantText(heavy),
		nacelle.UserText(heavy),
		nacelle.AssistantText(heavy),
	}
}
