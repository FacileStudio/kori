package ide

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// editor is the far side of the socket: a client that reads the exact bytes a
// plugin receives and writes the exact lines one sends. The framing, the accept
// loop and the per type field sets are what a mocked net.Conn would hide, so
// the tests that matter drive this one over a real unix socket.
type editor struct {
	t    *testing.T
	conn net.Conn
	r    *bufio.Reader
	seen strings.Builder
}

// dial connects a fake editor and reads nothing, so a test can say what the
// session opens with.
func dial(t *testing.T, srv *Server) *editor {
	t.Helper()
	conn, err := net.Dial("unix", srv.path)
	if err != nil {
		t.Fatalf("dialling the IDE socket: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return &editor{t: t, conn: conn, r: bufio.NewReader(conn)}
}

// line returns the next raw line an editor was sent, without its terminator.
// The read is bounded, so a session that stops framing fails the test rather
// than hanging it.
func (e *editor) line() string {
	e.t.Helper()
	if err := e.conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		e.t.Fatalf("bounding the read: %v", err)
	}
	raw, err := e.r.ReadString('\n')
	if err != nil {
		e.t.Fatalf("reading a line: %v", err)
	}
	e.seen.WriteString(raw)
	return strings.TrimSuffix(raw, "\n")
}

// object reads the next line and decodes it as one protocol object.
func (e *editor) object() map[string]any {
	e.t.Helper()
	var ev map[string]any
	if err := json.Unmarshal([]byte(e.line()), &ev); err != nil {
		e.t.Fatalf("decoding an object: %v", err)
	}
	return ev
}

// send writes one raw line the way an editor does.
func (e *editor) send(line string) {
	e.t.Helper()
	if _, err := io.WriteString(e.conn, line+"\n"); err != nil {
		e.t.Fatalf("writing %s: %v", line, err)
	}
}

// rest reads to the end of the connection and reports what was left on it,
// which is empty once the session has closed its side. The read is bounded so
// a connection the session never closed fails the test instead of hanging it.
func (e *editor) rest() string {
	e.t.Helper()
	if err := e.conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		e.t.Fatalf("bounding the read: %v", err)
	}
	raw, err := io.ReadAll(e.r)
	if err != nil {
		e.t.Fatalf("reading the rest of the connection: %v", err)
	}
	e.seen.Write(raw)
	return string(raw)
}
