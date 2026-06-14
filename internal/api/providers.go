// Package api exposes CORTEX's higher-level HTTP endpoints (GitHub-wide repo
// scanning, LLM provider configuration and knowledge-graph retrieval) as plain
// JSON routes mounted on the PocketBase router. These complement the
// ConnectRPC services and are consumed by the UI via fetch.
package api

import (
	"github.com/NexVed/Cortex/internal/llm"
	"github.com/pocketbase/pocketbase/core"
)

const prefProvidersKey = "providers"

// LoadProviderConfig reads the per-user LLM/embedding configuration from the
// user record's preferences JSON, applying defaults.
func LoadProviderConfig(user *core.Record) llm.ProviderConfig {
	cfg := llm.ProviderConfig{}
	if user != nil {
		var prefs map[string]any
		if err := user.UnmarshalJSONField("preferences", &prefs); err == nil {
			if raw, ok := prefs[prefProvidersKey].(map[string]any); ok {
				assignProviderConfig(&cfg, raw)
			}
		}
	}
	cfg.Defaults()
	return cfg
}

// SaveProviderConfig persists the provider configuration onto the user record,
// preserving any other preference keys. Empty secret fields are left untouched
// so the UI can submit redacted forms without wiping stored keys.
func SaveProviderConfig(app core.App, user *core.Record, incoming llm.ProviderConfig) error {
	existing := LoadProviderConfig(user)

	// Preserve secrets when the incoming value is blank or redacted.
	if incoming.MistralKey == "" || isRedacted(incoming.MistralKey) {
		incoming.MistralKey = existing.MistralKey
	}
	incoming.Defaults()

	var prefs map[string]any
	if err := user.UnmarshalJSONField("preferences", &prefs); err != nil || prefs == nil {
		prefs = map[string]any{}
	}
	prefs[prefProvidersKey] = map[string]any{
		"llm_provider":      incoming.LLMProvider,
		"mistral_key":       incoming.MistralKey,
		"mistral_model":     incoming.MistralModel,
		"ollama_chat_model": incoming.OllamaChatModel,
		"embedder":          incoming.Embedder,
		"ollama_url":        incoming.OllamaURL,
		"ollama_model":      incoming.OllamaModel,
		"mistral_emb_model": incoming.MistralEmbModel,
	}
	user.Set("preferences", prefs)
	return app.Save(user)
}

func assignProviderConfig(cfg *llm.ProviderConfig, raw map[string]any) {
	str := func(k string) string {
		if v, ok := raw[k].(string); ok {
			return v
		}
		return ""
	}
	cfg.LLMProvider = str("llm_provider")
	cfg.MistralKey = str("mistral_key")
	cfg.MistralModel = str("mistral_model")
	cfg.OllamaChatModel = str("ollama_chat_model")
	cfg.Embedder = str("embedder")
	cfg.OllamaURL = str("ollama_url")
	cfg.OllamaModel = str("ollama_model")
	cfg.MistralEmbModel = str("mistral_emb_model")
}

func isRedacted(s string) bool {
	for _, r := range s {
		if r == '•' {
			return true
		}
	}
	return false
}
