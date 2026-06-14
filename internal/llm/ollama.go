package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Ollama is a minimal client for the local Ollama chat API, used for prompt
// generation when the user selects Ollama as their LLM provider.
type Ollama struct {
	URL   string
	Model string
	HTTP  *http.Client
}

func NewOllama(url, model string) *Ollama {
	if url == "" {
		url = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3.1"
	}
	return &Ollama{URL: url, Model: model, HTTP: &http.Client{Timeout: 180 * time.Second}}
}

func (o *Ollama) Enabled() bool { return o.URL != "" }

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatReq struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type ollamaChatResp struct {
	Message ollamaChatMessage `json:"message"`
}

// Summarize sends a system+user prompt to Ollama's chat endpoint.
func (o *Ollama) Summarize(ctx context.Context, system, user string) (string, error) {
	body, _ := json.Marshal(ollamaChatReq{
		Model:  o.Model,
		Stream: false,
		Messages: []ollamaChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.URL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama chat: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama chat status %d: %s", resp.StatusCode, string(b))
	}
	var out ollamaChatResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Message.Content, nil
}
