package auth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	gh "github.com/NexVed/Cortex/internal/github"
	"github.com/NexVed/Cortex/internal/services"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDeviceFlowCanBeCancelledDuringNetworkRequest(t *testing.T) {
	entered := make(chan struct{})
	client := gh.Client{ClientID: "test", HTTP: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/login/device/code" {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"device_code":"device","user_code":"code","verification_uri":"https://github.com/login/device","expires_in":900,"interval":5}`))}, nil
		}
		close(entered)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}}
	s := &Service{ClientID: "test", GitHub: services.Onboarding{GitHub: client}}
	if _, err := s.StartGitHub(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("poll did not start")
	}
	done := make(chan struct{})
	go func() { s.CancelGitHub(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancel did not interrupt the pending HTTP request")
	}
	if s.active || s.AuthenticationError() != "" {
		t.Fatal("cancelled flow remained active or displayed an error")
	}
}
