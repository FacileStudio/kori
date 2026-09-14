package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/FacileStudio/nacelle"
)

// taskTitled carries one generated title back to the update loop. The short
// description LLM runs off the loop, so its outputs arrive here and are
// applied on the one thread that owns the task rows.
type taskTitled struct {
	Call  string
	Index int
	Title string
}

// taskTitles is the channel the title summarizer posts to, drained by the
// watchTitles command on the update loop.
var taskTitles = make(chan taskTitled, 64)

func watchTitles() tea.Cmd {
	return func() tea.Msg {
		return <-taskTitles
	}
}

func (m *Model) recordTitle(t taskTitled) tea.Cmd {
	if tasks, ok := m.parallelTasks[t.Call]; ok && t.Index < len(tasks) {
		if tasks[t.Index].Title == "" {
			tasks[t.Index].Title = t.Title
		}
	}
	return watchTitles()
}

// titleSystem is the one-shot summarizer that turns a subagent's task into a
// 4-7 word status-line title. It runs on the same backend as the session —
// billed like the work it names — and with no tools.
const titleSystem = "You turn a list of subagent tasks into short, cool, punchy status-line titles. Reply with exactly one title per task, one per line, in the same order as given. Each title is a 4-7 word plain description of the task (e.g. 'audit auth middleware', 'search postgres backup logs'). Action-oriented, lower case, no punctuation, no emoji, no numbering, no leading bullets. Reply with nothing but the titles."

const titleMaxTokens int64 = 200

// buildTitlesPrompt numbers the tasks so the model's returned lines align.
func buildTitlesPrompt(tasks []string) string {
	var lines []string
	for i, task := range tasks {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, task))
	}
	return strings.Join(lines, "\n")
}

// shortTitle strips path and URL tokens from a returned title, then hard-caps
// it at seven words, so a model that ran long — or pasted a path the task
// happened to name — cannot push a task row onto the next line or read as a
// file location. Used for both generated titles and the collapsed-task fallback.
func shortTitle(s string) string {
	s = strings.TrimSpace(s)
	fields := stripPathTokens(strings.Fields(s))
	if len(fields) > 7 {
		fields = fields[:7]
		for len(fields) > 2 {
			last := strings.ToLower(fields[len(fields)-1])
			if last == "in" || last == "for" || last == "to" || last == "with" || last == "on" || last == "the" || last == "a" || last == "an" || last == "and" || last == "of" || last == "at" {
				fields = fields[:len(fields)-1]
				continue
			}
			break
		}
	}
	return strings.Join(fields, " ")
}

// stripPathTokens drops tokens that look like a file path or URL — anything
// carrying a path separator or a scheme — so a task title stays a plain
// description of the work, never the location the work was done in.
func stripPathTokens(fields []string) []string {
	keep := fields[:0]
	for _, f := range fields {
		if strings.Contains(f, "://") || strings.Contains(f, "/") || strings.Contains(f, `\`) {
			continue
		}
		keep = append(keep, f)
	}
	return keep
}

// titleParallelTasks names a parallel call's tasks in the background and
// reports each title back on the shared channel as it arrives. It runs only
// when a session is live, so the summarizer is billed like the work it names.
func (m *Model) titleParallelTasks(call string, tasks []string) {
	if m.agent == nil {
		return
	}
	backend := m.agent.Backend()
	go func() {
		titles := titleTasks(tasks, backend)
		for i, title := range titles {
			if title != "" {
				taskTitles <- taskTitled{Call: call, Index: i, Title: title}
			}
		}
	}()
}

// titleTasks is the extra LLM call behind the task rows: one summarizer run
// takes the whole fan-out and returns one short title per task, in order.
// A backend that errors or returns no text leaves the rows on their collapsed
// full-prompt title, the same thing displayed before titles existed.
func titleTasks(tasks []string, backend nacelle.Backend) []string {
	out := make([]string, len(tasks))
	if backend == nil {
		return out
	}
	agent, err := nacelle.New(nacelle.Config{
		Backend:       backend,
		System:        titleSystem,
		Thinking:      nacelle.Thinking{},
		MaxTokens:     titleMaxTokens,
		MaxIterations: 1,
	})
	if err != nil {
		return out
	}
	var b strings.Builder
	for event, err := range agent.Stream(context.Background(), []nacelle.Message{
		{Role: nacelle.RoleUser, Parts: []nacelle.Part{nacelle.Text{Text: buildTitlesPrompt(tasks)}}},
	}) {
		if err != nil {
			return out
		}
		if event.Kind == nacelle.KindText {
			b.WriteString(event.Text)
		}
	}
	lines := strings.Split(b.String(), "\n")
	for i := range out {
		if i < len(lines) {
			out[i] = shortTitle(lines[i])
		}
	}
	return out
}
