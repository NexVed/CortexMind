package daemon

import (
	"net/http"
	"time"

	cortexapi "github.com/NexVed/Cortex/internal/api"
	"github.com/NexVed/Cortex/internal/auth"
	"github.com/NexVed/Cortex/internal/config"
	"github.com/NexVed/Cortex/internal/db"
	cgit "github.com/NexVed/Cortex/internal/git"
	"github.com/NexVed/Cortex/internal/mcp"
	"github.com/NexVed/Cortex/internal/rpc"
	"github.com/NexVed/Cortex/internal/scanner"
	"github.com/NexVed/Cortex/internal/search"
	"github.com/NexVed/Cortex/internal/vector"
	"github.com/NexVed/Cortex/internal/watcher"
	"github.com/NexVed/Cortex/internal/web"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog/log"
)

// Daemon owns the PocketBase application and all CORTEX subsystems.
type Daemon struct {
	App     *pocketbase.PocketBase
	Config  *config.Config
	Scanner *scanner.Scanner
	Watcher *watcher.Watcher
	Search  *search.Engine

	watcherRunning bool
}

// New constructs a daemon and registers all PocketBase hooks. Call App.Start()
// afterwards to run the server.
func New(cfg *config.Config) *Daemon {
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: cfg.DataDirPath(),
	})

	d := &Daemon{App: app, Config: cfg}

	d.Scanner = scanner.New(app, &cfg.Scanner)
	d.Search = search.New(app)

	auth.RegisterOAuthHooks(app)
	d.registerHooks()
	return d
}

func (d *Daemon) registerHooks() {
	app := d.App

	// Ensure collections exist once the database is bootstrapped.
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		if err := db.EnsureCollections(app); err != nil {
			log.Error().Err(err).Msg("failed to ensure collections")
			return err
		}
		log.Info().Msg("collections ready")
		return nil
	})

	// Register RPC handlers and start subsystems when the HTTP server boots.
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		d.mountRPC(se)
		d.startWatcher()
		log.Info().
			Int("port", d.Config.Server.Port).
			Msg("CORTEX daemon ready")
		return se.Next()
	})
}

func (d *Daemon) mountRPC(se *core.ServeEvent) {
	var embedder vector.Embedder
	var store vector.Store
	if d.Config.Search.EnableSemantic {
		embedder = vector.NewOllamaEmbedder(d.Config.Search.OllamaURL, d.Config.Search.EmbeddingModel)
		if d.Config.Search.VectorDBURL != "" {
			store = vector.NewLanceStore(d.Config.Search.VectorDBURL)
		}
	}

	handlers := &rpc.Handlers{
		App:            d.App,
		Scanner:        d.Scanner,
		SearchEngine:   d.Search,
		Embedder:       embedder,
		Vector:         store,
		Started:        time.Now(),
		WatcherRunning: func() bool { return d.watcherRunning },
	}

	handlers.Mount(func(pattern string, handler http.Handler) {
		se.Router.Any(pattern+"{path...}", apis.WrapStdHandler(handler))
		log.Info().Str("rpc", pattern).Msg("mounted service")
	})

	// CORTEX higher-level JSON routes (GitHub-wide scan, providers, graph).
	apiSvc := &cortexapi.Service{
		App:     d.App,
		Scanner: d.Scanner,
		Config:  d.Config,
		Git:     cgit.NewSyncEngine(),
		Vector:  store,
	}
	apiSvc.RegisterRoutes(se)
	log.Info().Msg("mounted cortex api routes")

	// MCP server for IDE connections (Streamable HTTP / JSON-RPC).
	mcpServer := mcp.New(d.App)
	se.Router.Any("/mcp", apis.WrapStdHandler(mcpServer.Handler()))
	log.Info().Msg("mounted MCP endpoint at /mcp")

	// Serve the embedded SolidJS UI (SPA) for all remaining routes. Registered
	// last, and as a catch-all, so PocketBase's own /api, /_/ and the routes
	// above keep priority; unmatched paths fall back to index.html.
	if web.Available() {
		se.Router.Any("/{path...}", apis.Static(web.FS(), true))
		log.Info().Msg("mounted embedded UI at /")
	} else {
		log.Warn().Msg("no embedded UI found (run the UI build); serving API only")
	}
}

func (d *Daemon) startWatcher() {
	w, err := watcher.New(d.Scanner)
	if err != nil {
		log.Error().Err(err).Msg("failed to create watcher")
		return
	}
	d.Watcher = w
	d.watcherRunning = true

	go func() {
		projects, err := d.App.FindRecordsByFilter(db.CollProjects, "path != ''", "", 500, 0, nil)
		if err != nil {
			log.Warn().Err(err).Msg("could not list projects for watching")
			return
		}
		for _, p := range projects {
			path := p.GetString("path")
			log.Info().Str("path", path).Msg("registering watcher for project path")
			if err := w.Add(p.Id, p.GetString("owner"), path); err != nil {
				log.Warn().Err(err).Str("path", path).Msg("failed to watch project")
			}
		}
		w.Start()
	}()
}
