package ide

import (
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestCloseEndsEveryConnection pins that an attached editor is dropped by the
// close rather than left holding a socket nobody serves any more.
func TestCloseEndsEveryConnection(t *testing.T) {
	srv := startTestServer(t, Options{})
	e := dial(t, srv)
	wantString(t, e.object(), "t", "hello")

	if err := srv.Close(); err != nil {
		t.Fatalf("closing the IDE server: %v", err)
	}
	if rest := e.rest(); rest != "" {
		t.Errorf("the connection carried %q after the close, want it ended", rest)
	}
}

// TestNoGoroutineOutlivesClose pins that the accept loop and one read loop per
// editor are all gone once the close returns, and that the close returns at all
// with three editors still attached.
func TestNoGoroutineOutlivesClose(t *testing.T) {
	before := runtime.NumGoroutine()
	srv := startTestServer(t, Options{})
	for range 3 {
		e := dial(t, srv)
		e.object()
		srv.Publish(Turn(1))
		e.object()
	}

	closed := make(chan error, 1)
	go func() { closed <- srv.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatalf("closing the IDE server: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the close never returned with three editors attached")
	}
	settle(t, before)
}

// settle waits for the goroutine count to come back to what it was before the
// surface opened, so a read loop that outlives the close fails the test instead
// of leaking quietly.
func settle(t *testing.T, before int) {
	t.Helper()
	for range 200 {
		if runtime.NumGoroutine() <= before {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("goroutines outlived the close: %d, want %d", runtime.NumGoroutine(), before)
}

// TestPublishDoesNotBlockOnAnEditorThatStopsReading pins the bound on a write.
// A client that stops reading fills its socket buffer, and without the bound
// the session's own hook would wait on it forever.
func TestPublishDoesNotBlockOnAnEditorThatStopsReading(t *testing.T) {
	srv := startTestServer(t, Options{})
	e := dial(t, srv)
	wantString(t, e.object(), "t", "hello")

	big := Edit("edit_file-1", Change{Path: "a.go", Tool: "edit_file", Diff: strings.Repeat("x", 1<<20)})
	published := make(chan struct{})
	go func() {
		defer close(published)
		for range 200 {
			srv.Publish(big)
		}
	}()

	select {
	case <-published:
	case <-time.After(writeTimeout + 2*time.Second):
		t.Fatal("publishing blocked on an editor that stopped reading")
	}
}

// TestSetSessionCannotOutliveClose pins that the discovery file a late session
// rewrites is gone once the close returns: the rewrite happens under the same
// lock, so it cannot land after the file was taken down.
func TestSetSessionCannotOutliveClose(t *testing.T) {
	srv := startTestServer(t, Options{})
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			<-start
			if err := srv.setSession("/tmp/late.jsonl"); err != nil {
				t.Errorf("recording a late session: %v", err)
			}
		})
	}
	wg.Go(func() {
		<-start
		if err := srv.Close(); err != nil {
			t.Errorf("closing the IDE server: %v", err)
		}
	})
	close(start)
	wg.Wait()

	if err := srv.setSession("/tmp/after.jsonl"); err != nil {
		t.Errorf("recording a session after the close: %v", err)
	}
	if _, err := os.Stat(File(os.Getpid())); !os.IsNotExist(err) {
		t.Errorf("the discovery file came back after the close: %v", err)
	}
}
