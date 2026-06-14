package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/NexVed/Cortex/internal/db"
	"github.com/pocketbase/pocketbase/core"
)

// MCPConnection is the API representation of an IDE connection.
type MCPConnection struct {
	ID         string `json:"id"`
	IDE        string `json:"ide"`
	Label      string `json:"label"`
	ProjectID  string `json:"project_id"`
	ClientName string `json:"client_name"`
	Enabled    bool   `json:"enabled"`
	LastUsed   string `json:"last_used"`
	Connected  bool   `json:"connected"` // used within the last 5 minutes
	Created    string `json:"created"`
	Endpoint   string `json:"endpoint"`
	// Token is only populated on creation (shown once).
	Token  string         `json:"token,omitempty"`
	Config map[string]any `json:"config,omitempty"`
}

func genMCPToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return "cortex_mcp_" + hex.EncodeToString(b)
}

func (s *Service) mcpEndpoint() string {
	port := 8090
	if s.Config != nil && s.Config.Server.Port != 0 {
		port = s.Config.Server.Port
	}
	return fmt.Sprintf("http://127.0.0.1:%d/mcp", port)
}

func (s *Service) mcpConfigSnippet(token string) map[string]any {
	return map[string]any{
		"mcpServers": map[string]any{
			"cortex": map[string]any{
				"url": s.mcpEndpoint(),
				"headers": map[string]any{
					"Authorization": "Bearer " + token,
				},
			},
		},
	}
}

// CreateMCPConnection issues a new MCP token bound to a user/project/IDE.
func (s *Service) CreateMCPConnection(user *core.Record, projectID, ide, label string) (*MCPConnection, error) {
	coll, err := s.App.FindCollectionByNameOrId(db.CollMCPTokens)
	if err != nil {
		return nil, err
	}
	token := genMCPToken()
	rec := core.NewRecord(coll)
	rec.Set("owner", user.Id)
	if projectID != "" {
		rec.Set("project", projectID)
	}
	rec.Set("ide", ide)
	rec.Set("label", label)
	rec.Set("token", token)
	rec.Set("enabled", true)
	if err := s.App.Save(rec); err != nil {
		return nil, err
	}

	out := s.connectionFromRecord(rec)
	out.Token = token
	out.Config = s.mcpConfigSnippet(token)
	return out, nil
}

func (s *Service) connectionFromRecord(rec *core.Record) *MCPConnection {
	lastUsed := rec.GetDateTime("last_used")
	connected := false
	if !lastUsed.IsZero() {
		connected = time.Since(lastUsed.Time()) < 5*time.Minute
	}
	return &MCPConnection{
		ID:         rec.Id,
		IDE:        rec.GetString("ide"),
		Label:      rec.GetString("label"),
		ProjectID:  rec.GetString("project"),
		ClientName: rec.GetString("client_name"),
		Enabled:    rec.GetBool("enabled"),
		LastUsed:   rfc3339(rec, "last_used"),
		Connected:  connected,
		Created:    rfc3339(rec, "created"),
		Endpoint:   s.mcpEndpoint(),
	}
}

func rfc3339(rec *core.Record, field string) string {
	dt := rec.GetDateTime(field)
	if dt.IsZero() {
		return ""
	}
	return dt.Time().Format(time.RFC3339)
}
