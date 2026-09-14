package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

var shellElevationCommands = map[string]struct{}{
	"sudo":       {},
	"doas":       {},
	"su":         {},
	"pkexec":     {},
	"docker":     {},
	"nsenter":    {},
	"unshare":    {},
	"chroot":     {},
	"setpriv":    {},
	"runuser":    {},
	"machinectl": {},
	"ksu":        {},
}

func shellCheckElevation(command string) error {
	for _, segment := range shellCommandSegments(command) {
		if err := checkSingleCmd(shellSegmentExecutable(segment)); err != nil {
			return err
		}
	}
	return nil
}

func checkSingleCmd(cmd string) error {
	if cmd == "" {
		return nil
	}
	if shellElevates(cmd) {
		return fmt.Errorf("%q attempts privilege elevation and is blocked (security.deny_elevation): you lack permission to raise privileges; do not retry this command — report the blocker in your result and finish", cmd)
	}
	if shellSetuidRoot(cmd) {
		return fmt.Errorf("%q attempts privilege elevation (setuid root) and is blocked (security.deny_elevation): you lack permission to raise privileges; do not retry this command — report the blocker in your result and finish", cmd)
	}
	return nil
}

func shellCommandSegments(command string) []string {
	var segments []string
	var cur strings.Builder
	inQuote := rune(0)
	for _, r := range command {
		prevQuote := inQuote
		inQuote = nextQuote(r, inQuote)
		if prevQuote != 0 || inQuote != 0 || !isSeparator(r) {
			cur.WriteRune(r)
			continue
		}
		if s := strings.TrimSpace(cur.String()); s != "" {
			segments = append(segments, s)
		}
		cur.Reset()
	}
	if s := strings.TrimSpace(cur.String()); s != "" {
		segments = append(segments, s)
	}
	return segments
}

func nextQuote(r, inQuote rune) rune {
	switch {
	case inQuote != 0 && r == inQuote:
		return 0
	case inQuote == 0 && (r == '\'' || r == '"'):
		return r
	default:
		return inQuote
	}
}

func isSeparator(r rune) bool {
	return r == ';' || r == '&' || r == '|' || r == '\n' || r == '(' || r == ')' || r == '`'
}

func shellSegmentExecutable(segment string) string {
	sawEnv := false
	for f := range strings.FieldsSeq(segment) {
		clean := strings.Trim(f, `"'\()`)
		if clean == "" || (strings.Contains(clean, "=") && !strings.HasPrefix(clean, "/")) {
			continue
		}
		if clean == "env" || (sawEnv && strings.HasPrefix(clean, "-")) {
			sawEnv = true
			continue
		}
		return clean
	}
	return ""
}

func shellElevates(cmd string) bool {
	base := filepath.Base(cmd)
	base = strings.Trim(base, `"'\`)
	_, ok := shellElevationCommands[base]
	return ok
}

func shellSetuidRoot(cmd string) bool {
	clean := strings.Trim(cmd, `"'\`)
	path := clean
	if !strings.Contains(clean, "/") {
		found, err := exec.LookPath(clean)
		if err != nil {
			return false
		}
		path = found
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	sys, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.Mode()&os.ModeSetuid != 0 && sys.Uid == 0
}
