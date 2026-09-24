package ide

import "errors"

// errUnsupported is what a receiver answers a protocol version it does not
// know: it refuses the object and closes, rather than reading fields whose
// meaning it cannot be sure of.
var errUnsupported = errors.New("unsupported protocol version")

// Commands is how an editor's prompts reach a session. A nil Commands still
// accepts the messages and drops them, so a session that never wires a handler
// (or wires one only after its socket is open) is an observer rather than a
// closed socket.
type Commands interface {
	Send(text, path string, line int, branch string)
	Open(path string, line int)
	Stop()
}

// handle applies one command from an attached editor. Only a protocol
// violation stops the attachment: an editor that speaks a type this session
// does not know, or answers a call nobody is waiting on, must not break the
// session it is watching.
func (s *Server) handle(cmd Command) error {
	if cmd.V != Protocol {
		return errUnsupported
	}
	switch cmd.T {
	case "approve":
		s.answer(cmd)
	case "send", "open", "stop":
		s.forward(cmd)
	}
	return nil
}

// forward hands one session command to the caller's own handler, if it has
// one.
func (s *Server) forward(cmd Command) {
	commands := s.commands()
	if commands == nil {
		return
	}
	switch cmd.T {
	case "send":
		commands.Send(cmd.Text, cmd.Path, cmd.Line, cmd.Branch)
	case "open":
		commands.Open(cmd.Path, cmd.Line)
	case "stop":
		commands.Stop()
	}
}

// commands reads the handler an editor's commands reach, under the lock
// SetCommands writes it with. The handler is set after the socket opens, so
// reading it without that lock would race the session's own start-up.
func (s *Server) commands() Commands {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.opts.Commands
}
