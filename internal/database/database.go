package database

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ *sql.DB }
type User struct {
	ID, Provider, GitHubID, Username, DisplayName, AvatarURL string
	Offline                                                  bool
}
type Repository struct {
	GitHubID  int64  `json:"github_id"`
	Name      string `json:"name"`
	FullName  string `json:"full_name"`
	Private   bool   `json:"private"`
	CloneURL  string `json:"clone_url"`
	HTMLURL   string `json:"html_url"`
	UpdatedAt string `json:"updated_at"`
}
type Project struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	Description  string   `json:"description"`
	GitHubURL    string   `json:"github_url"`
	GitHubRepoID string   `json:"github_repo_id"`
	Status       string   `json:"status"`
	Progress     float64  `json:"progress"`
	Technologies []string `json:"technologies"`
	LastScanned  string   `json:"last_scanned"`
	LastActivity string   `json:"last_activity"`
	IconColor    string   `json:"icon_color"`
	Created      string   `json:"created"`
	Updated      string   `json:"updated"`
}

func Open(dataDir string) (*DB, error) {
	// Apply these pragmas on every pooled connection, not just the first one.
	conn, err := sql.Open("sqlite", filepath.Join(dataDir, "cortexmind.db")+"?_pragma=busy_timeout%285000%29&_pragma=foreign_keys%281%29")
	if err != nil {
		return nil, err
	}
	db := &DB{conn}
	conn.SetMaxOpenConns(4)
	conn.SetMaxIdleConns(4)
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS project_worktrees (project_id TEXT PRIMARY KEY, local_path TEXT NOT NULL)`); err != nil {
		conn.Close()
		return nil, err
	}
	if _, err = db.Exec(`PRAGMA journal_mode = WAL; CREATE TABLE IF NOT EXISTS local_records (collection_name TEXT NOT NULL, id TEXT PRIMARY KEY, project_id TEXT NOT NULL DEFAULT '', payload TEXT NOT NULL, created TEXT NOT NULL, updated TEXT NOT NULL); CREATE INDEX IF NOT EXISTS idx_local_records_collection_project ON local_records(collection_name,project_id,updated DESC); CREATE TABLE IF NOT EXISTS mcp_connections (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, ide TEXT NOT NULL, label TEXT NOT NULL, client_name TEXT NOT NULL DEFAULT '', token_hash TEXT NOT NULL UNIQUE, enabled INTEGER NOT NULL DEFAULT 1, created TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, last_used TEXT NOT NULL DEFAULT ''); CREATE INDEX IF NOT EXISTS idx_mcp_connections_token ON mcp_connections(token_hash); CREATE TABLE IF NOT EXISTS app_settings (id TEXT PRIMARY KEY, payload TEXT NOT NULL);`); err != nil {
		conn.Close()
		return nil, err
	}
	if _, err = db.Exec(`PRAGMA foreign_keys = ON; CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY, provider TEXT NOT NULL, github_id TEXT NOT NULL DEFAULT '', username TEXT NOT NULL DEFAULT '', display_name TEXT NOT NULL DEFAULT '', avatar_url TEXT NOT NULL DEFAULT '', offline INTEGER NOT NULL DEFAULT 0, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP); CREATE TABLE IF NOT EXISTS active_session (slot INTEGER PRIMARY KEY CHECK (slot = 1), user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP); CREATE TABLE IF NOT EXISTS github_organizations (login TEXT PRIMARY KEY, avatar_url TEXT NOT NULL DEFAULT ''); CREATE TABLE IF NOT EXISTS github_repositories (github_id INTEGER PRIMARY KEY, name TEXT NOT NULL, full_name TEXT NOT NULL, private INTEGER NOT NULL, clone_url TEXT NOT NULL, html_url TEXT NOT NULL, updated_at TEXT NOT NULL); CREATE TABLE IF NOT EXISTS repository_scans (github_id TEXT PRIMARY KEY, local_path TEXT NOT NULL, indexed_files INTEGER NOT NULL, last_scanned TEXT NOT NULL); CREATE TABLE IF NOT EXISTS code_graphs (project_id TEXT PRIMARY KEY, payload TEXT NOT NULL, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP); CREATE TABLE IF NOT EXISTS projects (id TEXT PRIMARY KEY, name TEXT NOT NULL, path TEXT NOT NULL DEFAULT '', description TEXT NOT NULL DEFAULT '', github_url TEXT NOT NULL DEFAULT '', github_repo_id TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'active', progress REAL NOT NULL DEFAULT 0, icon_color TEXT NOT NULL DEFAULT '', created TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, updated TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);`); err != nil {
		conn.Close()
		return nil, err
	}
	return db, nil
}
func (d *DB) SaveUser(u User) error {
	_, err := d.Exec(`INSERT INTO users (id,provider,github_id,username,display_name,avatar_url,offline,updated_at) VALUES (?,?,?,?,?,?,?,CURRENT_TIMESTAMP) ON CONFLICT(id) DO UPDATE SET provider=excluded.provider,github_id=excluded.github_id,username=excluded.username,display_name=excluded.display_name,avatar_url=excluded.avatar_url,offline=excluded.offline,updated_at=CURRENT_TIMESTAMP`, u.ID, u.Provider, u.GitHubID, u.Username, u.DisplayName, u.AvatarURL, boolInt(u.Offline))
	return err
}
func (d *DB) CurrentUser() (*User, error) {
	row := d.QueryRow(`SELECT u.id,u.provider,u.github_id,u.username,u.display_name,u.avatar_url,u.offline FROM active_session s JOIN users u ON u.id=s.user_id WHERE s.slot=1`)
	var u User
	var offline int
	if err := row.Scan(&u.ID, &u.Provider, &u.GitHubID, &u.Username, &u.DisplayName, &u.AvatarURL, &offline); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	u.Offline = offline != 0
	return &u, nil
}
func (d *DB) SetActiveUser(id string) error {
	_, err := d.Exec(`INSERT INTO active_session(slot,user_id,updated_at) VALUES(1,?,CURRENT_TIMESTAMP) ON CONFLICT(slot) DO UPDATE SET user_id=excluded.user_id,updated_at=CURRENT_TIMESTAMP`, id)
	return err
}

func (d *DB) ClearActiveUser() error {
	_, err := d.Exec(`DELETE FROM active_session WHERE slot=1`)
	return err
}
func (d *DB) ReplaceGitHubData(orgs []Organization, repos []Repository) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`DELETE FROM github_organizations; DELETE FROM github_repositories`); err != nil {
		return err
	}
	for _, o := range orgs {
		if _, err = tx.Exec(`INSERT INTO github_organizations(login,avatar_url) VALUES (?,?)`, o.Login, o.AvatarURL); err != nil {
			return err
		}
	}
	for _, r := range repos {
		if _, err = tx.Exec(`INSERT INTO github_repositories(github_id,name,full_name,private,clone_url,html_url,updated_at) VALUES (?,?,?,?,?,?,?)`, r.GitHubID, r.Name, r.FullName, boolInt(r.Private), r.CloneURL, r.HTMLURL, r.UpdatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListProjects exposes local projects plus GitHub repositories as ready-to-open projects.
func (d *DB) ListProjects() ([]Project, error) {
	rows, err := d.Query(`SELECT CAST(g.github_id AS TEXT),g.name,COALESCE(w.local_path,''),'',g.html_url,CAST(g.github_id AS TEXT),'active',0,'',g.updated_at,g.updated_at FROM github_repositories g LEFT JOIN project_worktrees w ON w.project_id=CAST(g.github_id AS TEXT) ORDER BY g.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Path, &p.Description, &p.GitHubURL, &p.GitHubRepoID, &p.Status, &p.Progress, &p.IconColor, &p.LastActivity, &p.Updated); err != nil {
			return nil, err
		}
		p.Created = p.Updated
		p.Technologies = []string{}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	localRows, err := d.Query(`SELECT p.id,p.name,COALESCE(w.local_path,p.path),p.description,p.github_url,p.github_repo_id,p.status,p.progress,p.icon_color,p.created,p.updated FROM projects p LEFT JOIN project_worktrees w ON w.project_id=p.id ORDER BY p.updated DESC`)
	if err != nil {
		return nil, err
	}
	defer localRows.Close()
	for localRows.Next() {
		var p Project
		if err := localRows.Scan(&p.ID, &p.Name, &p.Path, &p.Description, &p.GitHubURL, &p.GitHubRepoID, &p.Status, &p.Progress, &p.IconColor, &p.Created, &p.Updated); err != nil {
			return nil, err
		}
		p.Technologies = []string{}
		p.LastActivity = p.Updated
		out = append(out, p)
	}
	return out, localRows.Err()
}
func (d *DB) Project(id string) (*Project, error) {
	for _, p := range mustProjects(d.ListProjects()) {
		if p.ID == id {
			return &p, nil
		}
	}
	var p Project
	err := d.QueryRow(`SELECT id,name,path,description,github_url,github_repo_id,status,progress,icon_color,created,updated FROM projects WHERE id=?`, id).
		Scan(&p.ID, &p.Name, &p.Path, &p.Description, &p.GitHubURL, &p.GitHubRepoID, &p.Status, &p.Progress, &p.IconColor, &p.Created, &p.Updated)
	if err != nil {
		return nil, err
	}
	p.Technologies = []string{}
	p.LastActivity = p.Updated
	return &p, nil
}
func mustProjects(projects []Project, err error) []Project {
	if err != nil {
		return nil
	}
	return projects
}
func (d *DB) CreateProject(name, path, description, githubURL string) (*Project, error) {
	if name == "" {
		return nil, fmt.Errorf("project name is required")
	}
	id := "local-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	now := time.Now().UTC().Format(time.RFC3339)
	p := &Project{ID: id, Name: name, Path: path, Description: description, GitHubURL: githubURL, Status: "active", Technologies: []string{}, LastActivity: now, Created: now, Updated: now}
	_, err := d.Exec(`INSERT INTO projects(id,name,path,description,github_url,status,progress,icon_color,created,updated) VALUES(?,?,?,?,?,'active',0,'',?,?)`, p.ID, p.Name, p.Path, p.Description, p.GitHubURL, now, now)
	if err != nil {
		return nil, err
	}
	return p, nil
}

type Organization struct{ Login, AvatarURL string }

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func (d *DB) Close() error { return d.DB.Close() }

// appended repository lookup
func (d *DB) Repository(id string) (*Repository, error) {
	var r Repository
	var private int
	err := d.QueryRow(`SELECT github_id,name,full_name,private,clone_url,html_url,updated_at FROM github_repositories WHERE CAST(github_id AS TEXT)=?`, id).Scan(&r.GitHubID, &r.Name, &r.FullName, &private, &r.CloneURL, &r.HTMLURL, &r.UpdatedAt)
	r.Private = private != 0
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (d *DB) SaveRepositoryScan(id, path string, indexed int) error {
	_, err := d.Exec(`INSERT INTO repository_scans(github_id,local_path,indexed_files,last_scanned) VALUES(?,?,?,CURRENT_TIMESTAMP) ON CONFLICT(github_id) DO UPDATE SET local_path=excluded.local_path,indexed_files=excluded.indexed_files,last_scanned=CURRENT_TIMESTAMP`, id, path, indexed)
	return err
}
func (d *DB) RepositoryPath(id string) (string, error) {
	var path string
	err := d.QueryRow(`SELECT local_path FROM repository_scans WHERE github_id=?`, id).Scan(&path)
	return path, err
}
