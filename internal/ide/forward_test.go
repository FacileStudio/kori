package ide

import (
	"testing"
	"time"
)

// expectCommand waits for the session to have handled one editor command, so a
// test asserts on the session having read the line rather than hoping it has.
func expectCommand(t *testing.T, commands *recorder, want string) {
	t.Helper()
	select {
	case got := <-commands.events:
		if got != want {
			t.Errorf("the session handled %q, want %q", got, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("the session never handled %q", want)
	}
}

// TestSetCommandsWiresTheSessionAfterTheSocketIsOpen pins the path the session
// itself uses: the handler is the prompt loop, which does not exist yet when
// Start runs, so it is installed later and has to reach the socket that is
// already accepting editors.
func TestSetCommandsWiresTheSessionAfterTheSocketIsOpen(t *testing.T) {
	srv := startTestServer(t, Options{})
	sock, _ := attach(t, srv)
	commands := newRecorder()
	srv.SetCommands(commands)

	write(t, sock, `{"v":1,"t":"send","text":"after the fact","path":"a.go","line":2,"branch":"dev"}`)
	expectCommand(t, commands, "send after the fact a.go 2 dev")
}

// TestSetCommandsNilLeavesTheSocketAnObserver pins that a session can take its
// handler back: with none installed the commands are dropped, not queued, and
// the editor stays attached.
func TestSetCommandsNilLeavesTheSocketAnObserver(t *testing.T) {
	commands := newRecorder()
	srv := startTestServer(t, Options{Commands: commands})
	sock, dec := attach(t, srv)
	srv.SetCommands(nil)

	write(t, sock, `{"v":1,"t":"stop"}`)
	srv.Publish(Turn(1))

	wantString(t, readEvent(t, dec), "t", "turn")
	select {
	case got := <-commands.events:
		t.Errorf("a command reached a handler that was taken back: %q", got)
	default:
	}
}
