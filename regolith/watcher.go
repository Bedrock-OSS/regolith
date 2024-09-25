package regolith

import (
	"path/filepath"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"github.com/arexon/fsnotify"
)

// DirWatcher handles watching for changes in a single directory (e.g. RP).
//
// fsnotify doesn't *officially* support recursive file watching yet. Windows
// and and Linux are supported, but not macOS. For now, this implementation uses
// a custom fork with patches to manually enable it.
//
// Fork patch: https://github.com/arexon/fsnotify/blob/main/fsnotify.go#L481
type DirWatcher struct {
	watcher      *fsnotify.Watcher
	root         string
	kind         string      // Whether the watched directory is "RP", "BP", or "data".
	interruption chan string // see RunContext
	errors       chan error  // see RunContext
}

// NewDirWatcher creates a new pack watcher for the given pack kind.
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
		watcher,
		root,
		kind,
		interruption,
		errors,
	}
	path := filepath.Join(root, "...")
	if err := d.watcher.Add(path); err != nil {
		return burrito.WrapErrorf(err, "Could not start watching `%f`", root)
	}
	go d.start()
	return nil
}

// Start starts the file watching loop and blocks the goroutine until it
// receives an event. Once it does, it sends a message to interruptionChannel
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
		case _, ok := <-d.watcher.Events:
			if !ok {
				return
			}
			d.interruption <- d.kind
		}
	}
}
