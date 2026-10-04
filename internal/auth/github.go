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
	cancel  context.CancelFunc
	done    chan struct{}
}

type StartResult struct {
	URL          string `json:"url"`
	UserCode     string `json:"user_code"`
	ExpiresIn    int    `json:"expires_in"`
	Interval     int    `json:"interval"`
	BrowserError string `json:"browser_error,omitempty"`
}

// StartGitHub begins the OAuth Device Flow. The desktop client displays the
// returned code while this service polls GitHub in the background.
func (s *Service) StartGitHub(ctx context.Context) (StartResult, error) {
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
	flowCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	s.cancel = cancel
	s.done = make(chan struct{})
	s.mu.Unlock()

	startCtx, cancelStart := context.WithCancel(ctx)
	stop := context.AfterFunc(flowCtx, cancelStart)
	defer stop()
	defer cancelStart()
	device, err := s.GitHub.GitHub.StartDeviceFlow(startCtx)
	if err != nil {
		s.finish(err)
		return StartResult{}, err
	}

	go s.completeDeviceFlow(flowCtx, device)
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

func (s *Service) completeDeviceFlow(ctx context.Context, device gh.DeviceCode) {
	interval := time.Duration(device.Interval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(device.ExpiresIn) * time.Second)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	for time.Now().Before(deadline) {
		token, err := s.GitHub.GitHub.ExchangeDeviceCode(ctx, device.DeviceCode)
		if err == nil {
			s.finish(s.saveGitHubToken(ctx, token))
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

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			s.finish(ctx.Err())
			return
		case <-timer.C:
		}
	}
	s.finish(fmt.Errorf("GitHub device authorization expired; please try again"))
}

func (s *Service) saveGitHubToken(ctx context.Context, token string) error {
	u, err := s.GitHub.CompleteGitHub(ctx, token)
	if err != nil {
		return err
	}
	if err = ctx.Err(); err == nil {
		err = s.Tokens.Set("github:"+u.ID, token)
	}
	if err != nil {
		_ = s.GitHub.DB.ClearActiveUser()
	}
	return err
}

func (s *Service) finish(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = false
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if s.done != nil {
		close(s.done)
		s.done = nil
	}
	if err == nil || errors.Is(err, context.Canceled) {
		s.lastErr = ""
		return
	}
	s.lastErr = err.Error()
}

// CancelGitHub waits for the in-flight flow before logout or reset can clear data.
func (s *Service) CancelGitHub() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
	s.mu.Lock()
	s.lastErr = ""
	s.mu.Unlock()
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
	s.CancelGitHub()
	u, err := s.GitHub.Users.Current()
	if err != nil {
		return err
	}
	if u != nil && u.Provider == "github" {
		if err := s.Tokens.Delete("github:" + u.ID); err != nil {
			return err
		}
	}
	return s.GitHub.DB.ClearActiveUser()
}
