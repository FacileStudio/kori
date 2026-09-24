package ide

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/FacileStudio/nacelle"
)

// fire runs one hook the way a session does.
func fire(hook nacelle.Hook, point nacelle.HookPoint, tool, input string, err error) {
	hook(context.Background(), nacelle.HookEvent{Point: point, Tool: tool, Input: input, Err: err})
}

func TestHooksPublishAToolCallAndTheFileItChanged(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "a.txt", "one\ntwo\nthree\n")
	srv := startTestServer(t, Options{Root: root})
	_, dec := attach(t, srv)
	hooks := srv.Hooks()
	input := `{"path":"a.txt","content":"one\nTWO\nthree\n"}`

	fire(hooks[nacelle.BeforeToolCall][0], nacelle.BeforeToolCall, "write_file", input, nil)
	start := readEvent(t, dec)
	wantString(t, start, "t", "tool")
	wantString(t, start, "status", "start")
	wantString(t, start, "path", "a.txt")

	fire(hooks[nacelle.AfterToolCall][0], nacelle.AfterToolCall, "write_file", input, nil)
	done := readEvent(t, dec)
	wantString(t, done, "status", "done")
	wantString(t, done, "name", "write_file")

	edit := readEvent(t, dec)
	wantString(t, edit, "t", "edit")
	wantString(t, edit, "path", "a.txt")
	wantNumber(t, edit, "first", 2)
	wantNumber(t, edit, "last", 2)
	wantNumber(t, edit, "added", 1)
	wantNumber(t, edit, "removed", 1)

	for _, ev := range []map[string]any{start, done, edit} {
		if ev["id"] != start["id"] {
			t.Errorf("the call's events carry different ids: %v", ev)
		}
	}
}

// writeFixture puts a file in a temporary root the way a session would find
// it before a call rewrote it.
func writeFixture(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}
}

func TestHooksPublishNoEditForAToolThatChangesNoFile(t *testing.T) {
	srv := startTestServer(t, Options{Root: t.TempDir()})
	_, dec := attach(t, srv)
	hooks := srv.Hooks()

	fire(hooks[nacelle.BeforeToolCall][0], nacelle.BeforeToolCall, "read_file", `{"path":"a.txt"}`, nil)
	start := readEvent(t, dec)
	wantString(t, start, "name", "read_file")
	if path, ok := start["path"]; ok && path != "a.txt" {
		t.Errorf("path = %v, want the one the call named", path)
	}

	fire(hooks[nacelle.AfterToolCall][0], nacelle.AfterToolCall, "read_file", `{"path":"a.txt"}`, nil)
	done := readEvent(t, dec)
	wantString(t, done, "status", "done")
	if done["ok"] != true {
		t.Errorf("a call that did not fail was reported as failed: %v", done)
	}
}

func TestHooksPublishNoEditForACallThatFailed(t *testing.T) {
	srv := startTestServer(t, Options{Root: t.TempDir()})
	sock, dec := attach(t, srv)
	hooks := srv.Hooks()
	input := `{"path":"a.txt","content":"new\n"}`

	fire(hooks[nacelle.BeforeToolCall][0], nacelle.BeforeToolCall, "write_file", input, nil)
	readEvent(t, dec)
	fire(hooks[nacelle.AfterToolCall][0], nacelle.AfterToolCall, "write_file", input, os.ErrPermission)

	done := readEvent(t, dec)
	if done["ok"] != false {
		t.Errorf("a failed call was reported as ok: %v", done)
	}
	write(t, sock, `{"v":1,"t":"stop"}`)
	srv.Publish(Turn(1))
	wantString(t, readEvent(t, dec), "t", "turn")
}
