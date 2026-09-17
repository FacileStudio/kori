package sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	StatusRunning   = "running"
	StatusIdle      = "idle"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// SessionInfo records metadata, process liveness, and status for a session.
type SessionInfo struct {
	ID          string    `json:"id"`
	Path        string    `json:"path"`
	PID         int       `json:"pid"`
	Started     time.Time `json:"started"`
	ModTime     time.Time `json:"mod_time"`
	Backend     string    `json:"backend"`
	Model       string    `json:"model"`
	Root        string    `json:"root"`
	Status      string    `json:"status"`
	IsActive    bool      `json:"is_active"`
	LastMessage string    `json:"last_message"`
	TotalLines  int       `json:"total_lines"`
}

// IsPIDRunning reports whether a process with the given PID is currently alive.
func IsPIDRunning(pid int) bool {
	return isProcessAlive(pid)
}

func parseSessionInfo(path string) (*SessionInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	pid := extractPID(id)
	scan, err := scanSession(path)
	if err != nil {
		return nil, err
	}
	if scan.header.PID > 0 {
		pid = scan.header.PID
	}
	started := parseStarted(scan.header.Started, id, fi.ModTime())
	active := IsPIDRunning(pid)
	lastMsg, lastWho, explicit := inspectEntries(scan.entries)
	return &SessionInfo{
		ID:          id,
		Path:        path,
		PID:         pid,
		Started:     started,
		ModTime:     fi.ModTime(),
		Backend:     scan.header.Backend,
		Model:       scan.header.Model,
		Root:        scan.header.Root,
		Status:      computeStatus(active, explicit, lastWho),
		IsActive:    active,
		LastMessage: lastMsg,
		TotalLines:  scan.lines,
	}, nil
}

// ListSessions returns metadata and liveness info for all sessions matching the project root.
func ListSessions(projectRoot string) []SessionInfo {
	files := ListSessionFiles(projectRoot)
	if len(files) == 0 {
		return nil
	}
	var results []SessionInfo
	for _, p := range files {
		info, err := parseSessionInfo(p)
		if err != nil {
			continue
		}
		results = append(results, *info)
	}
	return results
}

// GetSession resolves a session identifier or path and returns its metadata and status.
func GetSession(idOrPath string) (*SessionInfo, error) {
	if idOrPath == "" {
		return nil, fmt.Errorf("session not found: empty identifier")
	}
	path := ResolveSession(idOrPath)
	if path == "" {
		if fi, err := os.Stat(idOrPath); err == nil && !fi.IsDir() {
			path = idOrPath
		}
	}
	if path == "" {
		return nil, fmt.Errorf("session not found: %s", idOrPath)
	}
	return parseSessionInfo(path)
}

// FormatSessionEntry formats a session file entry for display.
func FormatSessionEntry(filePath string) string {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Sprintf("  %s (error reading file)", filepath.Base(filePath))
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Sprintf("  %s (unreadable)", filepath.Base(filePath))
	}

	linesData := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(linesData) == 0 {
		return fmt.Sprintf("  %s · %s", filepath.Base(filePath), info.ModTime().Format("2006-01-02 15:04:05"))
	}

	var header sessionHeader
	if err := json.Unmarshal([]byte(linesData[0]), &header); err == nil && header.Version == 1 {
		started := header.Started
		if len(started) > 19 {
			started = started[:19]
		}
		return fmt.Sprintf("  %s · %s · %s · %s", filepath.Base(filePath), header.Backend, header.Model, started)
	}

	return fmt.Sprintf("  %s · %s", filepath.Base(filePath), info.ModTime().Format("2006-01-02 15:04:05"))
}
