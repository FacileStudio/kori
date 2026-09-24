package ide

import (
	"net"
	"slices"
	"sync"
	"time"
)

// conn is one attached editor: the socket, the encoder that keeps its lines
// whole, and nothing the session reads back except the answers it sends.
type conn struct {
	srv    *Server
	sock   net.Conn
	enc    *Encoder
	mu     sync.Mutex
	closed bool
}

// newConn wraps one accepted socket.
func newConn(s *Server, sock net.Conn) *conn {
	return &conn{srv: s, sock: sock, enc: NewEncoder(sock)}
}

// writeTimeout bounds one write to an editor. A client that stops reading must
// not be able to hold up the session that is publishing to it: the socket
// buffer fills, the write blocks, and a hook in the agent's own path waits on
// it. A local socket drains in microseconds, so only a client that has stopped
// reading reaches this, and that client is one to drop.
const writeTimeout = 2 * time.Second

// send writes one event, and drops the editor when the write fails: a client
// that has gone away must not take the session with it.
//
// An error object ends the attachment whatever produced it, because that is
// what the object means on the wire. Only the read loop raises one today, and
// it closes behind itself either way, so the rule is kept here rather than
// resting on every future caller remembering to.
func (c *conn) send(ev Event) {
	if err := c.sock.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		c.close()
		return
	}
	if err := c.enc.Encode(ev); err != nil {
		c.close()
		return
	}
	if ev.T == "error" {
		c.close()
	}
}

// close ends the attachment, as the cleanup it is: it reports nothing, because
// a socket that refuses to close is already gone and neither a caller mid
// publish nor one shutting down has anything left to do about it. The deferred
// close is where the error goes for the same reason — a cleanup's error is
// discarded in this repository on purpose, not by a bare call and not by
// assigning it to _ mid-function. Closing twice is harmless.
func (c *conn) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	defer func() { _ = c.sock.Close() }()
}

// serve reads one editor's commands until it goes away or breaks the
// protocol. Each attachment is read on its own goroutine, so the commands of
// one editor stay in the order it sent them.
func (c *conn) serve() {
	defer c.srv.detach(c)
	defer c.close()
	dec := NewDecoder(c.sock)
	for {
		var cmd Command
		if err := dec.Decode(&cmd); err != nil {
			return
		}
		if err := c.srv.handle(cmd); err != nil {
			c.send(Err(err.Error()))
			return
		}
	}
}

// accept takes editors as they dial, until the socket is closed.
func (s *Server) accept() {
	for {
		sock, err := s.ln.Accept()
		if err != nil {
			return
		}
		c := newConn(s, sock)
		if !s.attach(c) {
			c.close()
			return
		}
		s.wg.Go(c.serve)
	}
}

// attach records one editor and writes the hello it opens with, under the
// lock a publish takes: an editor is never told about a tool call before it
// has been told what it attached to.
func (s *Server) attach(c *conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.conns = append(s.conns, c)
	c.send(s.hello())
	return true
}

// detach forgets one editor that has gone away, and refuses whatever was
// waiting once none are left: with nobody attached there is nobody to ask, and
// a call nobody can answer is denied rather than left hanging.
func (s *Server) detach(c *conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns = slices.DeleteFunc(s.conns, func(other *conn) bool { return other == c })
	if len(s.conns) > 0 {
		return
	}
	for id, answer := range s.waiting {
		select {
		case answer <- false:
		default:
		}
		delete(s.waiting, id)
	}
}
