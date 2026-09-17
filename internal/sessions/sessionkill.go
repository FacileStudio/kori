package sessions

import (
	"fmt"
	"time"
)

func waitForProcessExit(pid int, attempts int, delay time.Duration) bool {
	for range attempts {
		time.Sleep(delay)
		if !IsPIDRunning(pid) {
			return true
		}
	}
	return false
}

// KillSession terminates a running background session process.
func KillSession(idOrPath string) error {
	session, err := GetSession(idOrPath)
	if err != nil {
		return err
	}
	if session.PID <= 0 {
		return fmt.Errorf("invalid pid %d for session %s", session.PID, session.ID)
	}
	if !IsPIDRunning(session.PID) {
		return MarkSessionStatus(session.Path, StatusCompleted)
	}

	if err := terminateProcess(session.PID); err != nil {
		return err
	}
	if waitForProcessExit(session.PID, 10, 50*time.Millisecond) {
		return MarkSessionStatus(session.Path, StatusCompleted)
	}

	if err := killProcess(session.PID); err != nil {
		return err
	}
	waitForProcessExit(session.PID, 10, 20*time.Millisecond)
	return MarkSessionStatus(session.Path, StatusCompleted)
}
