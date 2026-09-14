package agent

import (
	"context"
	"errors"
	"os"
	"time"
)

// call runs one command in the session shell and returns its result in the
// stateless runner's shape. The whole call — write, read, kill — holds the
// session lock, because one shell has one stdin and two interleaved
// commands would corrupt each other's protocol.
func (s *shellSession) call(ctx context.Context, command string, timeout time.Duration, maxOutput int, emit func(string)) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.broken {
		return "", errShellDown
	}
	if err := s.ensure(); err != nil {
		s.broken = true
		return "", errShellDown
	}

	inner, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	nonce, err := shellNonce()
	if err != nil {
		s.broken = true
		return "", errShellDown
	}
	payload := shellPayload(nonce, command)
	if err := s.writeWithRespawn(payload); err != nil {
		s.broken = true
		return "", errShellDown
	}
	return (&shellExchange{session: s, ctx: inner, timeout: timeout, maxOutput: maxOutput, emit: emit}).run(nonce)
}

// writeWithRespawn writes one payload to the shell's stdin, respawning a
// shell that died since the last call and writing the payload again.
func (s *shellSession) writeWithRespawn(payload string) error {
	if err := s.write(payload); err == nil {
		return nil
	}
	if err := s.respawn(); err != nil {
		return err
	}
	return s.write(payload)
}

// shellExchange is one call's read side: the bytes that arrived, how far
// they have been emitted and scanned, and everything the report needs.
type shellExchange struct {
	session   *shellSession
	ctx       context.Context
	timeout   time.Duration
	maxOutput int
	emit      func(string)
	data      []byte
	pos       int
	scanned   int
}

// run reads the shell's output until the marker line closes the command.
// The read carries the call's deadline, and a watcher unblocks it early
// when the context is cancelled, so an interrupt answers as fast as the
// stateless runner's does.
func (x *shellExchange) run(nonce string) (string, error) {
	marker := shellDonePrefix + nonce + ":"
	chunk := make([]byte, 32*1024)

	if deadline, ok := x.ctx.Deadline(); ok {
		_ = x.session.stdout.SetReadDeadline(deadline)
	}
	watching := make(chan struct{})
	go x.awaitCancel(watching)
	defer close(watching)

	for {
		n, err := x.session.stdout.Read(chunk)
		if n > 0 {
			if end, rc, done := x.absorb(chunk[:n], marker); done {
				return x.complete(end, rc), nil
			}
		}
		if err != nil {
			return x.finish(err)
		}
	}
}

// awaitCancel unblocks the read the moment the context is cancelled, which
// the deadline alone would not notice until it expired.
func (x *shellExchange) awaitCancel(watching <-chan struct{}) {
	select {
	case <-x.ctx.Done():
		_ = x.session.stdout.SetReadDeadline(time.Now())
	case <-watching:
	}
}

// absorb appends one chunk, streams every completed line it finished, and
// reports the marker when the whole call has arrived.
func (x *shellExchange) absorb(chunk []byte, marker string) (end, rc int, done bool) {
	x.data = append(x.data, chunk...)
	if end, rc, ok := findDone(x.data, x.scanned, marker); ok {
		return end, rc, true
	}
	emitLines(x.data, &x.pos, x.emit)
	if x.scanned = len(x.data) - len(marker) - 16; x.scanned < 0 {
		x.scanned = 0
	}
	return 0, 0, false
}

// complete stops reading — bytes after the marker belong to background
// jobs, so they are drained, not shown — and reports the result.
func (x *shellExchange) complete(end, rc int) string {
	emitLines(x.data[:end], &x.pos, x.emit)
	x.session.drain()
	x.session.reapReady()
	return shellReport(string(x.data[:end]), rc, nil, x.maxOutput)
}

// finish reports a call that ended without its marker. A cancelled context
// and a timeout both kill the group and the shell with it — the next call
// starts a fresh one — while a shell that died on its own reports its own
// exit status the way the stateless runner reports a command's.
func (x *shellExchange) finish(readErr error) (string, error) {
	switch {
	case errors.Is(x.ctx.Err(), context.Canceled):
		_ = x.session.reap(true)
		return shellReport(string(x.data), -1, x.ctx.Err(), x.maxOutput), x.ctx.Err()
	case errors.Is(readErr, os.ErrDeadlineExceeded) || x.ctx.Err() != nil:
		_ = x.session.reap(true)
		return shellReport(string(x.data), -1, shellTimedOut{after: x.timeout}, x.maxOutput), nil //nolint:nilerr // the timeout is reported in the report, not as an error
	default:
		dead := x.session.reap(false)
		failure := dead
		if failure == nil {
			failure = readErr
		}
		return shellReport(string(x.data), shellExitCode(dead), failure, x.maxOutput), nil
	}
}
