package agent

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// shellDonePrefix opens the line the session shell prints after every
// command: the prefix, this call's nonce, and the command's exit status.
const shellDonePrefix = "__NC_DONE_"

// shellLoop is the program the session shell runs for its whole life. Each
// NUL-delimited record on its stdin is one payload line; the eval runs it
// in the current shell, so cd, export and shell options survive between
// calls. The command's own stdin is /dev/null: an interactive child would
// otherwise eat the next record, and the stateless runner this tool's
// results must match gives children /dev/null too.
const shellLoop = `while IFS= read -r -d '' __nc_payload; do
  eval "$__nc_payload" </dev/null
done`

// shellPayload wraps one command for the loop. The whole record is one line
// with the command single-quoted, so an unterminated if or heredoc inside
// the command fails inside its own eval instead of swallowing the marker
// and hanging the session. The nonce makes the marker unfakeable by output
// that merely mentions it.
func shellPayload(nonce, command string) string {
	return "eval " + shellQuote(command) + " </dev/null; __nc_rc=$?; printf '" +
		shellDonePrefix + nonce + ":%s\\n' \"$__nc_rc\""
}

// shellQuote single-quotes a string for shell use, the '\” dance for the
// embedded quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// shellNonce is 16 hex characters from the crypto source.
func shellNonce() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "deadbeefdeadbeef"
	}
	return hex.EncodeToString(buf)
}

// findDone looks for the marker line in data starting at from, so chunks
// already scanned are not searched twice. It answers the marker's offset
// and the exit status that followed it, and accepts only a marker with a
// full decimal status and a terminating newline, so echoed or half-arrived
// text never passes.
func findDone(data []byte, from int, marker string) (int, int, bool) {
	if from < 0 {
		from = 0
	}
	if from > len(data) {
		from = len(data)
	}
	for i := from; ; {
		at := bytes.Index(data[i:], []byte(marker))
		if at < 0 {
			return 0, 0, false
		}
		at += i
		if rc, ok := doneStatus(data, at+len(marker)); ok {
			return at, rc, true
		}
		i = at + 1
	}
}

// doneStatus reads the exit status that follows a marker, refusing
// anything that is not a complete decimal line.
func doneStatus(data []byte, start int) (int, bool) {
	end := start
	for end < len(data) && end-start < 8 && data[end] >= '0' && data[end] <= '9' {
		end++
	}
	if end == start || end >= len(data) || data[end] != '\n' {
		return 0, false
	}
	rc := 0
	for _, digit := range data[start:end] {
		rc = rc*10 + int(digit-'0')
	}
	return rc, true
}

// emitLines hands every completed line in data past pos to emit, leaving a
// trailing partial line unemitted — the same buffering the stateless
// runner's emitter applies.
func emitLines(data []byte, pos *int, emit func(string)) {
	for *pos < len(data) {
		i := bytes.IndexByte(data[*pos:], '\n')
		if i < 0 {
			return
		}
		emit(string(data[*pos : *pos+i]))
		*pos += i + 1
	}
}
