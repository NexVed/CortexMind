package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/NexVed/Cortex/internal/database"
	"github.com/NexVed/Cortex/internal/repositories"
)

type MCPConnectionService struct {
	Connections repositories.MCPConnectionRepository
	Projects    *database.DB
	Endpoint    string
}

type CreateMCPConnectionInput struct {
	ProjectID string `json:"project_id"`
	IDE       string `json:"ide"`
	Label     string `json:"label"`
}

type CreatedMCPConnection struct {
	repositories.MCPConnection
	Token  string         `json:"token"`
	Config map[string]any `json:"config"`
}

func (s MCPConnectionService) Create(input CreateMCPConnectionInput) (*CreatedMCPConnection, error) {
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.IDE = strings.TrimSpace(input.IDE)
	input.Label = strings.TrimSpace(input.Label)
	if input.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	if input.IDE == "" {
		return nil, fmt.Errorf("ide is required")
	}
	if len(input.IDE) > 64 || len(input.Label) > 160 {
		return nil, fmt.Errorf("connection details are too long")
	}
	if _, err := s.Projects.Project(input.ProjectID); err != nil {
		return nil, fmt.Errorf("project not found")
	}
	token, err := newMCPToken()
	if err != nil {
		return nil, err
	}
	id, err := newMCPID()
	if err != nil {
		return nil, err
	}
	if input.Label == "" {
		input.Label = input.IDE + " connection"
	}
	connection, err := s.Connections.Create(repositories.MCPConnection{ID: id, ProjectID: input.ProjectID, IDE: input.IDE, Label: input.Label, Enabled: true, Endpoint: s.Endpoint}, token)
	if err != nil {
		return nil, err
	}
	return &CreatedMCPConnection{MCPConnection: *connection, Token: token, Config: connectionConfig(s.Endpoint, token)}, nil
}

func (s MCPConnectionService) List() ([]repositories.MCPConnection, error) {
	connections, err := s.Connections.List()
	if err != nil {
		return nil, err
	}
	for i := range connections {
		connections[i].Endpoint = s.Endpoint
	}
	return connections, nil
}

func (s MCPConnectionService) Get(id string) (*repositories.MCPConnection, error) {
	connection, err := s.Connections.Get(id)
	if err != nil {
		return nil, err
	}
	connection.Endpoint = s.Endpoint
	return connection, nil
}

func connectionConfig(endpoint, token string) map[string]any {
	return map[string]any{
		"transport": "streamable-http",
		"url":       endpoint,
		"headers":   map[string]string{"Authorization": "Bearer " + token},
	}
}
func newMCPToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "cmcp_" + base64.RawURLEncoding.EncodeToString(bytes), nil
}
func newMCPID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "mcp_" + base64.RawURLEncoding.EncodeToString(bytes), nil
}
