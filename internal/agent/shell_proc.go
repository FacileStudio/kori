package agent

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"time"
)

// launchShellPipes wires one session shell's plumbing by hand and starts
// it. The pipes are created rather than taken from StdinPipe and friends,
// because those close the read end behind their back on Wait and this
// session holds it open across calls; stdout and stderr share one pipe the
// way the stateless runner merges them, so interleaved output keeps its
// order. The parent's ends of what the child inherited close before Start
// returns, so an exiting shell is seen as EOF and nothing holds the write
// side.
func launchShellPipes(cmd *exec.Cmd) (io.WriteCloser, *os.File, error) {
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return nil, nil, errors.Join(err, stdinR.Close(), stdinW.Close())
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdinR, stdoutW, stdoutW

	if err := cmd.Start(); err != nil {
		return nil, nil, errors.Join(err, stdinR.Close(), stdinW.Close(), stdoutR.Close(), stdoutW.Close())
	}
	return stdinW, stdoutR, errors.Join(stdinR.Close(), stdoutW.Close())
}

// respawn replaces a shell that died since the last call. Any state it held
// is gone with it; the next command starts fresh, the way the stateless
// runner starts every command.
func (s *shellSession) respawn() error {
	reapErr := s.reap(false)
	if err := s.spawn(); err != nil {
		return errors.Join(err, reapErr)
	}
	return nil
}

// write sends one NUL-terminated record to the shell's stdin.
func (s *shellSession) write(payload string) error {
	_, err := io.WriteString(s.stdin, payload+"\x00")
	return err
}

// drain discards whatever the shell wrote after the marker, so a background
// job's output does not resurface at the head of the next call's result.
// A short read deadline catches the bytes already in flight plus anything
// arriving just behind them; the tool's description points long jobs at a
// log file instead of the pipe.
func (s *shellSession) drain() {
	if s.stdout == nil {
		return
	}
	if err := s.stdout.SetReadDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		return
	}
	chunk := make([]byte, 32*1024)
	for {
		if _, err := s.stdout.Read(chunk); err != nil {
			return
		}
	}
}
