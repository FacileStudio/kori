package provider

import (
	"testing"
)

func TestNewAnthropic(t *testing.T) {
	b, err := New(Config{Backend: "anthropic", Model: "claude-opus-5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b == nil {
		t.Fatal("expected non-nil backend")
	}
}

func TestNewAnthropicRejectsEndpoint(t *testing.T) {
	_, err := New(Config{Backend: "anthropic", BaseURL: "http://localhost:8080"})
	if err == nil {
		t.Fatal("expected error when baseURL is passed to anthropic")
	}
}

func TestNewOpenRouterRequiresModel(t *testing.T) {
	_, err := New(Config{Backend: "openrouter"})
	if err == nil {
		t.Fatal("expected error when model is missing for openrouter")
	}
}

func TestNewUnknownBackend(t *testing.T) {
	_, err := New(Config{Backend: "unknown"})
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}
