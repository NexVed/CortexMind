package daemon

import (
	"github.com/NexVed/Cortex/internal/services"
	"net/http"
)

func (d *Daemon) repositoryInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	insights, err := (services.RepositoryInsightsService{DB: d.DB}).Get(r.PathValue("id"))
	if err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, insights)
}
