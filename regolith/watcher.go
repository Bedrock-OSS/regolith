package regolith

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"github.com/arexon/fsnotify"
)

// DirWatcher handles watching for changes in a multiple root directories.
//
// fsnotify doesn't *officially* support recursive file watching yet. Windows
// and and Linux are supported, but not macOS. For now, this implementation uses
// a custom fork with patches to manually enable it.
//
// Fork patch: https://github.com/arexon/fsnotify/blob/main/fsnotify.go#L481
type DirWatcher struct {
	// watcher is the underlying fsnotify watcher that notifies about the
	// changes in the files.
	watcher *fsnotify.Watcher

	// roots is a list of directories to watch.
	// TODO: Currently, all of the information needed to determine the roots
	// is already stored in the config. Having this as separate field is
	// redundant and should be removed to have a single source of truth.
	roots []string

	// config is a reference to the configuration of the project.
	config *Config

	debounce *time.Timer
	// interruption channel is used to notify the main thread about the kind of
	// interruption that was detected by 'watcher', it can be 'rp', 'bp' or 'data'.
	interruption chan string

	// errors is a channel used by DirWatcher to inform the main thread about
	// errors related to watching files.
	errors chan error

	// stage is a channel used for receiving commands from the main thread, to
	// pause or restart the watcher.
	stage <-chan string
}

func NewDirWatcher(
	config *Config,
	interruption chan string,
	errors chan error,
	stage <-chan string,
) error {
	var roots []string
	if config.ResourceFolder != "" {
		roots = append(roots, config.ResourceFolder)
	}
	if config.BehaviorFolder != "" {
		roots = append(roots, config.BehaviorFolder)
	}
	if config.DataPath != "" {
		roots = append(roots, config.DataPath)
	}
	d := &DirWatcher{
		roots:        roots,
		config:       config,
		interruption: interruption,
		errors:       errors,
		stage:        stage,
	}
	err := d.watch()
	if err != nil {
		return err
	}
	go d.start()
	return nil
}

func (d *DirWatcher) watch() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return burrito.WrapError(err, "Could not initialize directory watching")
	}
	d.watcher = watcher
	for _, root := range d.roots {
		// We have to manually signal to fsnotify that it should recursively watch this
		// path by using "/..." or "\...".
		recursiveRoot := filepath.Join(root, "...")
		if err := d.watcher.Add(recursiveRoot); err != nil {
			return burrito.WrapErrorf(err, "Could not start watching `%s`", root)
		}
	}
	return nil
}

func (d *DirWatcher) start() {
	paused := false
	for {
		var debounce <-chan time.Time
		if d.debounce != nil {
			debounce = d.debounce.C
		}
		select {
		case err, ok := <-d.watcher.Errors:
			if !ok {
				if paused {
					continue
				}
				return
			}
			d.errors <- err
			return
		case event, ok := <-d.watcher.Events:
			if !ok {
				if paused {
					continue
				}
				return
			}
			if d.debounce != nil || event.Op.Has(fsnotify.Chmod) {
				continue
			}
			if isInDir(event.Name, d.config.ResourceFolder) {
				d.interruption <- "rp"
			} else if isInDir(event.Name, d.config.BehaviorFolder) {
				d.interruption <- "bp"
			} else if isInDir(event.Name, d.config.DataPath) {
				d.interruption <- "data"
			}
			if d.debounce == nil {
				d.debounce = time.NewTimer(100 * time.Millisecond)
			} else {
				d.debounce.Reset(100 * time.Millisecond)
			}
		case <-debounce:
			if d.debounce != nil {
				d.debounce.Stop()
				d.debounce = nil
			}
		case stage := <-d.stage:
			switch stage {
			case "pause":
				d.watcher.Close()
				paused = true
			case "restart":
				if err := d.watch(); err != nil {
					d.errors <- err
				}
				paused = false
			}
		}
	}
}

func isInDir(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && !strings.HasPrefix(rel, "..")
}
