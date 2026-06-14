package watcher

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/NexVed/Cortex/internal/scanner"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

// Watcher monitors registered repositories for file changes and triggers
// incremental re-scans with debouncing.
type Watcher struct {
	scanner *scanner.Scanner
	fs      *fsnotify.Watcher

	mu       sync.Mutex
	projects map[string]string // repoPath -> projectID
	owners   map[string]string // repoPath -> ownerID
	debounce map[string]*time.Timer

	stop chan struct{}
}

func New(sc *scanner.Scanner) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		scanner:  sc,
		fs:       fw,
		projects: map[string]string{},
		owners:   map[string]string{},
		debounce: map[string]*time.Timer{},
		stop:     make(chan struct{}),
	}, nil
}

// Add registers a repository tree for watching.
func (w *Watcher) Add(projectID, ownerID, repoPath string) error {
	w.mu.Lock()
	w.projects[repoPath] = projectID
	w.owners[repoPath] = ownerID
	w.mu.Unlock()

	return filepath.Walk(repoPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil || !fi.IsDir() {
			return nil
		}
		name := fi.Name()
		if name != "." && strings.HasPrefix(name, ".") && name != ".cortex" {
			return filepath.SkipDir
		}
		if name == "node_modules" || name == "vendor" || name == "target" || name == "dist" || name == "build" {
			return filepath.SkipDir
		}
		return w.fs.Add(path)
	})
}

// Start begins the event loop. Call in a goroutine.
func (w *Watcher) Start() {
	log.Info().Msg("file watcher started")
	for {
		select {
		case <-w.stop:
			return
		case ev, ok := <-w.fs.Events:
			if !ok {
				return
			}
			w.handle(ev)
		case err, ok := <-w.fs.Errors:
			if !ok {
				return
			}
			log.Warn().Err(err).Msg("watcher error")
		}
	}
}

func (w *Watcher) handle(ev fsnotify.Event) {
	if scanner.DetectLanguage(ev.Name) == "" {
		return
	}
	repoPath, projectID, ownerID := w.lookup(ev.Name)
	if projectID == "" {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if t, ok := w.debounce[ev.Name]; ok {
		t.Stop()
	}
	// Debounce 500ms — editors emit multiple write events per save.
	w.debounce[ev.Name] = time.AfterFunc(500*time.Millisecond, func() {
		_, err := w.scanner.Run(scanner.Job{
			ProjectID: projectID,
			RepoPath:  repoPath,
			OwnerID:   ownerID,
			FullScan:  false,
		})
		if err != nil {
			log.Warn().Err(err).Str("repo", repoPath).Msg("incremental scan failed")
		}
	})
}

func (w *Watcher) lookup(changed string) (repoPath, projectID, ownerID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for rp, pid := range w.projects {
		if strings.HasPrefix(changed, rp) {
			return rp, pid, w.owners[rp]
		}
	}
	return "", "", ""
}

// Stop terminates the watcher.
func (w *Watcher) Stop() {
	close(w.stop)
	_ = w.fs.Close()
}
