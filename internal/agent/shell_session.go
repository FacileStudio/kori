package agent

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// errShellDown reports that the session shell could not start. The tool
// answers it by falling back to the stateless runner for the rest of the
// session.
var errShellDown = errors.New("persistent shell unavailable")

// errShellNoCommand is the stateless runner's own empty-input refusal.
var errShellNoCommand = errors.New("no command given")

// shellSession is one long-lived bash driving the shellLoop protocol over
// pipes: commands arrive NUL-delimited on stdin, output merges on one
// stdout pipe, and a marker line closes each call. It belongs to one agent
// session; calls are serialised because a shell has one stdin, and state —
// the current directory, exports, background jobs — survives between them.
//
// Teardown is deliberately three-limbed. An explicit Close ends it; the
// finalizer the tool sets on itself ends it when the agent is dropped and
// nobody called Close; and when the process exits outright, bash sees EOF
// on stdin and the loop ends on its own. The finalizer is the safety net
// for hosts that rebuild tool sets without restarting, not the primary
// path.
type shellSession struct {
	dir           string
	env           []string
	denyElevation bool
	binary        string

	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *os.File
	alive  bool
	broken bool
	waitCh chan error
}

// newShellSession prepares the session shell for one agent session. dir is
// where the first command starts; env is what the shell runs with, nil
// meaning the guarded minimal environment — the same posture the stateless
// runner is built with.
func newShellSession(dir string, env []string, denyElevation bool) *shellSession {
	if env == nil {
		env = minimalShellEnv()
	}
	return &shellSession{dir: dir, env: env, denyElevation: denyElevation, binary: "bash"}
}

// accept applies the same refusals the stateless runner would before any
// command reaches the shell.
func (s *shellSession) accept(command string) error {
	if strings.TrimSpace(command) == "" {
		return errShellNoCommand
	}
	if s.denyElevation {
		return shellCheckElevation(command)
	}
	return nil
}

// ensure makes sure a live shell is waiting for work, spawning one or
// replacing one that died — the model can run exit, and a dead session
// must not become a dead tool.
func (s *shellSession) ensure() error {
	if s.cmd != nil && s.alive {
		select {
		case <-s.waitCh:
			s.waitCh = nil
			s.alive = false
		default:
		}
	}
	if s.alive && s.cmd != nil {
		return nil
	}
	return s.spawn()
}

// spawn starts bash on the loop protocol in its own process group behind
// hand-rolled pipes.
func (s *shellSession) spawn() error {
	cmd := exec.Command(s.binary, "--noprofile", "--norc", "-c", shellLoop)
	cmd.Dir = s.dir
	cmd.Env = s.env
	cmd.SysProcAttr = shellSetpgidAttr()

	stdinW, stdoutR, err := launchShellPipes(cmd)
	if err != nil {
		return err
	}

	if err := s.closePipes(); err != nil {
		s.broken = true
		return err
	}
	s.cmd = cmd
	s.stdin = stdinW
	s.stdout = stdoutR
	s.alive = true
	s.waitCh = make(chan error, 1)
	go func() { s.waitCh <- cmd.Wait() }()
	return nil
}

// reapReady collects the shell's exit when it has already exited, and does
// nothing when it is still running.
func (s *shellSession) reapReady() {
	if s.waitCh == nil {
		return
	}
	select {
	case <-s.waitCh:
		s.waitCh = nil
		s.alive = false
	default:
	}
}

// reap waits out the shell's exit, asking the group to stop first when
// asked, then making it. It answers whatever Wait reported, or nil when the
// wait was given up on.
func (s *shellSession) reap(killFirst bool) error {
	if s.waitCh == nil {
		s.alive = false
		return nil
	}
	if killFirst && s.alive {
		shellTerminateGroup(s.cmd)
	}
	select {
	case err := <-s.waitCh:
		s.waitCh = nil
		s.alive = false
		return err
	case <-time.After(shellGrace):
	}
	shellForceKillGroup(s.cmd)
	select {
	case err := <-s.waitCh:
		s.waitCh = nil
		s.alive = false
		return err
	case <-time.After(shellGrace):
		return nil
	}
}

// Close ends the session: stdin closes so the loop unwinds, the process
// group is asked to stop and then made to, and the pipes go. Safe to call
// twice; the finalizer calls it after the caller already has. It reports
// whatever the teardown hit, though nothing can usually act on it.
func (s *shellSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	if s.alive && s.stdin != nil {
		err = s.stdin.Close()
	}
	err = errors.Join(err, s.reap(true), s.closePipes())
	s.alive = false
	s.broken = true
	return err
}

// closePipes drops whatever pipes a previous shell still held.
func (s *shellSession) closePipes() error {
	var err error
	if s.stdin != nil {
		err = s.stdin.Close()
		s.stdin = nil
	}
	if s.stdout != nil {
		err = errors.Join(err, s.stdout.Close())
		s.stdout = nil
	}
	return err
}
