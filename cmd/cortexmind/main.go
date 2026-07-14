// Command cortexmind is the native desktop shell for CortexMind.
//
// It is a complete, self-contained desktop app: it embeds and boots the same
// CORTEX daemon that cmd/cortexd runs (PocketBase serving the SolidJS UI + API
// on 127.0.0.1) and opens a native Wails v3 webview window pointed at it — no
// browser, no separately-run backend.
//
// Robustness:
//   - If a CortexMind backend is already listening (a previous instance, or a
//     manually-run cortexd), it is reused instead of starting a second daemon
//     that would fight over the same SQLite database.
//   - A single-instance guard focuses the existing window on relaunch.
//   - Backend startup failures show a dialog / log instead of crashing.
//
// Because Wails uses the OS-native webview (WebView2 on Windows, WebKitGTK on
// Linux, WebKit on macOS), this binary is NOT cross-compilable like cmd/cortexd:
// build it on each target OS (see build/ scripts). cmd/cortexd stays the
// pure-Go, cross-compilable headless daemon.
package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/NexVed/Cortex/internal/config"
	"github.com/NexVed/Cortex/internal/daemon"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// singleInstanceKey scopes the single-instance guard to this app.
const singleInstanceID = "com.nexved.cortexmind"

var singleInstanceKey = [32]byte{
	0xc0, 0x11, 0xe2, 0x7e, 0x33, 0x44, 0x55, 0x66,
	0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee,
	0x0f, 0x1e, 0x2d, 0x3c, 0x4b, 0x5a, 0x69, 0x78,
	0x87, 0x96, 0xa5, 0xb4, 0xc3, 0xd2, 0xe1, 0xf0,
}

func main() {
	cfg := config.Load()
	setupLogging(cfg.LogLevel)

	addr := fmt.Sprintf("127.0.0.1:%d", cfg.Server.Port)

	// Reuse an already-running backend if one is up; otherwise boot our own.
	// This is what makes launching a second copy (or having a manual cortexd
	// running) safe: we never start a second daemon against the same data dir.
	if !serverAlive(addr) {
		os.Args = []string{os.Args[0], "serve", "--http", addr}
		d := daemon.New(cfg)
		errCh := make(chan error, 1)
		go func() { errCh <- d.App.Start() }()

		if err := waitForServer(addr, 25*time.Second, errCh); err != nil {
			fatal(cfg, "CortexMind could not start its backend.", err)
			return
		}
	}

	var window *application.WebviewWindow
	app := application.New(application.Options{
		Name:        "CortexMind",
		Description: "The Shared Brain For AI Development",
		Mac: application.MacOptions{
			// Single-window utility: quit when the window is closed.
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID:      singleInstanceID,
			EncryptionKey: singleInstanceKey,
			OnSecondInstanceLaunch: func(_ application.SecondInstanceData) {
				if window != nil {
					window.Restore()
					window.Focus()
				}
			},
		},
	})

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "CortexMind",
		Width:            1400,
		Height:           900,
		MinWidth:         960,
		MinHeight:        600,
		URL:              "http://" + addr,
		BackgroundColour: application.NewRGB(13, 17, 23),
		// Frameless: the UI draws its own titlebar (WindowTitleBar.tsx). On
		// Windows 11 frameless windows still get DWM rounded corners + shadow.
		Frameless: true,
		// Let the remote-origin UI drive the native window via bare Wails events
		// (window._wails.invoke("wails:event:emit:<name>")). The full JS runtime
		// isn't injected on a remote URL, but this minimal bridge is.
		AllowSimpleEventEmit: true,
		Windows: application.WindowsWindow{
			// Honour CSS `app-region: drag` on the custom titlebar so the
			// frameless window can still be moved.
			NonClientRegionSupport: true,
		},
	})

	// Window controls emitted by the custom titlebar.
	maximised := false
	app.Event.On("wnd:minimise", func(*application.CustomEvent) { window.Minimise() })
	app.Event.On("wnd:toggle-maximise", func(*application.CustomEvent) {
		if maximised {
			window.Restore()
		} else {
			window.Maximise()
		}
		maximised = !maximised
	})
	app.Event.On("wnd:close", func(*application.CustomEvent) { window.Close() })

	window.Show()

	if err := app.Run(); err != nil {
		log.Error().Err(err).Msg("wails app stopped")
	}
}

// waitForServer blocks until the daemon accepts connections on addr, the daemon
// reports a startup error, or the timeout elapses.
func waitForServer(addr string, timeout time.Duration, errCh <-chan error) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("backend exited during startup: %w", err)
			}
			return fmt.Errorf("backend exited during startup")
		default:
		}
		if serverAlive(addr) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("backend did not become ready within %s", timeout)
}

// serverAlive reports whether something is accepting TCP connections on addr.
// ponytail: a plain dial can't tell CORTEX apart from an unrelated process that
// happened to grab the port; on the fixed loopback CORTEX port this is a
// non-issue in practice. Upgrade path: probe GET / and check a response header.
func serverAlive(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// fatal surfaces a startup failure without crashing: it writes the reason to a
// log file in the data dir (the app has no console under -H windowsgui) and to
// stderr, then returns so main can exit cleanly.
func fatal(cfg *config.Config, msg string, err error) {
	log.Error().Err(err).Msg(msg)
	logPath := filepath.Join(cfg.DataDirPath(), "cortexmind-desktop.log")
	if f, ferr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); ferr == nil {
		fmt.Fprintf(f, "%s %s: %v\n", time.Now().Format(time.RFC3339), msg, err)
		_ = f.Close()
	}
}

func setupLogging(level string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}
