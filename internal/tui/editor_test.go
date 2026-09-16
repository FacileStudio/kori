package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/FacileStudio/kori/internal/settings"
)

func TestKeyCtrlShiftUTriggersEditorWhenEmpty(t *testing.T) {
	t.Setenv("EDITOR", "cat")
	m := sized()
	m.prompt.SetValue("")
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl | tea.ModShift})
	if cmd == nil {
		t.Fatal("Update with ctrl+shift+u returned nil cmd for empty prompt")
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
