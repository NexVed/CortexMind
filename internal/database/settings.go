package database

import (
	"database/sql"
	"encoding/json"
)

func (d *DB) Setting(id string, target any) error {
	var raw string
	err := d.QueryRow(`SELECT payload FROM app_settings WHERE id=?`, id).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), target)
}

func (d *DB) SaveSetting(id string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = d.Exec(`INSERT INTO app_settings(id,payload) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload`, id, string(raw))
	return err
}

func (d *DB) UserIDs() ([]string, error) {
	rows, err := d.Query(`SELECT id FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ResetData clears every application table in one transaction, including MCP credentials.
func (d *DB) ResetData() error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, table := range []string{"active_session", "mcp_connections", "local_records", "repository_scans", "code_graphs", "project_worktrees", "projects", "github_repositories", "github_organizations", "app_settings", "users"} {
		var count int
		if err = tx.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			continue
		}
		if _, err = tx.Exec(`DELETE FROM "` + table + `"`); err != nil {
			return err
		}
	}
	return tx.Commit()
}
