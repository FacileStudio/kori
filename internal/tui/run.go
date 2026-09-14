package tui

import (
	"context"
	"errors"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/FacileStudio/kori/internal/herdr"
	"github.com/FacileStudio/kori/internal/skills"
	"github.com/FacileStudio/nacelle"
)

var delegations = make(chan nacelle.Usage, 64)

// DelegateUsage delivers a delegated run's spend to the update loop.
func DelegateUsage(u nacelle.Usage) {
	delegations <- u
}

// LaunchContext carries the pre-turn prompt accounting the launch notes
// report: which context files were loaded and the rough token cost of the
// context and the whole system prompt.
type LaunchContext struct {
	ContextPaths  []string
	ContextTokens int64
	SystemTokens  int64
	// Diagnostics is the tools.diagnostics toggle, carried here because the
	// launch sweep it arms is one of the launch notes.
	Diagnostics bool
}

// SessionConfig configures the runtime settings for an interactive session.
type SessionConfig struct {
	Root              string
	Model             string
	Backend           string
	Diffs             bool
	GroupTools        *bool
	ShowThinking      bool
	CompactAt         int64
	AutoResume        bool
	Resume            string
	PromptPlaceholder string
	StartMessage      string
	Startup           LaunchContext
	// Grind is the per-run minimum spend: dollars, output tokens, and how
	// many continuation notices one run may be given. Both floors at zero
	// leave the budget off.
	GrindCost          float64
	GrindTokens        int64
	GrindContinuations int
}

// UISession holds the complete state needed to run an interactive terminal session.
type UISession struct {
	Agent             *nacelle.Agent
	Banner            string
	Skills            []skills.Skill
	HookNotice        string
	Gate              *Approvals
	DelegateConfig    nacelle.Config
	Mode              string
	TransparentBlocks bool
	SessionConfig
}

// Launch starts the Bubble Tea UI session loop for the given configuration.
func Launch(c UISession) error {
	opened := NewModel(c.Agent, c.Banner, c.Skills, c.SessionConfig)
	boot(opened, c)
	if c.HookNotice != "" {
		opened.say(fromClient, c.HookNotice)
	}
	startupPrint(opened)

	program := tea.NewProgram(opened)
	if c.Gate != nil {
		c.Gate.Wire(program.Send)
	}
	final, err := program.Run()

	if done, ok := final.(*Model); ok {
		herdr.Release(done.herdrClient)
		if recap := done.recap(); recap != "" {
			fmt.Println(recap)
		}
	}
	return err
}

func (m *Model) send(text string) tea.Cmd {
	m.run.stop = ""
	m.run.usage = nacelle.Usage{}
	m.run.liveOut = 0
	m.run.began = time.Now()
	m.run.turnBegan = time.Now()
	m.run.interrupted = time.Time{}
	m.run.asked, m.run.answered = nil, nil
	m.run.reported = false
	m.stranded()
	m.conversation = append(m.conversation, nacelle.UserText(text))

	ctx, cancel := context.WithCancel(context.Background())
	m.run.cancel = cancel
	m.run.bgCtx = ctx
	m.run.busy = true

	herdr.Report(m.herdrClient, herdr.Working)

	if count, err := m.agent.CountTokens(ctx, m.conversation); err == nil && m.compactAt > 0 && !m.thrashed() && count > m.compactAt+compactSlack {
		m.size = count
		if waiting := m.compactBeforeSend(ctx); waiting != nil {
			return waiting
		}
	}

	return m.startRun(ctx)
}

// startRun fires the model at the current conversation and returns the
// spinner-ticked wait a run is driven by. It is the one seam both send and
// settleCompaction reach: send starts a run directly, and a send that had to
// compact first hands control to the compaction's outcome, which calls this
// once the context is freed.
func (m *Model) startRun(ctx context.Context) tea.Cmd {
	m.run.results = start(ctx, m.agent, m.conversation)
	return tea.Batch(waitFor(m.run.results), m.spin.Tick)
}

func (m *Model) abandon() {
	m.run.interrupted = time.Now()
	m.run.stop = abandoned
	m.run.pending = nil
	m.run.cancel()
	if m.Len() > 0 {
		m.say(fromClient, fmt.Sprintf("%s dropped, not sent", countedNoun(m.Drop(), "queued message")))
		m.layout(m.windowHeight)
	}
}

func (m *Model) escaped() (bool, tea.Cmd) {
	if !m.run.busy {
		return false, nil
	}
	if time.Since(m.run.interrupted) >= forceQuit {
		m.run.interrupted = time.Now()
		m.run.stop = abandoned
		m.run.pending = nil
		m.run.cancel()
		if waiting := m.Len(); waiting > 0 {
			m.say(fromClient, "stopped · "+countedNoun(waiting, "queued message")+" still to send")
		}
	}
	return true, nil
}

// consume takes the next thing a run produced. A streamed event is recorded and
// absorbed; an error is committed as a failure — unless it is a cancellation,
// which is not a failure at all. The reader stopping the run, or a parallel
// fan-out relaxing the parent to free the prompt, both end the context; the
// stream then races a trailing context.Canceled past ctx.Done, and painting
// that red would flash an alarm over an otherwise clean stop. Cancels are
// swallowed so the run just ends; a genuine backend or tool error still fails
// loudly.
func (m *Model) consume(next result) tea.Cmd {
	if next.err != nil {
		if !errors.Is(next.err, context.Canceled) {
			m.flush()
			m.run.reported = true
			m.say(fromFailure, next.err.Error())
		}
		return waitFor(m.run.results)
	}
	m.record(next.event)
	m.absorb(next.event)
	return waitFor(m.run.results)
}

func (m *Model) recap() string {
	total := m.spent.Add(m.run.usage)
	tokens := total.InputTokens + total.OutputTokens + total.CacheReadTokens + total.CacheCreationTokens
	if m.tools == 0 && tokens == 0 {
		return ""
	}

	shape := "session · " + lasted(time.Since(m.began))
	switch {
	case m.tools == 1:
		shape += " · 1 tool"
	case m.tools > 1:
		shape += fmt.Sprintf(" · %d tools", m.tools)
	}
	if m.failed > 0 {
		shape += fmt.Sprintf(" · %d failed", m.failed)
	}

	spend := fmt.Sprintf("↑%s ↓%s",
		shortTokens(total.InputTokens+total.CacheCreationTokens),
		shortTokens(total.OutputTokens))
	if total.CacheReadTokens > 0 {
		spend += fmt.Sprintf(" · %s cached", shortTokens(total.CacheReadTokens))
	}
	if total.Cost > 0 {
		spend += fmt.Sprintf(" · $%.4f", total.Cost)
	}
	return shape + "\n" + spend
}
