package sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

func stamped() string {
	return time.Now().Format(time.RFC3339Nano)
}

func appendLine(path string, line []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	if _, err := file.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

// MarkSessionStatus appends a status event line to the session log.
func MarkSessionStatus(path string, status string) error {
	if path == "" {
		return fmt.Errorf("empty session path")
	}
	resolved := ResolveSession(path)
	if resolved != "" {
		path = resolved
	}
	entry := sessionEntry{
		At:     stamped(),
		Who:    "status",
		Status: status,
		Text:   status,
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return appendLine(path, line)
}

// OpenResumeSession opens an existing session log for appending without rewriting the header.
func OpenResumeSession(path, backend, model, root string) *SessionLog {
	if path == "" {
		return nil
	}
	resolved := ResolveSession(path)
	if resolved != "" {
		path = resolved
	}
	info, err := os.Stat(path)
	var size int64
	if err == nil {
		size = info.Size()
	}
	return &SessionLog{
		path:     path,
		lastSize: size,
		backend:  backend,
		model:    model,
		root:     root,
	}
}
