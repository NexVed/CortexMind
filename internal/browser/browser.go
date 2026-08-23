// Package browser opens trusted URLs with the operating system's configured
// default browser.
package browser

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

// OpenURL opens an absolute HTTP(S) URL with the user's default browser.
func OpenURL(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return fmt.Errorf("invalid browser URL")
	}

	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	case "darwin":
		command = exec.Command("open", rawURL)
	case "linux":
		command = exec.Command("xdg-open", rawURL)
	default:
		return fmt.Errorf("opening a browser is not supported on %s", runtime.GOOS)
	}

	if err := command.Start(); err != nil {
		return fmt.Errorf("open default browser: %w", err)
	}
	go func() { _ = command.Wait() }()
	return nil
}
