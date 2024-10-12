package regolith

import (
	"path/filepath"
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
	watcher           *fsnotify.Watcher
	recursiveRoot     string
	kind              string           // Whether the watched directory is "RP", "BP", or "data".
	debounce          <-chan time.Time // Debounce timer
	interruption      chan string      // See RunContext.
	errors            chan error       // See RunContext.
	shouldRestartData chan struct{}
}

// NewDirWatcher creates a new directory watcher.
func NewDirWatcher(
	root string,
	kind string,
	interruption chan string,
	errors chan error,
	shouldRestartData chan struct{},
) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return burrito.WrapError(err, "Could not initialize directory watching")
	}

	d := &DirWatcher{
		watcher: watcher,
		// We have to manually signal to fsnotify that it should recursively watch this
		// path by using "/..." or "\...".
		recursiveRoot:     filepath.Join(root, "..."),
		kind:              kind,
		interruption:      interruption,
		errors:            errors,
		shouldRestartData: shouldRestartData,
	}

	if err := d.watcher.Add(d.recursiveRoot); err != nil {
		return burrito.WrapErrorf(err, "Could not start watching `%s`", root)
	}
	go d.start()
	return nil
}

// Start starts the file watching loop and blocks the goroutine until it
// receives either:
//   - an event, in which it sends a message to interruption channel then
//     creates a debounce timer of 100ms. In this duration, all events are
//     silently ignored.
//   - an restart channel signal, and if the watcher is for the "data"
//     folder, it will restart watching the folder again.
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
			if d.debounce != nil || event.Op.Has(fsnotify.Chmod) {
				continue
			}
			d.interruption <- d.kind
			d.debounce = time.After(100 * time.Millisecond)
		case <-d.debounce:
			d.debounce = nil
		case <-d.shouldRestartData:
			if d.kind == "rp" || d.kind == "bp" {
				// Basically "forward" the signal to another listener until it reaches the
				// one for "data".
				d.shouldRestartData <- struct{}{}
				continue
			}
			d.watcher.Close()
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				d.errors <- burrito.WrapError(err, "Could not begin restarting file watching for data folder")
			}
			d.watcher = watcher
			if err := d.watcher.Add(d.recursiveRoot); err != nil {
				d.errors <- burrito.WrapErrorf(err, "Could not start watching the data folder")
			}
		}
	}
}
