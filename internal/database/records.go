package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// ReplaceProjectFiles commits an index snapshot without accumulating old scan rows.
func (s RecordStore) ReplaceProjectFiles(ctx context.Context, projectID string, files []map[string]any) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM local_records WHERE collection_name='file_index' AND project_id=?`, projectID); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, file := range files {
		id, err := recordID()
		if err != nil {
			return err
		}
		raw, err := json.Marshal(file)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO local_records(collection_name,id,project_id,payload,created,updated) VALUES('file_index',?,?,?,?,?)`, id, projectID, string(raw), now, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type RecordStore struct{ DB *DB }
type Page struct {
	Page       int              `json:"page"`
	PerPage    int              `json:"perPage"`
	TotalPages int              `json:"totalPages"`
	TotalItems int              `json:"totalItems"`
	Items      []map[string]any `json:"items"`
}

func (s RecordStore) List(collection, projectID string, limit int) (Page, error) {
	q := `SELECT id,payload,created,updated FROM local_records WHERE collection_name=?`
	args := []any{collection}
	if projectID != "" {
		q += ` AND project_id=?`
		args = append(args, projectID)
	}
	q += ` ORDER BY updated DESC`
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, raw, created, updated string
		if err := rows.Scan(&id, &raw, &created, &updated); err != nil {
			return Page{}, err
		}
		var item map[string]any
		if err := json.Unmarshal([]byte(raw), &item); err != nil {
			return Page{}, err
		}
		if item == nil {
			return Page{}, fmt.Errorf("record %s has an invalid null payload", id)
		}
		item["id"] = id
		item["created"] = created
		item["updated"] = updated
		items = append(items, item)
	}
	return Page{Page: 1, PerPage: len(items), TotalPages: 1, TotalItems: len(items), Items: items}, rows.Err()
}
func (s RecordStore) Get(collection, id string) (map[string]any, error) {
	var raw, created, updated string
	err := s.DB.QueryRow(`SELECT payload,created,updated FROM local_records WHERE collection_name=? AND id=?`, collection, id).Scan(&raw, &created, &updated)
	if err != nil {
		return nil, err
	}
	var item map[string]any
	if err = json.Unmarshal([]byte(raw), &item); err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("record %s has an invalid null payload", id)
	}
	item["id"] = id
	item["created"] = created
	item["updated"] = updated
	return item, nil
}
func (s RecordStore) Create(collection string, item map[string]any) (map[string]any, error) {
	id, err := recordID()
	if err != nil {
		return nil, err
	}
	return s.save(collection, id, item, true)
}
func (s RecordStore) Update(collection, id string, patch map[string]any) (map[string]any, error) {
	current, err := s.Get(collection, id)
	if err != nil {
		return nil, err
	}
	for k, v := range patch {
		if k != "id" && k != "created" && k != "updated" {
			current[k] = v
		}
	}
	return s.save(collection, id, current, false)
}
func (s RecordStore) Delete(collection, id string) error {
	_, err := s.DB.Exec(`DELETE FROM local_records WHERE collection_name=? AND id=?`, collection, id)
	return err
}
func (s RecordStore) save(collection, id string, item map[string]any, create bool) (map[string]any, error) {
	if item == nil {
		return nil, fmt.Errorf("record payload must be a JSON object")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := now
	if !create {
		var err error
		err = s.DB.QueryRow(`SELECT created FROM local_records WHERE collection_name=? AND id=?`, collection, id).Scan(&created)
		if err != nil {
			return nil, err
		}
	}
	project, _ := item["project"].(string)
	delete(item, "id")
	delete(item, "created")
	delete(item, "updated")
	raw, err := json.Marshal(item)
	if err != nil {
		return nil, err
	}
	if create {
		_, err = s.DB.Exec(`INSERT INTO local_records(collection_name,id,project_id,payload,created,updated) VALUES(?,?,?,?,?,?)`, collection, id, project, string(raw), created, now)
	} else {
		_, err = s.DB.Exec(`UPDATE local_records SET project_id=?,payload=?,updated=? WHERE collection_name=? AND id=?`, project, string(raw), now, collection, id)
	}
	if err != nil {
		return nil, err
	}
	item["id"] = id
	item["created"] = created
	item["updated"] = now
	return item, nil
}
func recordID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
