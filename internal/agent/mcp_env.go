package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/FacileStudio/nacelle/mcp/client"
)

func ensureUserPath() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	current := os.Getenv("PATH")
	toAdd := missingUserDirs(home, current)
	if len(toAdd) == 0 {
		return
	}
	if err := os.Setenv("PATH", strings.Join(toAdd, string(filepath.ListSeparator))+string(filepath.ListSeparator)+current); err != nil {
		return
	}
}

func missingUserDirs(home, current string) []string {
	candidates := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, ".cargo", "bin"),
		filepath.Join(home, ".bun", "bin"),
	}
	var missing []string
	for _, c := range candidates {
		if isDir(c) && !strings.Contains(":"+current+":", ":"+c+":") {
			missing = append(missing, c)
		}
	}
	return missing
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func expandServerDef(def client.ServerDef) client.ServerDef {
	def.Command = expandHome(def.Command)
	if def.Dir != "" {
		def.Dir = expandHome(def.Dir)
	}
	if len(def.Args) > 0 {
		args := make([]string, len(def.Args))
		for i, a := range def.Args {
			args[i] = expandHome(a)
		}
		def.Args = args
	}
	if len(def.Env) > 0 {
		expanded := make(map[string]string, len(def.Env))
		for k, v := range def.Env {
			expanded[k] = expandHome(v)
		}
		def.Env = expanded
	}
	return def
}

func expandPaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p != "" {
			out = append(out, expandHome(p))
		}
	}
	return out
}

func normalizeCallToolArgs(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage("{}")
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		trimmed := strings.TrimSpace(str)
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			return json.RawMessage(trimmed)
		}
	}
	return raw
}
