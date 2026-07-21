package database

import "database/sql"

// SaveCodeGraph stores a generated graph separately from operational records so
// the graph can be rendered immediately on a later application launch.
func (d *DB) SaveCodeGraph(projectID string, payload []byte) error {
	_, err := d.Exec(`INSERT INTO code_graphs(project_id,payload,updated_at) VALUES(?,?,CURRENT_TIMESTAMP) ON CONFLICT(project_id) DO UPDATE SET payload=excluded.payload,updated_at=CURRENT_TIMESTAMP`, projectID, string(payload))
	return err
}

// LoadCodeGraph returns the stored JSON graph and whether a snapshot exists.
func (d *DB) LoadCodeGraph(projectID string) ([]byte, bool, error) {
	var payload string
	err := d.QueryRow(`SELECT payload FROM code_graphs WHERE project_id=?`, projectID).Scan(&payload)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return []byte(payload), true, nil
}
