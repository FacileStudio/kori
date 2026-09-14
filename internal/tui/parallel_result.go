package tui

import (
	"encoding/json"
	"strings"

	"github.com/FacileStudio/nacelle"
)

// This file is the shared chromosome of a parallel fan-out: how the model's own
// parallel_agents call becomes row state, how streamed results get back onto
// the update loop, and the row helpers both the /parallel command and the model
// path read. The launcher and per-result handlers live in parallel_detached.go.

// rememberParallelCall stashes the tasks a model's parallel_agents call asked
// for, keyed by the tool call's ID, so the non-blocking stub result that follows
// can seed the batch's rows. The fan-out itself runs detached inside nacelle;
// the turn only needs to know the tasks to draw them.
func (m *Model) rememberParallelCall(tool nacelle.ToolEvent) {
	if m.pending == nil {
		m.pending = make(map[string][]string)
	}
	if m.pendingTitles == nil {
		m.pendingTitles = make(map[string][]string)
	}
	tasks, titles := ParseParallelTasks(tool.Input)
	if len(tasks) == 0 {
		return
	}
	m.pending[tool.ID] = tasks
	m.pendingTitles[tool.ID] = titles
}

// startDetachedParent consumes the stub a non-blocking parallel_agents call
// returned — `{"started":N,"batch":key}` — and registers that batch's rows. The
// results stream in tagged with the same batch, so they land exactly like the
// /parallel command's do.
func (m *Model) startDetachedParent(tool nacelle.ToolEvent, rawResult string) {
	tasks, ok := m.pending[tool.ID]
	if !ok {
		return
	}
	delete(m.pending, tool.ID)
	titles := m.pendingTitles[tool.ID]
	delete(m.pendingTitles, tool.ID)
	dec := json.NewDecoder(strings.NewReader(rawResult))
	var stub struct {
		Batch string `json:"batch"`
	}
	if err := dec.Decode(&stub); err != nil || stub.Batch == "" {
		return
	}
	m.registerParallel(stub.Batch, tasks, titles)
}

// PostDetached forwards a streamed parallel task outcome from nacelle's Detach
// callback — a goroutine it owns — into the detached channel. It is mounted as
// the parallel tool's Results hook, so the model's own calls surface exactly
// like a /parallel fan-out's do.
func PostDetached(r nacelle.ParallelTaskResult) {
	detached <- detachedResult{batch: r.Batch, idx: r.Index, result: r.Result, err: r.Err, usage: r.Usage}
}

// dropFinishedParallel forgets every parallel call whose tasks have all ended,
// so a completed fan-out's rows leave at the next send or run end rather than
// sitting under the prompt forever. Run from stranded(), which both send and
// settle reach.
func (m *Model) dropFinishedParallel() {
	for batch, tasks := range m.parallelTasks {
		done := true
		for _, pt := range tasks {
			if pt.Active {
				done = false
				break
			}
		}
		if done {
			delete(m.parallelTasks, batch)
		}
	}
}

// clearFinishedParallel is /clear's own cut at the parallel rows: every finished
// task is hidden, while still-running ones are left untouched. The finished
// tasks stay in place — Cleared, not deleted — so the running siblings keep the
// slice indices the live-update and result paths address; a batch left with no
// running task is dropped outright.
func (m *Model) clearFinishedParallel() {
	for batch, tasks := range m.parallelTasks {
		live := 0
		for i := range tasks {
			if tasks[i].Active {
				live++
				continue
			}
			tasks[i].Cleared = true
		}
		if live == 0 {
			delete(m.parallelTasks, batch)
		}
	}
}

// taskTitle is the title a running task row shows: the short description the
// summarizer generated when one has landed, else the task's prompt collapsed
// onto one line. Either way the row reads as an action, not a pasted block —
// the fallback is shortTitle-capped, so a fan-out whose summarizer has not
// landed yet never reads as the full prompt.
func taskTitle(pt parallelTaskInfo) string {
	if pt.Title != "" {
		return pt.Title
	}
	return shortTitle(strings.Join(strings.Fields(pt.Task), " "))
}

// parallelTaskRows returns the visible row count for a map of parallel
// fan-outs, matching the view — cleared tasks are not drawn and do not reserve
// a row.
func parallelTaskRows(tasks map[string][]parallelTaskInfo) int {
	rows := 0
	for _, call := range tasks {
		for _, pt := range call {
			if !pt.Cleared {
				rows++
			}
		}
	}
	return rows
}
