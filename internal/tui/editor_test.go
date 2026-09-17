package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/FacileStudio/kori/internal/settings"
)

func TestOpenEditorCreatesMarkdownFileWithPromptContent(t *testing.T) {
	t.Setenv("EDITOR", "cat")
	tDir := t.TempDir()
	t.Setenv("TMPDIR", tDir)

	m := sized()
	expected := "prompt text for editor"
	m.prompt.SetValue(expected)

	if cmd := m.openEditor(); cmd == nil {
		t.Fatal("openEditor returned nil cmd")
	}

	matches, err := filepath.Glob(filepath.Join(tDir, "kori-prompt-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("expected 1 created file, got %v (err: %v)", matches, err)
	}
	file := matches[0]

	if !strings.HasSuffix(file, ".md") {
		t.Errorf("created file %q does not end with .md", file)
	}
	data, err := os.ReadFile(file)
	if err != nil || string(data) != expected {
		t.Errorf("ReadFile = %q, %v, want %q", string(data), err, expected)
	}
}

func TestKeyCtrlOAndAltETriggersEditor(t *testing.T) {
	t.Setenv("EDITOR", "cat")
	m := sized()
	m.prompt.SetValue("draft")

	testCases := []struct {
		name string
		msg  tea.KeyPressMsg
	}{
		{name: "ctrl+o key", msg: tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl}},
		{name: "ctrl+O key", msg: tea.KeyPressMsg{Code: 'O', Mod: tea.ModCtrl}},
		{name: "ctrl+o text", msg: tea.KeyPressMsg{Text: "ctrl+o"}},
		{name: "alt+e text", msg: tea.KeyPressMsg{Text: "alt+e"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handled, cmd := m.key(tc.msg)
			if !handled {
				t.Errorf("%s was not handled", tc.name)
			}
			if cmd == nil {
				t.Errorf("%s returned nil cmd", tc.name)
			}
		})
	}
}

func TestCustomPromptEditKeyTriggersEditor(t *testing.T) {
	t.Setenv("EDITOR", "cat")
	m := sized()
	m.run.promptEditKey = "ctrl+g"
	m.prompt.SetValue("draft")

	handled, cmd := m.key(tea.KeyPressMsg{Text: "ctrl+g"})
	if !handled || cmd == nil {
		t.Errorf("custom promptEditKey ctrl+g was not handled")
	}
}

func TestFinishEditorSetsPromptValueAndTrimsNewline(t *testing.T) {
	m := sized()
	m.prompt.SetValue("original")

	tmpFile := filepath.Join(t.TempDir(), "test-prompt.md")
	if err := os.WriteFile(tmpFile, []byte("edited from external\n"), 0o600); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	cleaned := false
	cleanup := func() {
		cleaned = true
		if err := os.Remove(tmpFile); err != nil && !os.IsNotExist(err) {
			t.Errorf("cleanup remove: %v", err)
		}
	}

	m.route(editorFinishedMsg{
		path:    tmpFile,
		cleanup: cleanup,
		err:     nil,
	})

	if !cleaned {
		t.Error("cleanup function was not called")
	}
	if got := m.prompt.Value(); got != "edited from external" {
		t.Errorf("prompt.Value() = %q, want %q", got, "edited from external")
	}
}

func TestFinishEditorHandlesError(t *testing.T) {
	m := sized()
	m.prompt.SetValue("original content")

	cleaned := false
	cleanup := func() {
		cleaned = true
	}

	m.route(editorFinishedMsg{
		path:    "/nonexistent/path",
		cleanup: cleanup,
		err:     errors.New("editor crashed"),
	})

	if !cleaned {
		t.Error("cleanup function was not called")
	}
	if got := m.prompt.Value(); got != "original content" {
		t.Errorf("prompt.Value() = %q, want original content", got)
	}
}

func TestResolveEditorPriority(t *testing.T) {
	t.Setenv("GIT_EDITOR", "git-ed")
	t.Setenv("EDITOR", "ed")
	t.Setenv("VISUAL", "vis")
	if r := resolveEditor(settings.Config{}); r != "git-ed" {
		t.Errorf("resolveEditor = %q, want git-ed", r)
	}
}
