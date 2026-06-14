package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/NexVed/Cortex/internal/vector"
)

const mistralBase = "https://api.mistral.ai/v1"

// Mistral is a minimal client for the Mistral chat + embeddings APIs.
type Mistral struct {
	APIKey string
	Model  string
	HTTP   *http.Client
}

func NewMistral(apiKey, model string) *Mistral {
	if model == "" {
		model = "mistral-small-latest"
	}
	return &Mistral{
		APIKey: apiKey,
		Model:  model,
		HTTP:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (m *Mistral) Enabled() bool { return m.APIKey != "" }

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatReq struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatResp struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// Summarize sends a system+user prompt to Mistral chat completions.
func (m *Mistral) Summarize(ctx context.Context, system, user string) (string, error) {
	if m.APIKey == "" {
		return "", fmt.Errorf("mistral: no api key")
	}
	body, _ := json.Marshal(chatReq{
		Model: m.Model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.2,
		MaxTokens:   700,
	})
	var out chatResp
	if err := m.post(ctx, "/chat/completions", body, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("mistral: empty response")
	}
	return out.Choices[0].Message.Content, nil
}

func (m *Mistral) post(ctx context.Context, path string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mistralBase+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.APIKey)
	resp, err := m.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("mistral %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mistral %s status %d: %s", path, resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// ── Embeddings ─────────────────────────────────────────

// MistralEmbedder implements vector.Embedder using the Mistral embeddings API.
type MistralEmbedder struct {
	APIKey string
	Model  string
	HTTP   *http.Client
}

var _ vector.Embedder = (*MistralEmbedder)(nil)

func NewMistralEmbedder(apiKey, model string) *MistralEmbedder {
	if model == "" {
		model = "mistral-embed"
	}
	return &MistralEmbedder{APIKey: apiKey, Model: model, HTTP: &http.Client{Timeout: 60 * time.Second}}
}

type embedReq struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResp struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (e *MistralEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if e.APIKey == "" {
		return nil, fmt.Errorf("mistral: no api key")
	}
	body, _ := json.Marshal(embedReq{Model: e.Model, Input: []string{text}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mistralBase+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.APIKey)
	resp, err := e.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mistral embed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mistral embed status %d: %s", resp.StatusCode, string(b))
	}
	var out embedResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 || len(out.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("mistral returned empty embedding")
	}
	return out.Data[0].Embedding, nil
}

// NewEmbedder builds a vector.Embedder for the configured provider, or nil when
// embeddings are disabled.
func NewEmbedder(cfg ProviderConfig) vector.Embedder {
	switch cfg.Embedder {
	case "mistral":
		if cfg.MistralKey != "" {
			return NewMistralEmbedder(cfg.MistralKey, cfg.MistralEmbModel)
		}
	case "ollama":
		return vector.NewOllamaEmbedder(cfg.OllamaURL, cfg.OllamaModel)
	}
	return nil
}
