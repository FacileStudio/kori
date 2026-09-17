package provider

import "os"

// CatalogModel represents a curated model option for a provider.
type CatalogModel struct {
	Backend     string
	Model       string
	Description string
}

var curatedModels = []CatalogModel{
	{Backend: "anthropic", Model: "claude-opus-5", Description: "Maximum capability, deep reasoning"},
	{Backend: "anthropic", Model: "claude-sonnet-4-6", Description: "Balanced speed and capability"},
	{Backend: "anthropic", Model: "claude-haiku-4-5", Description: "Fast and lightweight"},
	{Backend: "openai", Model: "gpt-5.4", Description: "Flagship general-purpose model"},
	{Backend: "openai", Model: "o3-mini", Description: "Fast reasoning model"},
	{Backend: "openai", Model: "o1", Description: "High-tier reasoning model"},
	{Backend: "openai", Model: "gpt-4o", Description: "Fast multimodal model"},
	{Backend: "google", Model: "gemini-2.5-pro", Description: "Large context reasoning"},
	{Backend: "google", Model: "gemini-2.5-flash", Description: "Ultra-fast low-latency"},
	{Backend: "openrouter", Model: "anthropic/claude-3.7-sonnet", Description: "Claude via OpenRouter"},
	{Backend: "openrouter", Model: "deepseek/deepseek-r1", Description: "DeepSeek reasoning model"},
	{Backend: "openrouter", Model: "qwen/qwen-2.5-coder-32b-instruct", Description: "Specialized coding model"},
}

// DetectedProviders checks the environment for configured provider API keys.
func DetectedProviders() []string {
	var backends []string
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		backends = append(backends, "anthropic")
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		backends = append(backends, "openai")
	}
	if os.Getenv("GEMINI_API_KEY") != "" || os.Getenv("GOOGLE_API_KEY") != "" {
		backends = append(backends, "google")
	}
	if os.Getenv("OPENROUTER_API_KEY") != "" {
		backends = append(backends, "openrouter")
	}
	if len(backends) == 0 {
		return []string{"anthropic", "google", "openai", "openrouter"}
	}
	return backends
}

// ModelsForBackend returns curated models matching a given backend name.
func ModelsForBackend(backend string) []CatalogModel {
	var list []CatalogModel
	for _, m := range curatedModels {
		if m.Backend == backend {
			list = append(list, m)
		}
	}
	return list
}

// AvailableCatalogModels returns curated models for all detected providers.
func AvailableCatalogModels() []CatalogModel {
	detected := DetectedProviders()
	seen := make(map[string]bool, len(detected))
	for _, b := range detected {
		seen[b] = true
	}
	var list []CatalogModel
	for _, m := range curatedModels {
		if seen[m.Backend] {
			list = append(list, m)
		}
	}
	return list
}
