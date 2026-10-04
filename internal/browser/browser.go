// Package browser opens GitHub login pages with the operating system's
// configured default browser. The CortexMind UI itself always stays in the
// native desktop window and is never launched in a browser.
package browser

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// OpenGitHubLogin opens a GitHub device-flow verification page in the user's
// default browser. Local CortexMind URLs and any other site are rejected so the
// rest of the app cannot be pushed out of the native window.
func OpenGitHubLogin(rawURL string) error {
	if err := validateGitHubLoginURL(rawURL); err != nil {
		return err
	}
	return openDefaultBrowser(rawURL)
}

func validateGitHubLoginURL(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("invalid GitHub login URL")
	}
	host := strings.ToLower(u.Hostname())
	if host != "github.com" && host != "www.github.com" {
		return fmt.Errorf("refusing to open a non-GitHub URL in the browser")
	}
	path := strings.TrimSuffix(u.EscapedPath(), "/")
	if path != "/login/device" && !strings.HasPrefix(path, "/login/device/") {
		return fmt.Errorf("refusing to open a non-login GitHub URL in the browser")
	}
	return nil
}

func openDefaultBrowser(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return fmt.Errorf("invalid browser URL")
	}
	if isLoopbackHost(u.Hostname()) {
		return fmt.Errorf("refusing to open the CortexMind UI in a browser")
	}

	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		return openWindowsURL(rawURL)
	case "darwin":
		command = exec.Command("open", rawURL)
	case "linux":
		command = linuxOpenCommand(rawURL)
	default:
		return fmt.Errorf("opening a browser is not supported on %s", runtime.GOOS)
	}

	if err := command.Start(); err != nil {
		return fmt.Errorf("open default browser: %w", err)
	}
	go func() { _ = command.Wait() }()
	return nil
}

func isLoopbackHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

// linuxOpenCommand launches the host xdg-open with AppImage library overrides
// stripped so GitHub opens in the user's configured browser, not a bundled one.
func linuxOpenCommand(rawURL string) *exec.Cmd {
	bin := "xdg-open"
	if _, err := os.Stat("/usr/bin/xdg-open"); err == nil {
		bin = "/usr/bin/xdg-open"
	}
	cmd := exec.Command(bin, rawURL)
	cmd.Env = sanitizedLinuxEnv(os.Environ())
	return cmd
}

func sanitizedLinuxEnv(environ []string) []string {
	drop := map[string]struct{}{
		"LD_LIBRARY_PATH":          {},
		"LD_PRELOAD":               {},
		"PYTHONPATH":               {},
		"PYTHONHOME":               {},
		"PERLLIB":                  {},
		"APPDIR":                   {},
		"APPIMAGE":                 {},
		"ARGV0":                    {},
		"APPIMAGE_EXTRACT_AND_RUN": {},
	}
	out := make([]string, 0, len(environ))
	for _, entry := range environ {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, skip := drop[key]; skip {
			continue
		}
		out = append(out, entry)
	}
	return out
}
