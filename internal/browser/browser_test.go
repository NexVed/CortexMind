package browser

import "testing"

func TestValidateGitHubLoginURL(t *testing.T) {
	ok := []string{
		"https://github.com/login/device",
		"https://github.com/login/device?user_code=ABCD-1234",
		"https://www.github.com/login/device/",
	}
	for _, raw := range ok {
		if err := validateGitHubLoginURL(raw); err != nil {
			t.Errorf("%s: unexpected error: %v", raw, err)
		}
	}

	blocked := []string{
		"http://127.0.0.1:8090",
		"http://localhost:8090/?desktop=1",
		"https://github.com/NexVed/Cortex",
		"https://github.com/login/oauth/authorize",
		"https://example.com/login/device",
		"not-a-url",
	}
	for _, raw := range blocked {
		if err := validateGitHubLoginURL(raw); err == nil {
			t.Errorf("%s: expected rejection", raw)
		}
	}
}

func TestIsLoopbackHost(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "localhost", "::1", "LocalHost"} {
		if !isLoopbackHost(host) {
			t.Errorf("%s should be treated as loopback", host)
		}
	}
	if isLoopbackHost("github.com") {
		t.Fatal("github.com must not be treated as loopback")
	}
}
