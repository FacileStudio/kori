package sessions

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
)

type sessionScan struct {
	header  sessionHeader
	entries []sessionEntry
	lines   int
}

func extractPID(id string) int {
	idx := strings.LastIndex(id, "-")
	if idx == -1 {
		return 0
	}
	pid, _ := strconv.Atoi(id[idx+1:])
	return pid
}

func parseStarted(raw, id string, fallback time.Time) time.Time {
	if raw != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t
		}
	}
	if len(id) >= 16 {
		if t, err := time.Parse("20060102T150405Z", id[:16]); err == nil {
			return t
		}
	}
	return fallback
}

func parseSessionLine(line string, lines int, scan *sessionScan) {
	if lines == 1 {
		if json.Unmarshal([]byte(line), &scan.header) == nil && scan.header.Version == 1 {
			return
		}
	}
	var entry sessionEntry
	if json.Unmarshal([]byte(line), &entry) == nil {
		scan.entries = append(scan.entries, entry)
	}
}

func scanSession(path string) (sessionScan, error) {
	file, err := os.Open(path)
	if err != nil {
		return sessionScan{}, err
	}
	defer func() { _ = file.Close() }()

	var scan sessionScan
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		scan.lines++
		parseSessionLine(line, scan.lines, &scan)
	}
	return scan, scanner.Err()
}

func updateStatus(who, text, status string, explicit *string) {
	if who == "status" {
		if status != "" {
			*explicit = status
		} else if text != "" {
			*explicit = text
		}
	}
}

func inspectEntries(entries []sessionEntry) (string, string, string) {
	var lastMsg, lastWho, explicit string
	for _, e := range entries {
		lastWho = e.Who
		if (e.Who == "question" || e.Who == "answer") && e.Text != "" {
			lastMsg = e.Text
		}
		updateStatus(e.Who, e.Text, e.Status, &explicit)
	}
	return lastMsg, lastWho, explicit
}

func computeStatus(active bool, explicit, lastWho string) string {
	if active {
		switch {
		case explicit == StatusFailed || explicit == StatusCompleted:
			return explicit
		case lastWho == "question" || lastWho == "tool":
			return StatusRunning
		default:
			return StatusIdle
		}
	}
	if explicit != "" {
		return explicit
	}
	return StatusCompleted
}
