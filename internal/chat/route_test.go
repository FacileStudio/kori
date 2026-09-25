package chat

import (
	"testing"
	"time"
)

// testNow is the wall clock every staleness case is measured against, so a
// message built two hours "ago" is genuinely two hours old and the assertion
// on the dropped-message text can name an exact duration.
var testNow = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)

type checkCase struct {
	name    string
	router  Router
	msg     Message
	wantErr string
}

// checkCases is every refusal the router can produce. The allowlist is
// default-deny because it is the only wall between a room and a shell, so the
// empty-list case is pinned first, and each refusing case carries the exact
// error string the operator will read and paste into a bug report.
var checkCases = []checkCase{
	{
		name:    "empty allowlist refuses every sender",
		router:  Router{},
		msg:     matrixMsg("!room:example.org", "alice:example.org", testNow),
		wantErr: "refused matrix:!room:example.org: sender alice:example.org is not in the allowlist",
	},
	{
		name:   "exact mxid passes",
		router: Router{Allow: []string{"alice:example.org"}},
		msg:    matrixMsg("!room:example.org", "alice:example.org", testNow),
	},
	{
		name:   "case and surrounding space do not matter",
		router: Router{Allow: []string{" Alice:Example.org "}},
		msg:    matrixMsg("!room:example.org", "alice:example.org", testNow),
	},
	{
		name:    "a substring does not pass",
		router:  Router{Allow: []string{"alice"}},
		msg:     matrixMsg("!room:example.org", "alice:example.org", testNow),
		wantErr: "refused matrix:!room:example.org: sender alice:example.org is not in the allowlist",
	},
	{
		name:    "a different domain does not pass",
		router:  Router{Allow: []string{"alice:other.org"}},
		msg:     matrixMsg("!room:example.org", "alice:example.org", testNow),
		wantErr: "refused matrix:!room:example.org: sender alice:example.org is not in the allowlist",
	},
	{
		name:   "empty room allowlist accepts every room",
		router: Router{Allow: []string{"alice:example.org"}},
		msg:    matrixMsg("!any:example.org", "alice:example.org", testNow),
	},
	{
		name: "unlisted room is refused",
		router: Router{
			Allow: []string{"alice:example.org"},
			Rooms: []string{"!other:example.org"},
		},
		msg:     matrixMsg("!room:example.org", "alice:example.org", testNow),
		wantErr: "refused matrix:!room:example.org: room is not in the room allowlist",
	},
	{
		name:   "listed room is accepted",
		router: Router{Allow: []string{"alice:example.org"}, Rooms: []string{"!room:example.org"}},
		msg:    matrixMsg("!room:example.org", "alice:example.org", testNow),
	},
	{
		name:   "message inside max age passes",
		router: Router{Allow: []string{"alice:example.org"}, MaxAge: time.Hour},
		msg:    matrixMsg("!room:example.org", "alice:example.org", testNow.Add(-30*time.Minute)),
	},
	{
		name:    "weekend backlog is dropped, not replayed",
		router:  Router{Allow: []string{"alice:example.org"}, MaxAge: 2 * time.Hour},
		msg:     matrixMsg("!room:example.org", "alice:example.org", testNow.Add(-72*time.Hour)),
		wantErr: "dropped matrix:!room:example.org: message is 72h0m0s old, past the 2h0m0s limit",
	},
	{
		name:   "zero max age keeps the oldest message",
		router: Router{Allow: []string{"alice:example.org"}},
		msg:    matrixMsg("!room:example.org", "alice:example.org", testNow.Add(-72*time.Hour)),
	},
	{
		name:   "zero timestamp counts as fresh",
		router: Router{Allow: []string{"alice:example.org"}, MaxAge: time.Hour},
		msg:    matrixMsg("!room:example.org", "alice:example.org", time.Time{}),
	},
}

// matrixMsg builds an inbound message from the Matrix adapter, the only
// adapter these routing rules know about today.
func matrixMsg(room, sender string, at time.Time) Message {
	return Message{Identity: Identity{Adapter: "matrix", Room: room, Sender: sender}, At: at}
}

// TestCheck proves every routing decision in one pass: default-deny, the
// allowlist match, the room allowlist, and the staleness drop. The refusal
// text is compared whole, because it is what the operator sees on stderr.
func TestCheck(t *testing.T) {
	for _, tc := range checkCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.router.Check(tc.msg, testNow)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("Check refused an allowed message: %v", err)
			}
			if tc.wantErr != "" && err == nil {
				t.Fatalf("Check allowed a refused message, want %q", tc.wantErr)
			}
			if tc.wantErr != "" && err.Error() != tc.wantErr {
				t.Fatalf("refusal text changed:\n got %q\nwant %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// TestIdentityKey pins the session key to adapter plus room, and to nothing
// else. Two senders in one room must land in one session, and two rooms must
// not: --continue resumes the newest session for a project, so a key that
// collapsed rooms would interleave two chats into one transcript, which is the
// defect this surface exists to avoid.
func TestIdentityKey(t *testing.T) {
	base := Identity{Adapter: "matrix", Room: "!room:example.org", Sender: "alice:example.org"}
	if got, want := base.Key(), "matrix:!room:example.org"; got != want {
		t.Fatalf("Key() = %q, want %q", got, want)
	}

	otherSender := base
	otherSender.Sender = "bob:example.org"
	if base.Key() != otherSender.Key() {
		t.Errorf("two senders in one room must share a key: %q vs %q", base.Key(), otherSender.Key())
	}

	otherRoom := base
	otherRoom.Room = "!other:example.org"
	if base.Key() == otherRoom.Key() {
		t.Errorf("two rooms must not share a key: %q", base.Key())
	}

	otherAdapter := base
	otherAdapter.Adapter = "telegram"
	if base.Key() == otherAdapter.Key() {
		t.Errorf("two adapters must not share a key: %q", base.Key())
	}
}
