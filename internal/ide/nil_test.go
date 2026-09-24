package ide

import (
	"context"
	"testing"

	"github.com/FacileStudio/nacelle"
)

// TestANilServerIsSilence pins nil-means-off at every entry point the session
// uses, because the no-editor path is the one nobody exercises on purpose: a
// nil server publishes, refuses, installs inert hooks and closes, and none of
// it panics.
func TestANilServerIsSilence(t *testing.T) {
	var srv *Server
	srv.Publish(Turn(1))
	srv.SetCommands(sink{})
	if got := srv.Approve(context.Background(), "call-1", "run_command", `{"command":"ls"}`); got != Unanswered {
		t.Errorf("a nil server answered %v, want Unanswered", got)
	}
	if err := srv.Close(); err != nil {
		t.Errorf("closing a nil server: %v", err)
	}
	if err := srv.setSession("/tmp/x.jsonl"); err != nil {
		t.Errorf("recording a session on a nil server: %v", err)
	}

	for _, hooks := range srv.Hooks() {
		for _, hook := range hooks {
			hook(context.Background(), nacelle.HookEvent{Tool: "write_file", Input: `{"path":"a.go"}`})
		}
	}
}
