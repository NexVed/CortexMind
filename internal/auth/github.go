package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/NexVed/Cortex/internal/keychain"
	"github.com/NexVed/Cortex/internal/oauth"
	"github.com/NexVed/Cortex/internal/services"
)

type Service struct {
	ClientID string
	GitHub   services.Onboarding
	Tokens   keychain.TokenStore
	mu       sync.Mutex
	active   bool
}
type StartResult struct {
	URL string `json:"url"`
}

// StartGitHub creates a short-lived loopback callback server before returning
// the authorization URL. State and PKCE are held only in process memory.
func (s *Service) StartGitHub() (StartResult, error) {
	if s.ClientID == "" {
		return StartResult{}, fmt.Errorf("GitHub OAuth is not configured in this build")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		return StartResult{}, fmt.Errorf("GitHub authorization is already in progress")
	}
	s.active = true
	verifier, challenge, err := oauth.PKCE()
	if err != nil {
		s.active = false
		return StartResult{}, err
	}
	state, err := oauth.State()
	if err != nil {
		s.active = false
		return StartResult{}, err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		s.active = false
		return StartResult{}, err
	}
	redirect := "http://" + listener.Addr().String() + "/oauth/github/callback"
	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}
	finish := func() { s.mu.Lock(); s.active = false; s.mu.Unlock(); go server.Close() }
	mux.HandleFunc("/oauth/github/callback", func(w http.ResponseWriter, r *http.Request) {
		defer finish()
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
			return
		}
		if e := r.URL.Query().Get("error"); e != "" {
			http.Error(w, "GitHub authorization was declined: "+e, http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing authorization code", http.StatusBadRequest)
			return
		}
		token, err := s.GitHub.GitHub.Exchange(r.Context(), code, redirect, verifier)
		if err == nil {
			u, completeErr := s.GitHub.CompleteGitHub(context.Background(), token)
			err = completeErr
			if err == nil {
				err = s.Tokens.Set("github:"+u.ID, token)
			}
		}
		if err != nil {
			http.Error(w, "Unable to finish sign-in. Return to cortexMind and try again.", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<html><head><title>CortexMind connected</title></head><body><h2>CortexMind is connected to GitHub.</h2><p>This window closes automatically. Return to CortexMind.</p><script>window.close();</script></body></html>")
	})
	go server.Serve(listener)
	return StartResult{URL: s.GitHub.GitHub.AuthorizationURL(redirect, state, challenge)}, nil
}
func (s *Service) CurrentGitHubToken() (string, error) {
	u, err := s.GitHub.Users.Current()
	if err != nil {
		return "", err
	}
	if u == nil || u.Provider != "github" {
		return "", fmt.Errorf("GitHub is not connected")
	}
	return s.Tokens.Get("github:" + u.ID)
}

func (s *Service) Logout() error {
	u, err := s.GitHub.Users.Current()
	if err != nil {
		return err
	}
	if u != nil && u.Provider == "github" {
		_ = s.Tokens.Delete("github:" + u.ID)
	}
	return s.GitHub.DB.ClearActiveUser()
}
