package ide

import (
	"fmt"
	"testing"
	"time"
)

// recorder is a Commands that reports what reached the session, so a test can
// wait for the session to have handled a line rather than hope it has.
type recorder struct {
	events chan string
}

func newRecorder() *recorder {
	return &recorder{events: make(chan string, 8)}
}

func (r *recorder) Send(text, path string, line int, branch string) {
	r.events <- fmt.Sprintf("send %s %s %d %s", text, path, line, branch)
}

func (r *recorder) Open(path string, line int) {
	r.events <- fmt.Sprintf("open %s %d", path, line)
}

func (r *recorder) Stop() {
	r.events <- "stop"
}

func TestEditorCommandsReachTheSessionHandler(t *testing.T) {
	commands := newRecorder()
	srv := startTestServer(t, Options{Commands: commands})
	sock, _ := attach(t, srv)

	write(t, sock, `{"v":1,"t":"hello","root":"/repo","pid":1}`)
	write(t, sock, `{"v":1,"t":"send","text":"fix the test","path":"a.go","line":7,"branch":"main"}`)
	write(t, sock, `{"v":1,"t":"open","path":"b.go","line":3}`)
	write(t, sock, `{"v":1,"t":"stop"}`)

	want := []string{"send fix the test a.go 7 main", "open b.go 3", "stop"}
	for _, expected := range want {
		select {
		case got := <-commands.events:
			if got != expected {
				t.Errorf("the session handled %q, want %q", got, expected)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("the session never handled %q", expected)
		}
	}
}

func TestEditorCommandsAreDroppedWhenNobodyHandlesThem(t *testing.T) {
	srv := startTestServer(t, Options{})
	sock, dec := attach(t, srv)

	write(t, sock, `{"v":1,"t":"send","text":"fix the test"}`)
	srv.Publish(Turn(1))

	ev := readEvent(t, dec)
	wantString(t, ev, "t", "turn")
}
