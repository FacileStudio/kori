package tui

import (
	"strings"
	"testing"

	"github.com/FacileStudio/nacelle"
)

// The model's own parallel_agents call is non-blocking. The call only stashes
// the tasks it asked for; the stub result — "started", with a batch key — is
// what seeds the rows and announces the fan-out, so the parent's turn keeps
// going instead of waiting on the whole run.
func TestModelParallelCallStubSeedsRows(t *testing.T) {
	m := sized()
	tool := nacelle.ToolEvent{ID: "call1", Name: nacelle.ParallelAgentsToolName, Input: `{"tasks":["a","b","c"]}`}

	m.absorbToolCall(tool)
	if _, ok := m.pending["call1"]; !ok {
		t.Fatalf("pending = %v, want the call's tasks stashed", m.pending)
	}
	if len(m.parallelTasks) != 0 {
		t.Fatalf("rows created before the stub arrived: %v", m.parallelTasks)
	}

	m.absorbToolResult(tool, `{"started":3,"batch":"psa-1"}`)

	tasks, ok := m.parallelTasks["psa-1"]
	if !ok {
		t.Fatalf("no rows under the stub's batch key, have %v", m.parallelTasks)
	}
	if len(tasks) != 3 {
		t.Errorf("batch rows = %d, want 3", len(tasks))
	}
	for _, pt := range tasks {
		if !pt.Active {
			t.Errorf("task %q not active after seed", pt.Task)
		}
	}
	if _, ok := m.pending["call1"]; ok {
		t.Error("pending not cleared after the stub")
	}
	joined := strings.Join(spoken(m), "\n")
	announce := strings.Index(joined, "✓ 3 parallel agents started")
	call := strings.Index(joined, "parallel_agents")
	if announce < 0 || call < 0 || call > announce {
		t.Errorf("transcript = %q, want the tool call line above an announcement with the count", joined)
	}
}

// The stub carries the return-control rule appended to the JSON, so the result
// is prose after the object. Reading the first JSON value, not the whole
// string, is what keeps the rows seeding.
func TestModelParallelCallStubWithProseSeedsRows(t *testing.T) {
	m := sized()
	tool := nacelle.ToolEvent{ID: "call2", Name: nacelle.ParallelAgentsToolName, Input: `{"tasks":["a"]}`}

	m.absorbToolCall(tool)
	stub := `{"started":1,"batch":"psa-2"}` +
		"\nFan-out dispatched. End your turn now: the parallel agents run detached."
	m.absorbToolResult(tool, stub)

	if tasks, ok := m.parallelTasks["psa-2"]; !ok || len(tasks) != 1 {
		t.Fatalf("parallelTasks = %v, want one seeded row under psa-2", m.parallelTasks)
	}
}

// The streamed results that follow the stub route by the batch key and fold the
// task's spend into the session total, exactly like a /parallel fan-out's do.
func TestModelParallelStubResultsRouteToBatch(t *testing.T) {
	m := sized()
	tool := nacelle.ToolEvent{ID: "call1", Name: nacelle.ParallelAgentsToolName, Input: `{"tasks":["a","b"]}`}
	m.absorbTool(tool)
	m.absorbToolResult(tool, `{"started":2,"batch":"psa-2"}`)
	before := m.spent.OutputTokens

	m.recordDetached(detachedResult{batch: "psa-2", idx: 0, result: "r0", usage: nacelle.Usage{OutputTokens: 50}})

	pt := m.parallelTasks["psa-2"][0]
	if pt.Result != "r0" || pt.Active {
		t.Errorf("task 0 not updated: result=%q active=%v", pt.Result, pt.Active)
	}
	if m.parallelTasks["psa-2"][1].Active == false {
		t.Error("task 1 was marked done alongside its sibling")
	}
	if m.spent.OutputTokens != before+50 {
		t.Errorf("session spend not credited with the task's usage")
	}
}

// absorbTool is absorbToolCall then absorbToolResult, the order a stream
// delivers them.
func (m *Model) absorbTool(tool nacelle.ToolEvent) {
	m.absorbToolCall(tool)
	m.absorbToolResult(tool, `{"started":2,"batch":"psa-2"}`)
}

// A model path that never shows its stub — an error or a lost call — leaves the
// pending stash alone, so nothing phantoms into the transcript.
func TestModelParallelUnknownResultIsIgnored(t *testing.T) {
	m := sized()
	m.recordDetached(detachedResult{batch: "nope", idx: 0, result: "r"})
	if len(m.parallelTasks) != 0 {
		t.Errorf("unknown batch created state: %v", m.parallelTasks)
	}
}

func TestModelParallelCallWithTitlesSeedsRows(t *testing.T) {
	m := sized()
	input := `{"tasks":[{"title":"audit auth middleware","task":"check tokens"},{"title":"run tests","task":"go test ./..."}]}`
	tool := nacelle.ToolEvent{ID: "call3", Name: nacelle.ParallelAgentsToolName, Input: input}

	m.absorbToolCall(tool)
	m.absorbToolResult(tool, `{"started":2,"batch":"psa-3"}`)

	tasks, ok := m.parallelTasks["psa-3"]
	if !ok || len(tasks) != 2 {
		t.Fatalf("tasks not seeded: %v", tasks)
	}
	if tasks[0].Title != "audit auth middleware" {
		t.Errorf("task 0 title = %q, want %q", tasks[0].Title, "audit auth middleware")
	}
	if tasks[1].Title != "run tests" {
		t.Errorf("task 1 title = %q, want %q", tasks[1].Title, "run tests")
	}
	if taskTitle(tasks[0]) != "audit auth middleware" {
		t.Errorf("taskTitle(0) = %q, want %q", taskTitle(tasks[0]), "audit auth middleware")
	}
}
