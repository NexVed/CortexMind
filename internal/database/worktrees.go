package database

import "fmt"

func (d *DB) BindWorkingTree(projectID, path string) error {
	if _, err := d.Project(projectID); err != nil {
		return fmt.Errorf("project not found: %w", err)
	}
	_, err := d.Exec(`INSERT INTO project_worktrees(project_id,local_path) VALUES(?,?) ON CONFLICT(project_id) DO UPDATE SET local_path=excluded.local_path`, projectID, path)
	return err
}

func (d *DB) WorkingTreePath(projectID string) (string, error) {
	var path string
	err := d.QueryRow(`SELECT local_path FROM project_worktrees WHERE project_id=?`, projectID).Scan(&path)
	return path, err
}
