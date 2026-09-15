package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/FacileStudio/kori/internal/settings"
)

// resolveEditor determines which editor to use for editing a prompt
// externally. It searches in order: the configured editor path, the
// GIT_EDITOR environment variable, the system EDITOR variable, the VISUAL
// variable, and finally "vi" as a last resort. Returns an empty string only
// if every source is silent, which callers treat as "no editor available".
func resolveEditor(cfg settings.Config) string {
	if cfg.Editor.Editor != "" {
		return cfg.Editor.Editor
	}
	if e := os.Getenv("GIT_EDITOR"); e != "" {
		return e
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	return "vi"
}

// editInExternalEditor opens the given content in the user's editor,
// waits for the process to exit, and returns whatever the editor wrote
// back. The content is written to a temporary file, the editor is launched
// with that file as its argument, and the file is read again once the editor
// closes. If the editor cannot be found or launched, or if the temporary file
// cannot be written, the error is returned and the original content is untouched.
func editInExternalEditor(content string, editor string) (string, error) {
	if editor == "" {
		return content, errors.New("no editor configured")
	}

	tmpPath, closeFn, err := createTempFile(content, "*.kori")
	if err != nil {
		return content, err
	}
	defer closeFn()

	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return content, fmt.Errorf("editor exited with code %d", exitErr.ExitCode())
		}
		return content, fmt.Errorf("running editor: %w", err)
	}

	edited, err := os.ReadFile(tmpPath)
	if err != nil {
		return content, fmt.Errorf("reading edited file: %w", err)
	}
	return string(edited), nil
}

// createTempFile writes content to a temporary file and returns the path and
// a cleanup function that removes the file. The cleanup must be deferred by
// the caller.
func createTempFile(content string, suffix string) (string, func(), error) {
	tmpFile, err := os.CreateTemp("", suffix)
	if err != nil {
		return "", func() {}, fmt.Errorf("creating temp file: %w", err)
	}
	path := tmpFile.Name()

	if _, err := tmpFile.WriteString(content); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		cleanupTemp(path)
		return "", func() {}, fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		cleanupTemp(path)
		return "", func() {}, fmt.Errorf("closing temp file: %w", err)
	}

	return path, func() { cleanupTemp(path) }, nil
}

func cleanupTemp(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return
	}
}

// openEditor opens the current prompt content in the user's configured
// external editor and replaces the prompt with the result if it
// changed. Uses resolveEditor to find the editor (GIT_EDITOR > EDITOR
// > VISUAL > vi). Does nothing when there is no editor configured
// or the prompt is empty — the prompt itself is the editor.
func (m *Model) openEditor() {
	editor := resolveEditor(settings.Config{Editor: settings.Editor{Editor: m.run.editorPath, PromptEditKey: m.run.promptEditKey}})
	if editor == "" {
		return
	}
	content := m.prompt.Value()
	if content == "" {
		return
	}

	edited, err := editInExternalEditor(content, editor)
	if err != nil {
		m.say(fromReader, "editor failed: "+err.Error())
		return
	}
	if edited != content {
		m.prompt.SetValue(edited)
	}
}
