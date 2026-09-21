package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/FacileStudio/nacelle"
)

const abandoned nacelle.Stop = "abandoned"

func (m *Model) status() string {
	isReady := true
	state := "✓ ready"
	if cut := cutShort(m.run.stop); cut != "" {
		state = cut
		isReady = false
	}
	if m.run.busy {
		state = m.working()
		isReady = false
		if time.Since(m.run.interrupted) < forceQuit {
			state = "stopping · ctrl+c or ctrl+\\ to quit now"
		}
	}
	if m.run.pending != nil {
		state = fmt.Sprintf("approve %s(%s)? y = once · a = always this session · n = deny",
			m.run.pending.Name, truncate(unstyled(string(m.run.pending.Input)), 60))
	}

	if m.session != nil && m.session.HasWriteError() {
		state = "! could not write to session log · " + state
		isReady = false
	}

	width := max(m.width, 1)
	counts := strings.Join(m.footer(), " ")
	stateLine := truncate(state, width)
	if isReady {
		stateLine = m.theme.Ready.Render(stateLine)
	}
	return stateLine + "\n" + m.theme.Muted.Render(truncate(counts, width))
}

// footer is the costs line under the state line: price when the backend
// reported one, input and output tokens, and the context size. The output and
// context counts carry the live estimate (liveOut) so they tick as the model
// writes, and the price carries that estimate scaled by the last realised cost
// per token (rate) so the dollar figure moves too; the real per-turn usage
// replaces the estimates the moment a turn ends. A session with a grind
// budget adds what is left of the current run's minimum, and a session
// without one looks exactly as it did before.
func (m *Model) footer() []string {
	total := m.spent.Add(m.run.usage)
	live := m.rate * float64(m.run.liveOut)

	var spent []string
	if total.Cost+live > 0 {
		spent = append(spent, fmt.Sprintf("$%.4f", total.Cost+live))
	}
	spent = append(spent,
		"↑"+shortTokens(total.InputTokens+total.CacheCreationTokens),
		"↓"+shortTokens(total.OutputTokens+m.run.liveOut))
	if m.size > 0 {
		spent = append(spent, contextLoad(m.size+m.run.liveOut, m.policy))
	}
	liveSpend := m.run.usage
	liveSpend.OutputTokens += m.run.liveOut
	liveSpend.Cost += m.rate * float64(m.run.liveOut)
	if left := m.grind.left(liveSpend); left != "" {
		spent = append(spent, "grind "+left)
	}
	return spent
}

func (m *Model) working() string {
	if m.compacting {
		return m.theme.Compacting.Render(m.spin.View() + " compacting session")
	}
	doing := waitingVerb(time.Since(m.run.began))
	switch n := m.running(); n {
	case 0:
	case 1:
		name, ok := m.runningName()
		if ok {
			doing = "running " + name
		}
	default:
		doing = fmt.Sprintf("running %d tools", n)
	}
	if since := m.ongoing(); since != "" {
		doing += " · " + since
	}
	return yellow.Render(m.spin.View() + " " + doing)
}

func (m *Model) running() int {
	n := 0
	for _, g := range m.run.groups {
		if g.End.IsZero() {
			n++
		}
	}
	return n
}

func (m *Model) runningName() (string, bool) {
	for _, g := range m.run.groups {
		if g.End.IsZero() {
			return g.Tool.Name, true
		}
	}
	return "", false
}

func (m *Model) ongoing() string {
	if !m.run.busy || m.run.began.IsZero() {
		return ""
	}
	return lasted(time.Since(m.run.began))
}

// spun advances the spinner. A compaction pass keeps it (and the running-row
// timer that redraws with it) ticking even though run.busy is false on the
// idle and /compact paths — without it that row freezes at the frame and
// elapsed time it had when the pass started.
func (m *Model) spun(message spinner.TickMsg) tea.Cmd {
	return m.spin.Spun(message, m.run.busy || m.hasLiveParallel() || m.compacting)
}

func cutShort(stop nacelle.Stop) string {
	if stop == "" || stop.Complete() {
		return ""
	}
	switch stop {
	case nacelle.StopMaxTokens:
		return "cut off at the token limit"
	case nacelle.StopContext:
		return "cut off: out of context"
	case nacelle.StopRefusal:
		return "refused by the model"
	case nacelle.StopIterations:
		return "stopped at the iteration limit"
	case abandoned:
		return "abandoned"
	}
	return "stopped early"
}
