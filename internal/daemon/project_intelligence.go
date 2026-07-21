package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/NexVed/Cortex/internal/services"
)

func (d *Daemon) intelligence() services.ProjectIntelligenceService {
	return services.ProjectIntelligenceService{DB: d.DB}
}

func (d *Daemon) generateSessionDigest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, fmt.Errorf("invalid digest payload"), http.StatusBadRequest)
		return
	}
	digest, err := d.intelligence().GenerateDigest(r.PathValue("id"), input.SessionID)
	if err != nil {
		writeError(w, err, http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, http.StatusCreated, digest)
}
func (d *Daemon) sessionDigests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	digests, err := d.intelligence().ListDigests(r.PathValue("id"))
	if err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, digests)
}
func (d *Daemon) agentMemories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	memories, err := d.intelligence().ListAgentMemories(r.PathValue("id"), queryLimit(r, 500))
	if err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, memories)
}
func (d *Daemon) activeAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	agents, err := d.intelligence().ActiveAgents(r.PathValue("id"))
	if err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, agents)
}
func queryLimit(r *http.Request, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
func (d *Daemon) systemPrompt(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	service := d.intelligence()
	switch r.Method {
	case http.MethodGet:
		result, err := service.SystemPrompt(projectID)
		if err != nil {
			writeError(w, err, http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, result)
	case http.MethodPut:
		var input struct {
			Prompt string `json:"prompt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, fmt.Errorf("invalid system prompt payload"), http.StatusBadRequest)
			return
		}
		result, err := service.SaveSystemPrompt(projectID, input.Prompt)
		if err != nil {
			writeError(w, err, http.StatusUnprocessableEntity)
			return
		}
		writeJSON(w, http.StatusOK, result)
	case http.MethodPost:
		var input struct {
			IncludeTasks    bool `json:"include_tasks"`
			IncludeVault    bool `json:"include_vault"`
			IncludeActivity bool `json:"include_activity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, fmt.Errorf("invalid prompt generation payload"), http.StatusBadRequest)
			return
		}
		result, err := service.GenerateSystemPrompt(projectID, input.IncludeTasks, input.IncludeVault, input.IncludeActivity)
		if err != nil {
			writeError(w, err, http.StatusUnprocessableEntity)
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		methodNotAllowed(w)
	}
}
