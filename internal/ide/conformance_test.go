package ide

import (
	"context"
	"os"
	"testing"
	"time"
)

// conformanceStep is one object a session publishes and what the wire has to
// carry for it: exactly these fields, with these values.
type conformanceStep struct {
	event Event
	keys  []string
	value map[string]any
}

// TestProtocolConformanceOverARealSocket drives the whole protocol against a
// fake editor on an actual unix socket, in the order a real session drives it,
// and then checks the bytes that crossed it.
func TestProtocolConformanceOverARealSocket(t *testing.T) {
	commands := newRecorder()
	srv := startTestServer(t, Options{
		Root: "/repo", Model: "claude", Version: "v0.76.0", Session: "/tmp/s.jsonl",
	})
	srv.SetCommands(commands)
	e := dial(t, srv)

	wantHello(t, e)
	driveSessionEvents(t, e, srv)
	driveEditorCommands(t, e, commands)
	driveApproval(t, e, srv)
	raw := driveErrorAndClose(t, e, srv)

	assertWireShape(t, raw, 9)
}

// wantHello checks the first line an editor reads, which is the one that says
// what it has attached to.
func wantHello(t *testing.T, e *editor) {
	t.Helper()
	wantEvent(t, e.object(),
		[]string{"v", "t", "pid", "root", "session", "model", "version"},
		map[string]any{
			"v": float64(Protocol), "t": "hello", "pid": float64(os.Getpid()),
			"root": "/repo", "session": "/tmp/s.jsonl", "model": "claude", "version": "v0.76.0",
		})
}

// sessionSteps is one of every event a session sends, with the field set and
// the values its type is specified to carry.
func sessionSteps() []conformanceStep {
	return []conformanceStep{
		{Turn(1), []string{"v", "t", "n"}, map[string]any{"t": "turn", "n": float64(1)}},
		{ToolStart("edit_file-1", "edit_file", "a.go"),
			[]string{"v", "t", "id", "name", "status", "path"},
			map[string]any{"id": "edit_file-1", "name": "edit_file", "status": "start", "path": "a.go"}},
		{ToolDone("edit_file-1", "edit_file", true, "a.go"),
			[]string{"v", "t", "id", "name", "status", "ok", "path"},
			map[string]any{"id": "edit_file-1", "status": "done", "ok": true}},
		{Edit("edit_file-1", Change{Path: "a.go", Tool: "edit_file", First: 2, Last: 4, Added: 3, Removed: 1}),
			[]string{"v", "t", "id", "path", "tool", "first", "last", "added", "removed"},
			map[string]any{"path": "a.go", "tool": "edit_file", "first": float64(2), "last": float64(4)}},
		{Edit("edit_file-1", Change{Path: "a.go", Tool: "edit_file", Diff: unifiedDiff}),
			[]string{"v", "t", "id", "path", "tool", "first", "last", "added", "removed", "diff"},
			map[string]any{"diff": unifiedDiff}},
		{Done(EndTurn, 0.25), []string{"v", "t", "reason", "cost"},
			map[string]any{"reason": EndTurn, "cost": 0.25}},
	}
}

// unifiedDiff stands in for the diff an edit can carry. Its newlines are the
// point: they have to survive the wire escaped, or the object would arrive as
// several lines and the framing would be gone.
const unifiedDiff = "@@ -1,3 +1,3 @@\n one\n-two\n+TWO\n three\n"

// driveSessionEvents publishes one of every event a session sends and checks
// each arrived as one object carrying exactly its own fields.
func driveSessionEvents(t *testing.T, e *editor, srv *Server) {
	t.Helper()
	for _, step := range sessionSteps() {
		srv.Publish(step.event)
		wantEvent(t, e.object(), step.keys, step.value)
	}
}

// driveEditorCommands sends one of every command an editor sends: the handshake
// and an unknown type are ignored, an unknown field inside a known type is
// ignored, and the three that carry work reach the handler the session
// installed.
func driveEditorCommands(t *testing.T, e *editor, commands *recorder) {
	t.Helper()
	e.send(`{"v":1,"t":"hello","root":"/repo","pid":1}`)
	e.send(`{"v":1,"t":"something-new","field":1}`)
	e.send(`{"v":1,"t":"send","text":"fix the test","path":"a.go","line":7,"branch":"main","from_the_future":42}`)
	e.send(`{"v":1,"t":"open","path":"b.go","line":3}`)
	e.send(`{"v":1,"t":"stop"}`)

	for _, want := range []string{"send fix the test a.go 7 main", "open b.go 3", "stop"} {
		expectCommand(t, commands, want)
	}
}

// driveApproval round trips one approval: the session asks, the editor allows,
// and the answer comes back with the tool's own JSON carried verbatim.
func driveApproval(t *testing.T, e *editor, srv *Server) {
	t.Helper()
	answers := make(chan Answer, 1)
	go func() { answers <- srv.Approve(context.Background(), "call-1", "run_command", `{"command":"ls"}`) }()

	wantEvent(t, e.object(),
		[]string{"v", "t", "id", "tool", "input"},
		map[string]any{"t": "approval", "id": "call-1", "tool": "run_command", "input": `{"command":"ls"}`})
	e.send(`{"v":1,"t":"approve","id":"call-1","allow":true}`)

	select {
	case got := <-answers:
		if got != Allowed {
			t.Errorf("Approve = %v, want the explicit allow taken as one", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the approval never came back")
	}
}

// driveErrorAndClose ends the session the way the protocol says an error does:
// the object goes out and the connection closes after it. It returns every byte
// the editor was sent.
func driveErrorAndClose(t *testing.T, e *editor, srv *Server) string {
	t.Helper()
	srv.Publish(Err("boom"))
	wantEvent(t, e.object(), []string{"v", "t", "reason"}, map[string]any{"reason": "boom"})
	if rest := e.rest(); rest != "" {
		t.Errorf("the session sent %q after an error, want the connection closed", rest)
	}

	if err := srv.Close(); err != nil {
		t.Errorf("closing the IDE server: %v", err)
	}
	return e.seen.String()
}
