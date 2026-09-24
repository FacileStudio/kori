package ide

import (
	"bytes"
	"net"
	"strings"
	"testing"
)

func TestEncodedEventsAreOneObjectPerLine(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(ToolStart("c-1", "edit_file", "a.go")); err != nil {
		t.Fatalf("encoding a tool event: %v", err)
	}
	if err := enc.Encode(Err("multiline\nreason")); err != nil {
		t.Fatalf("encoding an error event: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("wrote %d lines for 2 events: %q", len(lines), buf.String())
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
			t.Errorf("a line is not one whole object: %q", line)
		}
	}

	dec := NewDecoder(bytes.NewReader(buf.Bytes()))
	for _, want := range []string{"tool", "error"} {
		var ev map[string]any
		if err := dec.Decode(&ev); err != nil {
			t.Fatalf("decoding the %s event back: %v", want, err)
		}
		wantString(t, ev, "t", want)
		wantNumber(t, ev, "v", Protocol)
	}
}

func TestDecodeRoundTripsOverASocket(t *testing.T) {
	srv := startTestServer(t, Options{Root: "/repo", Model: "claude", Version: "v0.76.0"})
	sock, dec := attach(t, srv)
	if _, ok := sock.(*net.UnixConn); !ok {
		t.Fatalf("the test client dialled something other than a unix socket: %T", sock)
	}

	srv.Publish(ToolStart("edit_file-1", "edit_file", "a.go"))
	ev := readEvent(t, dec)
	wantString(t, ev, "t", "tool")
	wantString(t, ev, "id", "edit_file-1")
	wantString(t, ev, "name", "edit_file")
	wantString(t, ev, "status", "start")
	wantString(t, ev, "path", "a.go")
}

func TestDecodeIgnoresBlankLinesAndUnknownFields(t *testing.T) {
	stream := "\n\n{\"v\":1,\"t\":\"send\",\"text\":\"hi\",\"from_the_future\":42}\n"
	dec := NewDecoder(strings.NewReader(stream))
	var cmd Command
	if err := dec.Decode(&cmd); err != nil {
		t.Fatalf("decoding a command with an unknown field: %v", err)
	}
	if cmd.T != "send" || cmd.Text != "hi" {
		t.Errorf("command = %+v, want the send with its text", cmd)
	}
}

func TestDecodeKeepsAnUnknownTypeForTheCallerToIgnore(t *testing.T) {
	dec := NewDecoder(strings.NewReader("{\"v\":1,\"t\":\"something-new\"}\n"))
	var cmd Command
	if err := dec.Decode(&cmd); err != nil {
		t.Fatalf("decoding an unknown type: %v", err)
	}
	if cmd.T != "something-new" {
		t.Errorf("command type = %q, want the type read as written", cmd.T)
	}
}
