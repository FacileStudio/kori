package agent

import (
	"os"
	"path/filepath"
)

func minimalShellEnv() []string {
	env := []string{"PATH=" + shellCommandPath()}
	if home := os.Getenv("HOME"); home != "" {
		env = append(env, "HOME="+home)
	}
	return env
}

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
