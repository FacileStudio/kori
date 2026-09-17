package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProfilesFromDirectory(t *testing.T) {
	dir := t.TempDir()
	fastYml := `name: fast
provider:
  backend: google
  model: gemini-2.5-flash
reasoning:
  effort: low
`
	smartYml := `provider:
  backend: anthropic
  model: claude-opus-5
`
	if err := os.WriteFile(filepath.Join(dir, "fast.yml"), []byte(fastYml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "smart.yml"), []byte(smartYml), 0o644); err != nil {
		t.Fatal(err)
	}

	profiles, err := LoadProfiles(dir)
	if err != nil {
		t.Fatalf("LoadProfiles failed: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("len(profiles) = %d, want 2", len(profiles))
	}
	if profiles[0].Name != "fast" || profiles[0].Provider.Backend != "google" {
		t.Errorf("profiles[0] = %+v, want fast/google", profiles[0])
	}
	if profiles[1].Name != "smart" || profiles[1].Provider.Backend != "anthropic" {
		t.Errorf("profiles[1] = %+v, want smart/anthropic", profiles[1])
	}
}

func TestLoadProfilesDuplicateName(t *testing.T) {
	dir := t.TempDir()
	p1 := "name: dup\nprovider:\n  backend: openai\n"
	p2 := "name: dup\nprovider:\n  backend: google\n"
	if err := os.WriteFile(filepath.Join(dir, "a.yml"), []byte(p1), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.yml"), []byte(p2), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadProfiles(dir)
	if err == nil {
		t.Fatal("expected error on duplicate profile name, got nil")
	}
}

func TestApplyProfile(t *testing.T) {
	thinking := true
	budget := int64(4096)
	p := Profile{
		Name: "test",
		Provider: Provider{
			Backend: "openrouter",
			Model:   "deepseek/deepseek-r1",
			BaseURL: "https://openrouter.ai/api/v1",
		},
		Reasoning: Reasoning{
			Effort:   "high",
			Thinking: &thinking,
			Budget:   &budget,
		},
	}

	var c Config
	ApplyProfile(&c, p)

	if c.Backend != "openrouter" || c.Model != "deepseek/deepseek-r1" {
		t.Errorf("ApplyProfile didn't set backend/model: %+v", c)
	}
	if c.Effort != "high" || *c.Budget != 4096 {
		t.Errorf("ApplyProfile didn't set reasoning: %+v", c)
	}
}

func setupProfileEnv(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KORI_BACKEND", "")
	t.Setenv("NACELLE_BACKEND", "")
	t.Setenv("KORI_PROFILE", "")
	t.Setenv("NACELLE_PROFILE", "")
	pDir := filepath.Join(home, ".kori", "profiles")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fastYml := "name: fast\nprovider:\n  backend: google\n  model: gemini-2.5-flash\n"
	if err := os.WriteFile(filepath.Join(pDir, "fast.yml"), []byte(fastYml), 0o644); err != nil {
		t.Fatal(err)
	}
	fileYml := "provider:\n  backend: anthropic\n"
	if err := os.WriteFile(filepath.Join(home, ConfigFile), []byte(fileYml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFlagProfileOverridesFileBackend(t *testing.T) {
	setupProfileEnv(t)
	cfg, err := Settings("", Config{Profile: "fast"})
	if err != nil {
		t.Fatalf("Settings with flag profile failed: %v", err)
	}
	if cfg.Backend != "google" {
		t.Errorf("flags profile backend = %q, want google", cfg.Backend)
	}
	if cfg.Model != "gemini-2.5-flash" {
		t.Errorf("flags profile model = %q, want gemini-2.5-flash", cfg.Model)
	}
}

func TestEnvProfileOverridesFileBackend(t *testing.T) {
	setupProfileEnv(t)
	t.Setenv("KORI_PROFILE", "fast")
	cfg, err := Settings("", Config{})
	if err != nil {
		t.Fatalf("Settings with env profile failed: %v", err)
	}
	if cfg.Backend != "google" {
		t.Errorf("env profile backend = %q, want google", cfg.Backend)
	}
	if cfg.Model != "gemini-2.5-flash" {
		t.Errorf("env profile model = %q, want gemini-2.5-flash", cfg.Model)
	}
}
