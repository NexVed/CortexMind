package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/NexVed/Cortex/internal/llm"
	"github.com/NexVed/Cortex/internal/services"
)

func (d *Daemon) providers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	u, err := d.DB.CurrentUser()
	if err != nil || u == nil {
		writeError(w, fmt.Errorf("sign in or continue offline first"), http.StatusUnauthorized)
		return
	}
	var cfg llm.ProviderConfig
	if err = d.DB.Setting("providers:"+u.ID, &cfg); err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	if r.Method == http.MethodPost {
		var patch map[string]json.RawMessage
		if err = json.NewDecoder(r.Body).Decode(&patch); err != nil {
			writeError(w, fmt.Errorf("invalid provider configuration"), http.StatusBadRequest)
			return
		}
		key, hasKey := patch["mistral_key"]
		delete(patch, "mistral_key")
		raw, _ := json.Marshal(patch)
		if err = json.Unmarshal(raw, &cfg); err != nil {
			writeError(w, fmt.Errorf("invalid provider configuration"), http.StatusBadRequest)
			return
		}
		cfg.Defaults()
		validProvider := func(value string) bool { return value == "none" || value == "mistral" || value == "ollama" }
		ollama, parseErr := url.Parse(cfg.OllamaURL)
		if !validProvider(cfg.LLMProvider) || !validProvider(cfg.Embedder) || parseErr != nil || (ollama.Scheme != "http" && ollama.Scheme != "https") || ollama.Host == "" || ollama.User != nil {
			writeError(w, fmt.Errorf("invalid provider or Ollama URL"), http.StatusUnprocessableEntity)
			return
		}
		if hasKey {
			var secret string
			if err := json.Unmarshal(key, &secret); err != nil {
				writeError(w, fmt.Errorf("invalid Mistral key"), http.StatusBadRequest)
				return
			}
			if !strings.Contains(secret, "•") {
				if secret == "" {
					err = d.Auth.Tokens.Delete("mistral:" + u.ID)
				} else {
					err = d.Auth.Tokens.Set("mistral:"+u.ID, secret)
				}
				if err != nil {
					writeError(w, err, http.StatusInternalServerError)
					return
				}
			}
		}
		cfg.MistralKey = ""
		if err = d.DB.SaveSetting("providers:"+u.ID, cfg); err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
	}
	cfg.Defaults()
	secret, err := d.Auth.Tokens.Get("mistral:" + u.ID)
	if err == nil {
		cfg.MistralKey = secret
	}
	writeJSON(w, http.StatusOK, cfg.Redacted())
}

// The UI's knowledge graph uses the persisted source graph instead of a missing endpoint.
func (d *Daemon) knowledgeGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	graph, found, err := (services.CodeGraphService{DB: d.DB}).Load(r.PathValue("id"))
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	nodes := []map[string]string{}
	edges := []services.GraphEdge{}
	if found {
		for _, node := range graph.Nodes {
			nodes = append(nodes, map[string]string{"id": node.ID, "label": node.Label, "type": node.Type, "color": "#64748B"})
		}
		edges = graph.Edges
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": nodes, "edges": edges})
}
