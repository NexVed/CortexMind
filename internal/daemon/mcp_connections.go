package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/NexVed/Cortex/internal/repositories"
	"github.com/NexVed/Cortex/internal/services"
)

func (d *Daemon) mcpConnectionService() services.MCPConnectionService {
	return services.MCPConnectionService{
		Connections: repositories.MCPConnectionRepository{DB: d.DB},
		Projects:    d.DB,
		Endpoint:    fmt.Sprintf("http://127.0.0.1:%d/mcp", d.Config.Server.Port),
	}
}

func (d *Daemon) mcpConnections(w http.ResponseWriter, r *http.Request) {
	service := d.mcpConnectionService()
	switch r.Method {
	case http.MethodGet:
		connections, err := service.List()
		if err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, connections)
	case http.MethodPost:
		var input services.CreateMCPConnectionInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, fmt.Errorf("invalid connection payload"), http.StatusBadRequest)
			return
		}
		connection, err := service.Create(input)
		if err != nil {
			writeError(w, err, http.StatusUnprocessableEntity)
			return
		}
		writeJSON(w, http.StatusCreated, connection)
	default:
		methodNotAllowed(w)
	}
}

func (d *Daemon) mcpConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w)
		return
	}
	if err := d.mcpConnectionService().Connections.Delete(r.PathValue("id")); err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (d *Daemon) mcpConnectionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	connection, err := d.mcpConnectionService().Get(r.PathValue("id"))
	if err != nil {
		writeError(w, fmt.Errorf("connection not found"), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, connection)
}
