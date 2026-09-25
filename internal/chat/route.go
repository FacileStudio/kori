package chat

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Router decides which inbound messages run, and serializes the ones that do
// so two messages in one conversation cannot interleave into one answer.
//
// Every list is checked here, before a run is built, and never inside the run:
// an agent that is asked to refuse its own input has already been given the
// input. Both lists empty refuses every message, which is the right default
// for a surface whose allowlist is the only wall between a room and a shell.
type Router struct {
	Allow  []string
	Rooms  []string
	MaxAge time.Duration
}

// Check reports why a message must not be answered, or nil when it may be. A
// zero MaxAge keeps every message, and a zero At counts as now: an adapter
// that does not stamp its messages has not shown that they are old.
func (r Router) Check(m Message, now time.Time) error {
	if !listed(r.Allow, m.Sender) {
		return fmt.Errorf("refused %s: sender %s is not in the allowlist", m.Key(), m.Sender)
	}
	if len(r.Rooms) > 0 && !listed(r.Rooms, m.Room) {
		return fmt.Errorf("refused %s: room is not in the room allowlist", m.Key())
	}
	if r.MaxAge > 0 && !m.At.IsZero() && now.Sub(m.At) > r.MaxAge {
		return fmt.Errorf("dropped %s: message is %s old, past the %s limit",
			m.Key(), now.Sub(m.At).Round(time.Second), r.MaxAge)
	}
	return nil
}

// listed compares without case, because a Matrix user ID is lowercased by the
// server while a human writing one into a config file has no reason to know
// that. Surrounding space is trimmed for the same reason: it is invisible.
func listed(list []string, want string) bool {
	for _, have := range list {
		if strings.EqualFold(strings.TrimSpace(have), want) {
			return true
		}
	}
	return false
}

// serial chains work per session key so that one conversation is answered one
// message at a time, in the order the messages arrived. A plain lock is not
// enough: it stops two answers interleaving but still lets the second question
// be answered first, and a transcript that answers out of order is wrong in
// the way a human notices immediately.
//
// The chain is built by wait, which the receive callback calls. That callback
// is single-threaded and in arrival order, which is the only place the order
// is known.
type serial struct {
	mu    sync.Mutex
	tails map[string]chan struct{}
}

func newSerial() *serial {
	return &serial{tails: map[string]chan struct{}{}}
}

// wait hands back the gate this key's next piece of work must pass and the
// release it must call when finished. A nil gate means nothing is ahead of it.
func (s *serial) wait(key string) (gate <-chan struct{}, release func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev := s.tails[key]
	done := make(chan struct{})
	s.tails[key] = done
	return prev, func() { close(done) }
}
