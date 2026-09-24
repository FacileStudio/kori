package ide

import (
	"bufio"
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

// sink is a Commands that throws away what reaches it, for the tests that only
// care that the session's own path stays sound while everything calls into it.
type sink struct{}

func (sink) Send(string, string, int, string) {}
func (sink) Open(string, int)                 {}
func (sink) Stop()                            {}

// TestConcurrentPublishAttachApproveAndClose runs everything the package
// exports against itself at once: four publishers, four editors connecting and
// going away, a handler being swapped, a call waiting for an answer, and the
// close landing in the middle. Under -race this is the test that says whether
// the locking holds.
func TestConcurrentPublishAttachApproveAndClose(t *testing.T) {
	srv := startTestServer(t, Options{})
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for i := range 50 {
				srv.Publish(Turn(i))
			}
		})
		wg.Go(func() { idle(srv) })
	}
	wg.Go(func() {
		srv.SetCommands(sink{})
		srv.SetCommands(nil)
	})
	wg.Go(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		if got := srv.Approve(ctx, "call-1", "run_command", `{"command":"ls"}`); got == Allowed {
			t.Errorf("Approve = %v with nobody to answer it, want anything but Allowed", got)
		}
	})

	time.Sleep(20 * time.Millisecond)
	if err := srv.Close(); err != nil {
		t.Errorf("closing the IDE server: %v", err)
	}
	wg.Wait()
}

// idle attaches an editor that reads what it is sent until the session drops
// it, and hands it one command on the way, which is what a plugin does while
// the session starts and stops around it.
func idle(srv *Server) {
	conn, err := net.Dial("unix", srv.path)
	if err != nil {
		return
	}
	defer conn.Close()
	if _, err := conn.Write([]byte(`{"v":1,"t":"stop"}` + "\n")); err != nil {
		return
	}
	reader := bufio.NewReader(conn)
	for {
		if _, err := reader.ReadString('\n'); err != nil {
			return
		}
	}
}
