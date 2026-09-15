package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/FacileStudio/kori/internal/settings"
)

func TestOpenEditorWhenPromptIsEmpty(t *testing.T) {
	t.Setenv("EDITOR", "cat")
	m := sized()
	m.prompt.SetValue("")
	cmd := m.openEditor()
	if cmd == nil {
		t.Fatal("openEditor returned nil for empty prompt, expected tea.Cmd")
	}
}

func TestOpenEditorWhenPromptHasContent(t *testing.T) {
	t.Setenv("EDITOR", "cat")
	m := sized()
	m.prompt.SetValue("some draft text")
	cmd := m.openEditor()
	if cmd == nil {
		t.Fatal("openEditor returned nil for non-empty prompt, expected tea.Cmd")
	}
}

func TestKeyCtrlETriggersEditorWhenEmpty(t *testing.T) {
	t.Setenv("EDITOR", "cat")
	m := sized()
	m.prompt.SetValue("")
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("Update with ctrl+e returned nil cmd for empty prompt")
	}
}

func TestFinishEditorSetsPromptValueAndTrimsNewline(t *testing.T) {
	m := sized()
	m.prompt.SetValue("original")

	tmpFile := filepath.Join(t.TempDir(), "test-prompt.kori")
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
