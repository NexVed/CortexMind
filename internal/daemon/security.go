package daemon

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/NexVed/Cortex/internal/localauth"
)

const maxRequestBytes = 2 << 20

func (d *Daemon) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		port := d.Config.Server.Port
		if r.Host != fmt.Sprintf("127.0.0.1:%d", port) && r.Host != fmt.Sprintf("localhost:%d", port) {
			writeError(w, fmt.Errorf("invalid local host"), http.StatusForbidden)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		origin := r.Header.Get("Origin")
		if origin != "" {
			u, err := url.Parse(origin)
			same := err == nil && u.Scheme == "http" && u.Host == r.Host && u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == ""
			dev := localauth.ValidDevOrigin(d.Config.Server.DevOrigin) && origin == strings.TrimSuffix(d.Config.Server.DevOrigin, "/")
			if !same && !dev {
				writeError(w, fmt.Errorf("untrusted request origin"), http.StatusForbidden)
				return
			}
		}
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			writeError(w, fmt.Errorf("cross-site requests are not allowed"), http.StatusForbidden)
			return
		}
		isAPI := strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/cortex.v1.")
		if isAPI || r.URL.Path == "/mcp" {
			w.Header().Set("Cache-Control", "no-store")
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if contentType != "application/json" {
					writeError(w, fmt.Errorf("Content-Type must be application/json"), http.StatusUnsupportedMediaType)
					return
				}
			}
		}
		if isAPI && r.URL.Path != "/api/health" {
			value := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			expected, actual := sha256.Sum256([]byte(d.APIToken)), sha256.Sum256([]byte(value))
			if d.APIToken == "" || !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || !hmac.Equal(expected[:], actual[:]) {
				w.Header().Set("WWW-Authenticate", `Bearer realm="CortexMind local API"`)
				writeError(w, fmt.Errorf("open CortexMind from the desktop app or the authenticated link opened by cortexd"), http.StatusUnauthorized)
				return
			}
		}
		if r.URL.Path == "/api/cortex/reset" {
			d.operations.Lock()
			defer d.operations.Unlock()
		} else {
			d.operations.RLock()
			defer d.operations.RUnlock()
		}
		next.ServeHTTP(w, r)
	})
}

func (d *Daemon) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	challenge := r.URL.Query().Get("challenge")
	if len(challenge) > 128 {
		writeError(w, fmt.Errorf("invalid challenge"), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"app": "cortexMind", "ready": true, "proof": localauth.Proof(d.APIToken, challenge)})
}

func (d *Daemon) status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	uptime := int64(0)
	if !d.started.IsZero() {
		uptime = int64(time.Since(d.started).Seconds())
	}
	writeJSON(w, http.StatusOK, map[string]any{"ready": true, "version": "0.1.0", "uptimeSeconds": uptime, "watcherRunning": false, "semanticEnabled": false})
}
