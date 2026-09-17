package sessions

import (
	"fmt"
	"os"
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
		return fmt.Errorf("session %s (PID %d) is not running", session.ID, session.PID)
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

// DeleteSession permanently removes a session file and any associated archives.
func DeleteSession(idOrPath string) error {
	path := ResolveSession(idOrPath)
	if path == "" {
		if fi, err := os.Stat(idOrPath); err == nil && !fi.IsDir() {
			path = idOrPath
		}
	}
	if path == "" {
		return fmt.Errorf("session not found: %s", idOrPath)
	}
	gzPath := path + ".gz"
	if fi, err := os.Stat(gzPath); err == nil && !fi.IsDir() {
		if err := os.Remove(gzPath); err != nil {
			return err
		}
	}
	return os.Remove(path)
}
