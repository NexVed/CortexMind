package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	gh "github.com/NexVed/Cortex/internal/github"
	"github.com/NexVed/Cortex/internal/keychain"
	"github.com/NexVed/Cortex/internal/services"
)

type Service struct {
	ClientID string
	GitHub   services.Onboarding
	Tokens   keychain.TokenStore

	mu      sync.Mutex
	active  bool
	lastErr string
}

type StartResult struct {
	URL       string `json:"url"`
	UserCode  string `json:"user_code"`
	ExpiresIn int    `json:"expires_in"`
	Interval  int    `json:"interval"`
}

// StartGitHub begins the OAuth Device Flow. The desktop client displays the
// returned code while this service polls GitHub in the background.
func (s *Service) StartGitHub() (StartResult, error) {
	if s.ClientID == "" {
		return StartResult{}, fmt.Errorf("GitHub OAuth is not configured in this build")
	}

	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return StartResult{}, fmt.Errorf("GitHub authorization is already in progress")
	}
	s.active = true
	s.lastErr = ""
	s.mu.Unlock()

	device, err := s.GitHub.GitHub.StartDeviceFlow(context.Background())
	if err != nil {
		s.finish(err)
		return StartResult{}, err
	}

	go s.completeDeviceFlow(device)
	url := device.VerificationURIComplete
	if url == "" {
		url = device.VerificationURI
	}
	return StartResult{
		URL:       url,
		UserCode:  device.UserCode,
		ExpiresIn: device.ExpiresIn,
		Interval:  device.Interval,
	}, nil
}

func (s *Service) completeDeviceFlow(device gh.DeviceCode) {
	interval := time.Duration(device.Interval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(device.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		token, err := s.GitHub.GitHub.ExchangeDeviceCode(context.Background(), device.DeviceCode)
		if err == nil {
			s.finish(s.saveGitHubToken(token))
			return
		}

		var flowErr *gh.DeviceFlowError
		if !errors.As(err, &flowErr) {
			s.finish(err)
			return
		}
		switch flowErr.Code {
		case "authorization_pending":
			// The user has not confirmed the code yet.
		case "slow_down":
			interval += 5 * time.Second
		default:
			s.finish(flowErr)
			return
		}

		time.Sleep(interval)
	}
	s.finish(fmt.Errorf("GitHub device authorization expired; please try again"))
}

func (s *Service) saveGitHubToken(token string) error {
	u, err := s.GitHub.CompleteGitHub(context.Background(), token)
	if err != nil {
		return err
	}
	return s.Tokens.Set("github:"+u.ID, token)
}

func (s *Service) finish(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = false
	if err == nil {
		s.lastErr = ""
		return
	}
	s.lastErr = err.Error()
}

// AuthenticationError returns a terminal device-flow error, if one occurred.
func (s *Service) AuthenticationError() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastErr
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
