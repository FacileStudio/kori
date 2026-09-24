package ide

import (
	"context"
	"testing"
	"time"
)

// pending approves one call in the background and waits until the editor has
// been asked, so a test can answer, break the connection or close the session
// while the session is still waiting on it.
func pending(t *testing.T, srv *Server, dec *Decoder, ctx context.Context) <-chan Answer {
	t.Helper()
	answers := make(chan Answer, 1)
	go func() { answers <- srv.Approve(ctx, "call-1", "run_command", `{"command":"ls"}`) }()
	wantString(t, readEvent(t, dec), "t", "approval")
	return answers
}

// refused checks that a call came back without an explicit allow, whichever
// way it was interrupted.
func refused(t *testing.T, answers <-chan Answer) {
	t.Helper()
	select {
	case got := <-answers:
		if got == Allowed {
			t.Error("a call was allowed without an explicit allow")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the approval never came back")
	}
}

// TestApproveRefusesAnAnswerThatNamesNothing pins the fail-closed floor for a
// reply that lost its id: an approve naming nothing answers nothing, even when
// the session is waiting under an empty id.
func TestApproveRefusesAnAnswerThatNamesNothing(t *testing.T) {
	srv := startTestServer(t, Options{})
	sock, dec := attach(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	answers := make(chan Answer, 1)
	go func() { answers <- srv.Approve(ctx, "", "run_command", `{"command":"ls"}`) }()
	readEvent(t, dec)
	write(t, sock, `{"v":1,"t":"approve","allow":true}`)

	refused(t, answers)
}

// TestApproveRefusesAMalformedReply pins that a line the session cannot read
// drops the editor, and dropping the editor refuses the call it was asked.
func TestApproveRefusesAMalformedReply(t *testing.T) {
	srv := startTestServer(t, Options{})
	sock, dec := attach(t, srv)
	answers := pending(t, srv, dec, context.Background())

	write(t, sock, `{"v":1,"t":"approve","id":"call-1",`)

	refused(t, answers)
}

// TestApproveRefusesAProtocolViolation pins that a reply carrying a version
// the session does not know is refused rather than read, and that the refusal
// is the answer to the call that was waiting.
func TestApproveRefusesAProtocolViolation(t *testing.T) {
	srv := startTestServer(t, Options{})
	sock, dec := attach(t, srv)
	answers := pending(t, srv, dec, context.Background())

	write(t, sock, `{"v":2,"t":"approve","id":"call-1","allow":true}`)

	refused(t, answers)
}

// TestApproveRefusesAnUnknownReplyType pins the other side of the same floor:
// a reply the session ignores leaves the call waiting, and a wait that runs
// out is a refusal rather than an answer that never arrived being read as one.
func TestApproveRefusesAnUnknownReplyType(t *testing.T) {
	srv := startTestServer(t, Options{})
	sock, dec := attach(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	answers := pending(t, srv, dec, ctx)

	write(t, sock, `{"v":1,"t":"approve-later","id":"call-1","allow":true}`)

	refused(t, answers)
}

// TestApproveRefusesWhenTheSessionClosesMidWait pins that a session going down
// under a waiting call denies it instead of leaving the caller blocked.
func TestApproveRefusesWhenTheSessionClosesMidWait(t *testing.T) {
	srv := startTestServer(t, Options{})
	_, dec := attach(t, srv)
	answers := pending(t, srv, dec, context.Background())

	if err := srv.Close(); err != nil {
		t.Fatalf("closing the IDE server: %v", err)
	}

	refused(t, answers)
}
