package search

import (
	"strings"
	"time"

	"github.com/NexVed/Cortex/internal/db"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Engine performs text search across CORTEX collections.
//
// This implementation uses PocketBase's built-in "contains" (~) filter, which
// maps to SQL LIKE. The backend guide targets SQLite FTS5 for ranking; that can
// layer on later as a virtual table without changing this interface.
type Engine struct {
	App core.App
}

func New(app core.App) *Engine {
	return &Engine{App: app}
}

// Query parameters for a search.
type Query struct {
	Query     string
	ProjectID string
	Scope     []string // files, functions, vault, tasks, handoffs
	Limit     int
	Offset    int
}

// Result is a single unified search hit.
type Result struct {
	ID         string    `json:"id"`
	Collection string    `json:"collection"`
	ProjectID  string    `json:"project_id"`
	Title      string    `json:"title"`
	Excerpt    string    `json:"excerpt"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// scopeMap defines how each scope searches: collection + searchable fields +
// the field used as the result title.
var scopeMap = map[string]struct {
	collection string
	fields     []string
	titleField string
}{
	"vault":     {db.CollVaultEntries, []string{"title", "content"}, "title"},
	"tasks":     {db.CollTasks, []string{"title", "description"}, "title"},
	"handoffs":  {db.CollHandoffs, []string{"title", "context"}, "title"},
	"files":     {db.CollFileIndex, []string{"path", "summary"}, "path"},
	"functions": {db.CollFileIndex, []string{"path"}, "path"},
}

// Search runs the query across the requested scopes.
func (e *Engine) Search(q Query) ([]Result, error) {
	if q.Limit <= 0 {
		q.Limit = 30
	}
	scopes := q.Scope
	if len(scopes) == 0 {
		scopes = []string{"files", "vault", "tasks"}
	}

	seen := map[string]bool{}
	results := []Result{}

	for _, scope := range scopes {
		def, ok := scopeMap[scope]
		if !ok {
			continue
		}

		var clauses []string
		params := map[string]any{"q": q.Query}
		for _, f := range def.fields {
			clauses = append(clauses, f+" ~ {:q}")
		}
		filter := "(" + strings.Join(clauses, " || ") + ")"
		if q.ProjectID != "" {
			filter += " && project = {:pid}"
			params["pid"] = q.ProjectID
		}

		records, err := e.App.FindRecordsByFilter(def.collection, filter, "-updated", q.Limit, q.Offset, params)
		if err != nil {
			continue
		}
		for _, r := range records {
			if seen[r.Id] {
				continue
			}
			seen[r.Id] = true
			results = append(results, Result{
				ID:         r.Id,
				Collection: def.collection,
				ProjectID:  r.GetString("project"),
				Title:      r.GetString(def.titleField),
				Excerpt:    excerpt(r, def.fields, q.Query),
				UpdatedAt:  r.GetDateTime("updated").Time(),
			})
		}
	}
	return results, nil
}

// RecordHistory persists the query into search_history.
func (e *Engine) RecordHistory(ownerID string, q Query, count int) {
	coll, err := e.App.FindCollectionByNameOrId(db.CollSearchHistory)
	if err != nil {
		return
	}
	rec := core.NewRecord(coll)
	rec.Set("owner", ownerID)
	rec.Set("query", q.Query)
	rec.Set("scope", q.Scope)
	rec.Set("results", count)
	_ = e.App.Save(rec)
}

var _ = types.NowDateTime // keep types import available for future ranking work

// excerpt returns a short snippet around the first match of the query.
func excerpt(r *core.Record, fields []string, query string) string {
	q := strings.ToLower(query)
	for _, f := range fields {
		val := r.GetString(f)
		if val == "" {
			continue
		}
		idx := strings.Index(strings.ToLower(val), q)
		if idx < 0 {
			continue
		}
		start := idx - 40
		if start < 0 {
			start = 0
		}
		end := idx + len(query) + 60
		if end > len(val) {
			end = len(val)
		}
		snippet := strings.TrimSpace(val[start:end])
		if start > 0 {
			snippet = "..." + snippet
		}
		if end < len(val) {
			snippet += "..."
		}
		return snippet
	}
	return ""
}
