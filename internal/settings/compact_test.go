package settings_test

import (
	"os"
	"path/filepath"
	"testing"

	s "github.com/FacileStudio/kori/internal/settings"
)

type testConfigEnv struct {
	file string
}

func setupConfigEnv(t *testing.T) testConfigEnv {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return testConfigEnv{file: filepath.Join(home, s.ConfigFile)}
}

func (e testConfigEnv) write(t *testing.T, body string) {
	t.Helper()
	if err := os.WriteFile(e.file, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func (e testConfigEnv) read(t *testing.T, over s.Config) s.Config {
	t.Helper()
	c, err := s.Settings("", over)
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	return c
}

func TestCompactAtDefaults(t *testing.T) {
	defaults := s.Defaults("")
	if *defaults.CompactAt != s.DefaultCompactAt {
		t.Fatalf("default compact_at = %d, want %d", *defaults.CompactAt, s.DefaultCompactAt)
	}
	env := setupConfigEnv(t)
	c := env.read(t, s.Config{})
	if *c.CompactAt != s.DefaultCompactAt {
		t.Errorf("default compact_at = %d, want %d", *c.CompactAt, s.DefaultCompactAt)
	}
}

func TestCompactAtPrecedence(t *testing.T) {
	env := setupConfigEnv(t)
	env.write(t, "limits:\n  compact_at: 204800\n")
	if c := env.read(t, s.Config{}); *c.CompactAt != 204800 {
		t.Errorf("file compact_at = %d, want 204800", *c.CompactAt)
	}

	t.Setenv("KORI_COMPACT_AT", "300000")
	if c := env.read(t, s.Config{}); *c.CompactAt != 300000 {
		t.Errorf("env compact_at = %d, want 300000", *c.CompactAt)
	}

	c := env.read(t, s.Config{Limits: s.Limits{CompactAt: new(int64(400000))}})
	if *c.CompactAt != 400000 {
		t.Errorf("flag compact_at = %d, want 400000", *c.CompactAt)
	}
}

func TestCompactAtFileVariants(t *testing.T) {
	env := setupConfigEnv(t)
	env.write(t, "provider:\n  backend: anthropic\n")
	if c := env.read(t, s.Config{}); *c.CompactAt != s.DefaultCompactAt {
		t.Errorf("unmentioned compact_at = %d, want default %d", *c.CompactAt, s.DefaultCompactAt)
	}

	env.write(t, "limits:\n  compact_at: 0\n")
	if c := env.read(t, s.Config{}); *c.CompactAt != 0 {
		t.Errorf("compact_at: 0 = %d, want 0", *c.CompactAt)
	}
}
