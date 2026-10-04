package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/NexVed/Cortex/internal/services"
)

func (d *Daemon) scanWorkingTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var input struct {
		RepoPath    string `json:"repo_path"`
		IncludeDiff bool   `json:"include_diff"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, fmt.Errorf("invalid checkout scan payload"), http.StatusBadRequest)
		return
	}
	result, err := (services.ScanService{DB: d.DB, Config: d.Config, ScanMutex: &d.scans}).ScanWorkingTree(r.Context(), r.PathValue("id"), input.RepoPath, input.IncludeDiff)
	if err != nil {
		writeError(w, err, http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
