package agent

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/FacileStudio/kori/internal/settings"
)

func readCrontab() (string, error) {
	cmd := exec.Command("crontab", "-l")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return stdout.String(), nil
	}
	errOut := strings.ToLower(stderr.String() + stdout.String())
	if strings.Contains(errOut, "no crontab") {
		return "", nil
	}
	if errors.Is(err, exec.ErrNotFound) {
		return "", fmt.Errorf("crontab executable not found in PATH: %w", err)
	}
	return "", fmt.Errorf("reading crontab: %w: %s", err, strings.TrimSpace(stderr.String()))
}

func writeCrontab(content string) error {
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("crontab executable not found in PATH: %w", err)
		}
		return fmt.Errorf("writing crontab: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func applyCrontabBlock(current, name, newBlock string) string {
	stripped, _ := stripCrontabBlock(current, name)
	trimmed := strings.TrimSpace(stripped)
	if trimmed == "" {
		return newBlock + "\n"
	}
	return trimmed + "\n\n" + newBlock + "\n"
}

func stripCrontabBlock(current, name string) (string, bool) {
	beginMarker := "# BEGIN KORI JOB " + name
	endMarker := "# END KORI JOB " + name
	legacyMatch := "cron run " + name
	lines := strings.Split(current, "\n")
	var result []string
	inBlock := false
	removed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == beginMarker {
			inBlock = true
			removed = true
			continue
		}
		if inBlock {
			if trimmed == endMarker {
				inBlock = false
			}
			continue
		}
		if strings.Contains(line, legacyMatch) {
			removed = true
			continue
		}
		result = append(result, line)
	}
	return cleanCrontabLines(result), removed
}

func cleanCrontabLines(lines []string) string {
	var cleaned []string
	lastEmpty := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if !lastEmpty && len(cleaned) > 0 {
				cleaned = append(cleaned, "")
				lastEmpty = true
			}
			continue
		}
		cleaned = append(cleaned, line)
		lastEmpty = false
	}
	out := strings.Join(cleaned, "\n")
	out = strings.TrimSpace(out)
	if out == "" {
		return ""
	}
	return out + "\n"
}

func findJobFile(files []settings.JobFile, name string) (settings.JobFile, error) {
	for _, f := range files {
		if f.Job.Name == name {
			return f, nil
		}
	}
	return settings.JobFile{}, fmt.Errorf("no cron job named %q", name)
}

func ensureTrusted(f settings.JobFile) error {
	ok, err := settings.IsTrusted(f.Path, f.Raw)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return fmt.Errorf("job %q is not trusted: review %s, then run `kori cron trust %s`", f.Job.Name, f.Path, f.Job.Name)
}

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
