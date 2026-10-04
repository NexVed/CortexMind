package daemon

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

func (d *Daemon) reset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	d.Auth.CancelGitHub()
	ids, err := d.DB.UserIDs()
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	for _, id := range ids {
		for _, account := range []string{"github:" + id, "mistral:" + id} {
			if err := d.Auth.Tokens.Delete(account); err != nil {
				writeError(w, fmt.Errorf("remove saved credential: %w", err), http.StatusInternalServerError)
				return
			}
		}
	}
	if err = d.DB.ResetData(); err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	// Only remove the daemon's cache; user-selected project directories are never here.
	root, err := filepath.Abs(d.Config.DataDirPath())
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	cache := filepath.Join(root, "repositories")
	if filepath.Dir(cache) != root {
		writeError(w, fmt.Errorf("invalid repository cache path"), http.StatusInternalServerError)
		return
	}
	if err = os.RemoveAll(cache); err != nil {
		writeError(w, fmt.Errorf("remove repository cache: %w", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
