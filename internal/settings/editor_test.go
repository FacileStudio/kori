package settings

import (
	"testing"
)

func TestEditorDefaultsToEmpty(t *testing.T) {
	written(t, "")
	config, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if config.Editor.Editor != "" {
		t.Errorf("editor = %q, want empty by default", config.Editor.Editor)
	}
	if config.PromptEditKey != "" {
		t.Errorf("prompt_edit_key = %q, want empty by default", config.PromptEditKey)
	}
}

func TestEditorComesFromTheFile(t *testing.T) {
	written(t, "editor:\n  editor: /usr/bin/nano\n  prompt_edit_key: message")

	config, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if config.Editor.Editor != "/usr/bin/nano" {
		t.Errorf("editor = %q, want the file's value", config.Editor.Editor)
	}
	if config.PromptEditKey != "message" {
		t.Errorf("prompt_edit_key = %q, want the file's value", config.PromptEditKey)
	}
}

func TestEditorFileBeatsDefaults(t *testing.T) {
	written(t, "editor:\n  editor: /opt/homebrew/bin/nvim")

	config, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if config.Editor.Editor != "/opt/homebrew/bin/nvim" {
		t.Errorf("editor = %q, want the file's value over defaults", config.Editor.Editor)
	}
}

func TestEditorEnvironmentVariable(t *testing.T) {
	written(t, "")
	t.Setenv(EnvPrefix+"EDITOR", "/usr/local/bin/vim")

	config, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if config.Editor.Editor != "/usr/local/bin/vim" {
		t.Errorf("editor = %q, want the env to win", config.Editor.Editor)
	}
}

func TestEditorFlagBeatsEnvAndFile(t *testing.T) {
	written(t, "editor:\n  editor: /usr/bin/nano")
	t.Setenv(EnvPrefix+"EDITOR", "/usr/local/bin/vim")

	flagEditor := "ed"
	config, err := settings(Config{Editor: Editor{Editor: flagEditor}})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if config.Editor.Editor != flagEditor {
		t.Errorf("editor = %q, want the flag to win over env and file", config.Editor.Editor)
	}
}

func TestEditorFlagOverridesEnv(t *testing.T) {
	written(t, "")
	t.Setenv(EnvPrefix+"EDITOR", "/usr/local/bin/vim")

	flagEditor := "ed"
	config, err := settings(Config{Editor: Editor{Editor: flagEditor}})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if config.Editor.Editor != flagEditor {
		t.Errorf("editor = %q, want the flag to win over env", config.Editor.Editor)
	}
}

func TestEditorFlagDoesNotOverrideFile(t *testing.T) {
	written(t, "editor:\n  editor: /usr/bin/nano")
	t.Setenv(EnvPrefix+"EDITOR", "/usr/local/bin/vim")

	flagEditor := "ed"
	config, err := settings(Config{Editor: Editor{Editor: flagEditor}})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if config.Editor.Editor != flagEditor {
		t.Errorf("editor = %q, want the flag to win", config.Editor.Editor)
	}
}

func TestEditorEmptyEnvDoesNotOverrideFile(t *testing.T) {
	written(t, "editor:\n  editor: /usr/bin/nano")
	t.Setenv(EnvPrefix+"EDITOR", "")

	config, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if config.Editor.Editor != "/usr/bin/nano" {
		t.Errorf("editor = %q, want the file's value when env is empty", config.Editor.Editor)
	}
}

func TestEditorPartialConfigFromFile(t *testing.T) {
	written(t, "editor:\n  editor: /usr/bin/nano")

	config, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if config.Editor.Editor != "/usr/bin/nano" {
		t.Errorf("editor = %q, want the file's editor", config.Editor.Editor)
	}
	if config.PromptEditKey != "" {
		t.Errorf("prompt_edit_key = %q, want empty when not set in file", config.PromptEditKey)
	}
}

func TestEditorPromptEditKeyFromEnv(t *testing.T) {
	written(t, "")
	t.Setenv(EnvPrefix+"PROMPT_EDIT_KEY", "content")

	config, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if config.PromptEditKey != "content" {
		t.Errorf("prompt_edit_key = %q, want the env to win", config.PromptEditKey)
	}
}

func TestEditorMergePreservesIndividualFields(t *testing.T) {
	written(t, "editor:\n  editor: /usr/bin/nano")
	t.Setenv(EnvPrefix+"PROMPT_EDIT_KEY", "custom_key")

	config, err := settings(Config{})
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if config.Editor.Editor != "/usr/bin/nano" {
		t.Errorf("editor = %q, want file value preserved", config.Editor.Editor)
	}
	if config.PromptEditKey != "custom_key" {
		t.Errorf("prompt_edit_key = %q, want env value merged", config.PromptEditKey)
	}
}
