// Command cortexd runs the local cortexMind backend without the desktop shell.
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/NexVed/Cortex/internal/config"
	"github.com/NexVed/Cortex/internal/daemon"
	"github.com/NexVed/Cortex/internal/localauth"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.Load()
	setupLogging(cfg.LogLevel)
	noBrowser, err := parseArgs(cfg, os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	d, err := daemon.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("initialize cortexd")
	}
	if !noBrowser {
		go openWhenReady(cfg.Server.Port, d.APIToken, cfg.Server.DevOrigin)
	}
	if err := d.Start(); err != nil {
		log.Fatal().Err(err).Msg("cortexd stopped")
	}
}
func parseArgs(cfg *config.Config, args []string) (bool, error) {
	if len(args) > 0 && args[0] == "serve" {
		args = args[1:]
	}
	flags := flag.NewFlagSet("cortexd [serve]", flag.ContinueOnError)
	addr := flags.String("http", fmt.Sprintf("127.0.0.1:%d", cfg.Server.Port), "loopback HTTP address")
	noBrowser := flags.Bool("no-browser", false, "do not open the authenticated UI link")
	if err := flags.Parse(args); err != nil {
		return false, err
	}
	if flags.NArg() != 0 {
		return false, fmt.Errorf("unknown command; use cortexd [serve] [--http 127.0.0.1:%d] [--no-browser]", cfg.Server.Port)
	}
	host, rawPort, err := net.SplitHostPort(*addr)
	if err != nil {
		return false, fmt.Errorf("invalid --http address: %w", err)
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil || port < 1 || port > 65535 || (host != "127.0.0.1" && host != "localhost") {
		return false, fmt.Errorf("--http must use 127.0.0.1 or localhost with a port between 1 and 65535")
	}
	cfg.Server.Port = port
	return *noBrowser, nil
}

func openWhenReady(port int, token, devOrigin string) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(300 * time.Millisecond) {
		if localauth.Alive(addr, token) {
			openBrowser(localauth.LaunchURL(addr, token, devOrigin))
			return
		}
	}
}
func openBrowser(target string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	_ = cmd.Start()
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
