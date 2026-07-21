package daemon

import (
	"fmt"
	"net/http"

	"github.com/NexVed/Cortex/internal/services"
)

// codeGraph returns the last persisted graph on GET. POST explicitly rebuilds it
// from the scanned local repository. The response contract stays identical for
// both methods, so the UI can render either state without special cases.
func (d *Daemon) codeGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	projectID := r.PathValue("id")
	if projectID == "" {
		writeError(w, fmt.Errorf("project id is required"), http.StatusBadRequest)
		return
	}
	graphs := services.CodeGraphService{DB: d.DB}
	if r.Method == http.MethodGet {
		graph, found, err := graphs.Load(projectID)
		if err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
		if !found {
			writeJSON(w, http.StatusOK, services.CodeGraph{ProjectID: projectID, Nodes: []services.GraphNode{}, Edges: []services.GraphEdge{}, Built: false})
			return
		}
		writeJSON(w, http.StatusOK, graph)
		return
	}
	graph, err := graphs.Build(projectID)
	if err != nil {
		writeError(w, err, http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, http.StatusOK, graph)
}
