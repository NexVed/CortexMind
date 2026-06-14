// Package llm provides optional Large Language Model enrichment for CORTEX
// scans. Providers are configured per-user (API keys live on the user record's
// preferences, never in source/config), so the daemon stays usable with no
// external services configured — every call degrades gracefully.
package llm

import "context"

// ProviderConfig is the persisted, per-user LLM/embedding configuration. It is
// stored as JSON under users.preferences["providers"].
type ProviderConfig struct {
	// LLM enrichment / prompt generation.
	LLMProvider string `json:"llm_provider"` // "mistral" | "ollama" | "none"
	MistralKey  string `json:"mistral_key"`
	MistralModel string `json:"mistral_model"`
	OllamaChatModel string `json:"ollama_chat_model"`

	// Embeddings (memory construction).
	Embedder       string `json:"embedder"` // "ollama" | "mistral" | "none"
	OllamaURL      string `json:"ollama_url"`
	OllamaModel    string `json:"ollama_model"`
	MistralEmbModel string `json:"mistral_emb_model"`
}

// Defaults fills empty fields with sensible defaults.
func (c *ProviderConfig) Defaults() {
	if c.MistralModel == "" {
		c.MistralModel = "mistral-small-latest"
	}
	if c.OllamaChatModel == "" {
		c.OllamaChatModel = "llama3.1"
	}
	if c.OllamaURL == "" {
		c.OllamaURL = "http://localhost:11434"
	}
	if c.OllamaModel == "" {
		c.OllamaModel = "bge-m3"
	}
	if c.MistralEmbModel == "" {
		c.MistralEmbModel = "mistral-embed"
	}
	if c.LLMProvider == "" {
		c.LLMProvider = "none"
	}
	if c.Embedder == "" {
		c.Embedder = "none"
	}
}

// Redacted returns a copy with secrets masked, safe to return to the UI.
func (c ProviderConfig) Redacted() ProviderConfig {
	c.MistralKey = mask(c.MistralKey)
	return c
}

func mask(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "••••"
	}
	return "••••" + s[len(s)-4:]
}

// Client is the high level enrichment surface used by the analyzer.
type Client interface {
	// Summarize returns a short natural-language description from a prompt.
	Summarize(ctx context.Context, system, user string) (string, error)
	// Enabled reports whether the client can actually reach a provider.
	Enabled() bool
}

// New builds a Client for the configured LLM provider. It always returns a
// non-nil Client; when no provider is configured it returns a no-op client.
func New(cfg ProviderConfig) Client {
	switch cfg.LLMProvider {
	case "mistral":
		if cfg.MistralKey != "" {
			return NewMistral(cfg.MistralKey, cfg.MistralModel)
		}
	case "ollama":
		return NewOllama(cfg.OllamaURL, cfg.OllamaChatModel)
	}
	return noopClient{}
}

type noopClient struct{}

func (noopClient) Summarize(context.Context, string, string) (string, error) { return "", nil }
func (noopClient) Enabled() bool                                             { return false }
