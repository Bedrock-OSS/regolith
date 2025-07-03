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
	watcher      *fsnotify.Watcher
	roots        []string
	config       *Config
	debounce     *time.Timer
	interruption chan string
	errors       chan error
	stage        <-chan string
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

func isInDir(path string, root string) bool {
	return strings.HasPrefix(path, filepath.Clean(root))
}
