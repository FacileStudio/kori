package sessions

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ResolveSession turns a --resume value into a session file path.
func ResolveSession(value string) string {
	if value == "" {
		return ""
	}
	if info, err := os.Stat(value); err == nil && !info.IsDir() {
		return value
	}
	all := ListSessionFiles("")
	trimmed := strings.TrimSuffix(value, ".jsonl")
	if path := matchExactOrSuffix(all, value, trimmed); path != "" {
		return path
	}
	if path := matchPrefix(all, trimmed); path != "" {
		return path
	}
	return matchPID(all, trimmed)
}

func matchExactOrSuffix(all []string, value, trimmed string) string {
	for _, p := range all {
		name := filepath.Base(p)
		base := strings.TrimSuffix(name, ".jsonl")
		if name == value || name == trimmed || base == trimmed || strings.HasSuffix(base, "-"+trimmed) {
			return p
		}
	}
	return ""
}

func matchPrefix(all []string, trimmed string) string {
	for _, p := range all {
		base := strings.TrimSuffix(filepath.Base(p), ".jsonl")
		if strings.HasPrefix(base, trimmed) {
			return p
		}
	}
	return ""
}

func matchPID(all []string, trimmed string) string {
	pid, err := strconv.Atoi(trimmed)
	if err != nil || pid <= 0 {
		return ""
	}
	for _, p := range all {
		if info, err := parseSessionInfo(p); err == nil && info.PID == pid {
			return p
		}
	}
	return ""
}
