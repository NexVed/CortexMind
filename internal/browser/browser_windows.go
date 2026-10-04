//go:build windows

package browser

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// ShellExecute uses the user's Windows HTTPS association, which opens the
// configured default browser without creating another CortexMind webview.
func openWindowsURL(rawURL string) error {
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(rawURL)
	if err != nil {
		return err
	}
	if err := windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("open default browser: %w", err)
	}
	return nil
}
