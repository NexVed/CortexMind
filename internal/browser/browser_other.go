//go:build !windows

package browser

import "fmt"

func openWindowsURL(string) error {
	return fmt.Errorf("Windows browser launcher is unavailable")
}
