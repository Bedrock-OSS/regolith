package regolith

import (
	"math"
	"path/filepath"
	"sync"
	"time"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"github.com/arexon/fsnotify"
)

// DirWatcher handles watching for changes in a specific directory (e.g. RP).
//
// fsnotify doesn't *officially* support recursive file watching yet. Windows
// and and Linux are supported, but not macOS. For now, this implementation uses
// a custom fork with patches to manually enable it.
//
// Fork patch: https://github.com/arexon/fsnotify/blob/main/fsnotify.go#L481
type DirWatcher struct {
	watcher      *fsnotify.Watcher
	root         string
	kind         string                 // Whether the watched directory is "RP", "BP", or "data".
	mu           sync.Mutex             // Used for locking during deboucning.
	timers       map[string]*time.Timer // Stores event path -> debounce timer.
	interruption chan string            // See RunContext.
	errors       chan error             // See RunContext.
}

// NewDirWatcher creates a new directory watcher.
func NewDirWatcher(
	root string,
	kind string,
	interruption chan string,
	errors chan error,
) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return burrito.WrapError(err, "Could not initialize directory watching")
	}

	d := &DirWatcher{
		watcher:      watcher,
		root:         root,
		kind:         kind,
		timers:       make(map[string]*time.Timer),
		interruption: interruption,
		errors:       errors,
	}

	// We have to manually signal to fsnotify that it should recursively watch this
	// path by using "/..." or "\...".
	recursiveRoot := filepath.Join(root, "...")
	if err := d.watcher.Add(recursiveRoot); err != nil {
		return burrito.WrapErrorf(err, "Could not start watching `%f`", root)
	}
	go d.start()
	return nil
}

// Start starts the file watching loop and blocks the goroutine until it
// receives an event. Once it does, it sends a message to interruption channel
// then resumes blocking the goroutine.
func (d *DirWatcher) start() {
	for {
		select {
		case err, ok := <-d.watcher.Errors:
			if !ok {
				return
			}
			d.errors <- err
			return
		case event, ok := <-d.watcher.Events:
			if !ok {
				return
			}
			if event.Op.Has(fsnotify.Chmod) {
				continue
			}

			d.mu.Lock()
			timer, exists := d.timers[event.Name]
			d.mu.Unlock()

			if !exists {
				timer = time.AfterFunc(math.MaxInt64, func() { d.interruption <- d.kind })
				timer.Stop()

				d.mu.Lock()
				d.timers[event.Name] = timer
				d.mu.Unlock()
			}

			timer.Reset(100 * time.Millisecond)
		}
	}
}
