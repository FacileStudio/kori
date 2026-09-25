package chat

import (
	"strings"
	"testing"
	"unicode/utf8"

	"maunium.net/go/mautrix/event"
)

const questionEvent = "$question:example.org"

// TestReplyToThreadsOntoTheQuestion proves an answer attaches to the message it
// answers. The relation is what a client renders as a reply and what a server
// reads to aggregate a thread, so losing it turns every answer into an
// unanchored message in a busy room.
func TestReplyToThreadsOntoTheQuestion(t *testing.T) {
	to := Identity{Adapter: "matrix", Room: roomID, Sender: "@alice:example.org", Event: questionEvent}
	content := replyTo(to, "the answer")
	if content.RelatesTo == nil {
		t.Fatal("no relation on a reply to a known event")
	}
	if got := content.RelatesTo.GetReplyTo(); got.String() != questionEvent {
		t.Errorf("reply target = %q, want %q", got, questionEvent)
	}
	if content.Mentions == nil || !content.Mentions.Has("@alice:example.org") {
		t.Errorf("mentions = %+v, want the sender mentioned so the reply notifies them", content.Mentions)
	}
	if content.MsgType != event.MsgText || content.Body != "the answer" {
		t.Errorf("content = %+v, want an m.text carrying the body", content)
	}
}

// TestReplyToStandsAloneWithoutATarget covers the other branch: an identity
// with no event id (a message an adapter did not stamp) must still produce a
// sendable message rather than a relation pointing at nothing.
func TestReplyToStandsAloneWithoutATarget(t *testing.T) {
	content := replyTo(Identity{Adapter: "matrix", Room: roomID, Sender: "@alice:example.org"}, "the answer")
	if content.RelatesTo != nil && content.RelatesTo.GetReplyTo() != "" {
		t.Errorf("reply target = %q, want none", content.RelatesTo.GetReplyTo())
	}
	if content.Body != "the answer" {
		t.Errorf("body = %q, want the answer", content.Body)
	}
}

// TestChunksSplitsWithoutLosingAnything is the property that matters for a
// split answer: every byte of the answer goes out, in order, exactly once. The
// homeserver caps one event at 65536 bytes and does not truncate for you, so a
// long answer is either chunked here or lost with M_TOO_LARGE.
func TestChunksSplitsWithoutLosingAnything(t *testing.T) {
	long := strings.Repeat("a line of the answer\n", 500)
	parts := chunks(long, 200)
	if len(parts) < 2 {
		t.Fatalf("split into %d parts, want several", len(parts))
	}
	if got := strings.Join(parts, ""); got != long {
		t.Errorf("rejoining the chunks lost or reordered %d bytes", len(long)-len(got))
	}
	for i, part := range parts {
		if len(part) > 200 {
			t.Errorf("chunk %d is %d bytes, over the 200 limit", i, len(part))
		}
	}
}

// TestChunksKeepsShortTextWhole pins the ordinary case: a normal reply is one
// message, so the split is invisible and a reader sees no seam.
func TestChunksKeepsShortTextWhole(t *testing.T) {
	parts := chunks("short answer", 200)
	if len(parts) != 1 || parts[0] != "short answer" {
		t.Fatalf("chunks = %q, want the text whole", parts)
	}
}

// TestChunksSplitsOnRuneBoundaries proves a cut never lands inside a character.
// Cutting a multi-byte rune in half sends a replacement character, and prose in
// most languages is multi-byte, so this is a rendering bug on every answer past
// the limit rather than an exotic one.
func TestChunksSplitsOnRuneBoundaries(t *testing.T) {
	text := strings.Repeat("é", 100)
	parts := chunks(text, 7)
	for i, part := range parts {
		if !utf8.ValidString(part) {
			t.Fatalf("chunk %d is not valid UTF-8: %q", i, part)
		}
	}
	if got := strings.Join(parts, ""); got != text {
		t.Errorf("runes lost across the split: %d in, %d out", len(text), len(got))
	}
}

// TestCutPointPrefersALineBreak is why the split reads well: a cut at a line
// break ends on a boundary the model itself wrote, rather than mid-word.
func TestCutPointPrefersALineBreak(t *testing.T) {
	if got := cutPoint("first line\nsecond line\n", 20); got != len("first line\n") {
		t.Errorf("cut at %d, want the line break at %d", got, len("first line\n"))
	}
	if got := cutPoint(strings.Repeat("x", 10), 4); got != 4 {
		t.Errorf("cut at %d, want 4 when there is no line break", got)
	}
}
