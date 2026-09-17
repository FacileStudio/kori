// Package provider constructs model backends and inspects available catalogs.
package provider

import (
	"fmt"

	"github.com/FacileStudio/nacelle"
	"github.com/FacileStudio/nacelle/anthropic"
	"github.com/FacileStudio/nacelle/google"
	"github.com/FacileStudio/nacelle/openai"
	"github.com/FacileStudio/nacelle/openrouter"
)

// Config holds connection parameters for building a backend.
type Config struct {
	Backend string
	Model   string
	BaseURL string
	APIKey  string
}

// New constructs the requested nacelle.Backend.
func New(cfg Config) (nacelle.Backend, error) {
	switch cfg.Backend {
	case "anthropic":
		if cfg.BaseURL != "" || cfg.APIKey != "" {
			return nil, fmt.Errorf("anthropic takes no custom endpoint: base_url and api_key apply only to google, openai, or openrouter")
		}
		return anthropic.New(anthropic.Config{Model: cfg.Model}), nil
	case "google":
		return google.New(google.Config{
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
			APIKey:  cfg.APIKey,
		})
	case "openai":
		return openai.New(openai.Config{
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
			APIKey:  cfg.APIKey,
		})
	case "openrouter":
		if cfg.Model == "" {
			return nil, fmt.Errorf("openrouter needs a model")
		}
		return openrouter.New(openrouter.Config{
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
			APIKey:  cfg.APIKey,
		})
	default:
		return nil, fmt.Errorf("unknown backend %q, want anthropic, google, openai, or openrouter", cfg.Backend)
	}
}
