package sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/FacileStudio/kori/internal/settings"
	"sort"
	"strings"

	"github.com/FacileStudio/nacelle"
)

// ListSessionFiles returns a slice of session file paths sorted by modification time (newest first)
// for the given project root. If projectRoot is empty, it lists all sessions.
// If the sessions directory doesn't exist, it returns an empty slice.
func ListSessionFiles(projectRoot string) []string {
	sessionsDir := projectSessionsDir(projectRoot)
	if sessionsDir == "" {
		return nil
	}

	files, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil
	}

	var sessionFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".jsonl") {
			sessionFiles = append(sessionFiles, filepath.Join(sessionsDir, f.Name()))
		}
	}
	sortSessionsByMtime(sessionFiles)
	return sessionFiles
}

func projectSessionsDir(projectRoot string) string {
	base, err := settings.HomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(base, "sessions")
	if projectRoot == "" {
		return dir
	}
	clean := filepath.Clean(projectRoot)
	switch clean {
	case ".":
		return dir
	case "..":
		return filepath.Join(dir, filepath.Base(clean))
	default:
		candidate := filepath.Join(dir, clean)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		return dir
	}
}

func sortSessionsByMtime(files []string) {
	sort.Slice(files, func(i, j int) bool {
		infoI, errI := os.Stat(files[i])
		infoJ, errJ := os.Stat(files[j])
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})
}

// LoadSession loads and parses a session file, returning the conversation.
// It returns nil if the file cannot be read or parsed.
func LoadSession(path string) []nacelle.Message {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) > 0 && hasSessionHeader(lines[0]) {
		lines = lines[1:]
	}

	var conversation []nacelle.Message
	for _, line := range lines {
		if msg, ok := parseSessionEntry(line); ok {
			conversation = append(conversation, msg)
		}
	}
	return conversation
}

func hasSessionHeader(firstLine string) bool {
	var header sessionHeader
	return json.Unmarshal([]byte(firstLine), &header) == nil && header.Version == 1
}

func parseSessionEntry(line string) (nacelle.Message, bool) {
	if line == "" {
		return nacelle.Message{}, false
	}
	var entry sessionEntry
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		return nacelle.Message{}, false
	}
	switch entry.Who {
	case "question":
		return nacelle.UserText(entry.Text), true
	case "answer":
		return nacelle.AssistantText(entry.Text), true
	default:
		return nacelle.Message{}, false
	}
}

// ResolveSession turns a --resume value into a session file path. A value
// that names an existing file is used as-is; anything else is matched against
// every session on disk by file name, so a bare id like
// 2026-09-10T14:03:00Z-1234 (with or without the .jsonl suffix) works from
// any directory. Returns "" when nothing matches.
func ResolveSession(value string) string {
	if value == "" {
		return ""
	}
	if info, err := os.Stat(value); err == nil && !info.IsDir() {
		return value
	}
	value = strings.TrimSuffix(value, ".jsonl")
	for _, p := range ListSessionFiles("") {
		if filepath.Base(p) == value || filepath.Base(p) == value+".jsonl" {
			return p
		}
	}
	return ""
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
		return fmt.Sprintf("  %s · %s",
			filepath.Base(filePath),
			info.ModTime().Format("2006-01-02 15:04:05"))
	}

	var header sessionHeader
	if err := json.Unmarshal([]byte(linesData[0]), &header); err == nil && header.Version == 1 {
		started := header.Started
		if len(started) > 19 {
			started = started[:19]
		}
		return fmt.Sprintf("  %s · %s · %s · %s",
			filepath.Base(filePath),
			header.Backend,
			header.Model,
			started)
	}

	return fmt.Sprintf("  %s · %s",
		filepath.Base(filePath),
		info.ModTime().Format("2006-01-02 15:04:05"))
}
