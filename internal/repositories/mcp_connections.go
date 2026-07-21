package repositories

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/NexVed/Cortex/internal/database"
)

// MCPConnection is a local agent connection. TokenHash is never exposed.
type MCPConnection struct {
	ID         string `json:"id"`
	IDE        string `json:"ide"`
	Label      string `json:"label"`
	ProjectID  string `json:"project_id"`
	ClientName string `json:"client_name"`
	Enabled    bool   `json:"enabled"`
	LastUsed   string `json:"last_used"`
	Connected  bool   `json:"connected"`
	Created    string `json:"created"`
	Endpoint   string `json:"endpoint"`
}

type MCPConnectionRepository struct{ DB *database.DB }

func (r MCPConnectionRepository) ensure() error {
	_, err := r.DB.Exec(`CREATE TABLE IF NOT EXISTS mcp_connections (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, ide TEXT NOT NULL, label TEXT NOT NULL, client_name TEXT NOT NULL DEFAULT '', token_hash TEXT NOT NULL UNIQUE, enabled INTEGER NOT NULL DEFAULT 1, created TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, last_used TEXT NOT NULL DEFAULT ''); CREATE INDEX IF NOT EXISTS idx_mcp_connections_token ON mcp_connections(token_hash);`)
	return err
}

func (r MCPConnectionRepository) Create(connection MCPConnection, token string) (*MCPConnection, error) {
	if err := r.ensure(); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.DB.Exec(`INSERT INTO mcp_connections(id,project_id,ide,label,client_name,token_hash,enabled,created,last_used) VALUES(?,?,?,?,?,?,?,?,?)`, connection.ID, connection.ProjectID, connection.IDE, connection.Label, connection.ClientName, tokenHash(token), boolInt(connection.Enabled), now, "")
	if err != nil {
		return nil, err
	}
	connection.Created = now
	connection.LastUsed = ""
	connection.Connected = false
	return &connection, nil
}

func (r MCPConnectionRepository) List() ([]MCPConnection, error) {
	if err := r.ensure(); err != nil {
		return nil, err
	}
	rows, err := r.DB.Query(`SELECT id,ide,label,project_id,client_name,enabled,last_used,created FROM mcp_connections ORDER BY created DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	connections := []MCPConnection{}
	for rows.Next() {
		connection, err := scanMCPConnection(rows)
		if err != nil {
			return nil, err
		}
		connections = append(connections, connection)
	}
	return connections, rows.Err()
}

func (r MCPConnectionRepository) Get(id string) (*MCPConnection, error) {
	if err := r.ensure(); err != nil {
		return nil, err
	}
	connection, err := scanMCPConnection(r.DB.QueryRow(`SELECT id,ide,label,project_id,client_name,enabled,last_used,created FROM mcp_connections WHERE id=?`, id))
	if err == sql.ErrNoRows {
		return nil, err
	}
	return &connection, err
}

func (r MCPConnectionRepository) Delete(id string) error {
	if err := r.ensure(); err != nil {
		return err
	}
	_, err := r.DB.Exec(`DELETE FROM mcp_connections WHERE id=?`, id)
	return err
}

func (r MCPConnectionRepository) Authenticate(token string) (*MCPConnection, error) {
	if err := r.ensure(); err != nil {
		return nil, err
	}
	connection, err := scanMCPConnection(r.DB.QueryRow(`SELECT id,ide,label,project_id,client_name,enabled,last_used,created FROM mcp_connections WHERE token_hash=? AND enabled=1`, tokenHash(token)))
	if err != nil {
		return nil, err
	}
	return &connection, nil
}

func (r MCPConnectionRepository) Touch(id string) error {
	if err := r.ensure(); err != nil {
		return err
	}
	_, err := r.DB.Exec(`UPDATE mcp_connections SET last_used=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

type scanner interface{ Scan(...any) error }

func scanMCPConnection(row scanner) (MCPConnection, error) {
	var connection MCPConnection
	var enabled int
	if err := row.Scan(&connection.ID, &connection.IDE, &connection.Label, &connection.ProjectID, &connection.ClientName, &enabled, &connection.LastUsed, &connection.Created); err != nil {
		return MCPConnection{}, err
	}
	connection.Enabled = enabled != 0
	connection.Connected = isRecentlyUsed(connection.LastUsed)
	return connection, nil
}
func isRecentlyUsed(value string) bool {
	if value == "" {
		return false
	}
	used, err := time.Parse(time.RFC3339, value)
	return err == nil && time.Since(used) < 5*time.Minute
}
func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
