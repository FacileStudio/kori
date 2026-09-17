package provider

import (
	"testing"
)

func TestDetectedProvidersFallbacks(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")

	all := DetectedProviders()
	if len(all) != 4 {
		t.Errorf("expected 4 fallback providers, got %d", len(all))
	}

	t.Setenv("ANTHROPIC_API_KEY", "sk-test")
	detected := DetectedProviders()
	if len(detected) != 1 || detected[0] != "anthropic" {
		t.Errorf("expected [anthropic], got %v", detected)
	}
}

func TestModelsForBackend(t *testing.T) {
	anthropicModels := ModelsForBackend("anthropic")
	if len(anthropicModels) == 0 {
		t.Fatal("expected curated models for anthropic")
	}
	for _, m := range anthropicModels {
		if m.Backend != "anthropic" {
			t.Errorf("wrong backend: %s", m.Backend)
		}
	}
}

func TestAvailableCatalogModels(t *testing.T) {
	models := AvailableCatalogModels()
	if len(models) == 0 {
		t.Fatal("expected non-empty available catalog models")
	}
}
