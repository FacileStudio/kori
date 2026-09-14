package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode"
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

// minimalShellEnv is what a command sees when security.env_isolation is on:
// a PATH and a HOME and nothing else, so the process environment — where a
// service keeps the credentials the model is not supposed to read — stays
// out of reach. PATH is inherited rather than dropped because it names
// executables, not credentials.
func minimalShellEnv() []string {
	env := []string{"PATH=" + shellCommandPath()}
	if home := os.Getenv("HOME"); home != "" {
		env = append(env, "HOME="+home)
	}
	return env
}

// shellCommandPath is the launching shell's own PATH with the standard user
// bin directories appended, in case the shell that started kori was
// itself minimal.
func shellCommandPath() string {
	path := os.Getenv("PATH")
	if path == "" {
		path = "/usr/local/bin:/usr/bin:/bin"
	}
	seen := map[string]bool{}
	for _, dir := range filepath.SplitList(path) {
		seen[dir] = true
	}
	for _, dir := range shellUserBins() {
		if !seen[dir] {
			path += string(os.PathListSeparator) + dir
			seen[dir] = true
		}
	}
	return path
}

func shellUserBins() []string {
	home := os.Getenv("HOME")
	if home == "" {
		return nil
	}
	return []string{filepath.Join(home, ".local", "bin"), filepath.Join(home, "bin")}
}

// shellCheckElevation refuses commands that raise privileges, however
// spelled, quoted, chained or hidden behind a path. It is a denylist, and a
// denylist is a brake on the obvious spellings, not a jail.
func shellCheckElevation(command string) error {
	for field := range strings.FieldsSeq(command) {
		if shellElevates(field) {
			return fmt.Errorf("%q attempts privilege elevation and is blocked (security.deny_elevation): you lack permission to raise privileges; do not retry this command — report the blocker in your result and finish", field)
		}
		if shellSetuidRoot(field) {
			return fmt.Errorf("%q attempts privilege elevation (setuid root) and is blocked (security.deny_elevation): you lack permission to raise privileges; do not retry this command — report the blocker in your result and finish", field)
		}
	}
	return nil
}

func shellElevates(field string) bool {
	for _, word := range shellElevationWords(field) {
		if _, ok := shellElevationCommands[filepath.Base(word)]; ok {
			return true
		}
	}
	return false
}

func shellElevationWords(field string) []string {
	return strings.FieldsFunc(field, func(r rune) bool {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			return false
		case r == '_', r == '.', r == '/':
			return false
		default:
			return true
		}
	})
}

func shellSetuidRoot(field string) bool {
	path := field
	if !strings.Contains(field, "/") {
		found, err := exec.LookPath(field)
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
