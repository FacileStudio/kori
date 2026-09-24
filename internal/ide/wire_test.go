package ide

import (
	"encoding/json"
	"strings"
	"testing"
)

// wantEvent checks that one object carries exactly the fields its type is
// specified to carry, with the values it is specified to carry them with.
func wantEvent(t *testing.T, ev map[string]any, keys []string, value map[string]any) {
	t.Helper()
	if len(ev) != len(keys) {
		t.Errorf("object carries %d fields, want exactly %d: %v", len(ev), len(keys), ev)
	}
	for _, key := range keys {
		if _, ok := ev[key]; !ok {
			t.Errorf("object is missing %q: %v", key, ev)
		}
	}
	for key, want := range value {
		if ev[key] != want {
			t.Errorf("field %q = %v, want %v (object: %v)", key, ev[key], want, ev)
		}
	}
}

// assertWireShape checks the framing over the bytes that actually crossed the
// socket: every object on a line of its own, every line a whole JSON object,
// and the stream ending on a newline. A literal newline inside an object cannot
// survive json.Valid, which is the same guarantee the encoder has to keep.
func assertWireShape(t *testing.T, raw string, want int) {
	t.Helper()
	if !strings.HasSuffix(raw, "\n") {
		t.Errorf("the stream does not end with a newline: %q", raw)
	}
	lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
	if len(lines) != want {
		t.Fatalf("the stream carried %d lines, want %d: %q", len(lines), want, raw)
	}
	for i, line := range lines {
		if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") || !json.Valid([]byte(line)) {
			t.Errorf("line %d is not one whole JSON object: %q", i, line)
		}
	}
}

// TestHelloIsTheFirstLineUnderAPublishingSession pins the ordering the accept
// path has to keep: an editor is never told about a tool call before it has
// been told what it attached to, even when the session is publishing as it
// connects.
func TestHelloIsTheFirstLineUnderAPublishingSession(t *testing.T) {
	srv := startTestServer(t, Options{})
	go func() {
		for i := range 200 {
			srv.Publish(Turn(i))
		}
	}()

	e := dial(t, srv)
	wantString(t, e.object(), "t", "hello")
}
