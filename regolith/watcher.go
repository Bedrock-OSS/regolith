package regolith

import (
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const watchDebounceDuration = 100 * time.Millisecond

type watchRoot struct {
	path   string
	source string
}

type watchBackend interface {
	Close() error
}

type stageRequest struct {
	action string
	done   chan error
}

// DirWatcher handles watching for changes in multiple root directories.
//
// On Windows, a dedicated native backend watches each root with a single
// subtree notification handle so Regolith never holds open handles on
// subdirectories. On Unix systems (Linux/macOS), the official fsnotify package
// is used with recursive directory traversal and dynamic subdirectory watching.
type DirWatcher struct {
	roots       []watchRoot
	backend     watchBackend
	events      chan string
	backendErrs chan error
	debounce    *time.Timer

	// interruption notifies the main thread about which watch root changed:
	// "rp", "bp", "data", or "extras".
	interruption chan string

	// errors passes watcher errors to the main thread.
	errors chan error

	// stage receives legacy "pause" / "restart" commands if used directly.
	stage <-chan string

	control   chan stageRequest
	done      chan struct{}
	closeOnce sync.Once
}

func collectWatchRoots(config *Config) []watchRoot {
	if config == nil {
		return nil
	}
	var roots []watchRoot
	if config.ResourceFolder != "" {
		roots = append(roots, watchRoot{path: config.ResourceFolder, source: "rp"})
	}
	if config.BehaviorFolder != "" {
		roots = append(roots, watchRoot{path: config.BehaviorFolder, source: "bp"})
	}
	if config.DataPath != "" {
		roots = append(roots, watchRoot{path: config.DataPath, source: "data"})
	}
	for _, path := range config.WatchPaths {
		if path != "" {
			roots = append(roots, watchRoot{path: path, source: "extras"})
		}
	}
	return roots
}

// NewDirWatcher starts watching the configured project directories and sends
// change sources to interruption and fatal watcher errors to errors.
func NewDirWatcher(
	config *Config,
	interruption chan string,
	errors chan error,
	stage <-chan string,
) error {
	_, err := newDirWatcher(config, interruption, errors, stage)
	return err
}

func newDirWatcher(
	config *Config,
	interruption chan string,
	errors chan error,
	stage <-chan string,
) (*DirWatcher, error) {
	d := &DirWatcher{
		roots:        collectWatchRoots(config),
		events:       make(chan string, 64),
		backendErrs:  make(chan error, 16),
		interruption: interruption,
		errors:       errors,
		stage:        stage,
		control:      make(chan stageRequest),
		done:         make(chan struct{}),
	}
	backend, err := startWatchBackend(d.roots, d.events, d.backendErrs)
	if err != nil {
		return nil, err
	}
	d.backend = backend
	go d.start()
	return d, nil
}

// Pause stops the underlying OS watcher and waits until all OS handles are
// closed and queued internal events are drained.
func (d *DirWatcher) Pause() error {
	return d.sendControl("pause")
}

// Restart recreates the underlying OS watcher after a pause.
func (d *DirWatcher) Restart() error {
	return d.sendControl("restart")
}

// Close permanently stops the DirWatcher and releases all OS handles.
func (d *DirWatcher) Close() error {
	var err error
	d.closeOnce.Do(func() {
		err = d.sendControl("close")
		<-d.done
	})
	return err
}

func (d *DirWatcher) sendControl(action string) error {
	doneCh := make(chan error, 1)
	req := stageRequest{action: action, done: doneCh}
	select {
	case d.control <- req:
	case <-d.done:
		return nil
	}
	select {
	case err := <-doneCh:
		return err
	case <-d.done:
		return nil
	}
}

func (d *DirWatcher) start() {
	defer close(d.done)

	paused := false
	pendingSource := ""
	var pendingErr error

	for {
		var outCh chan string
		if !paused && pendingSource != "" {
			outCh = d.interruption
		}
		var errOutCh chan error
		if pendingErr != nil {
			errOutCh = d.errors
		}
		var debounceCh <-chan time.Time
		if d.debounce != nil {
			debounceCh = d.debounce.C
		}

		select {
		case outCh <- pendingSource:
			pendingSource = ""

		case errOutCh <- pendingErr:
			_ = d.handleClose()
			return

		case err := <-d.backendErrs:
			if !paused && pendingErr == nil {
				pendingErr = err
			}

		case source := <-d.events:
			if paused || d.debounce != nil {
				continue
			}
			// Preserve an unread user-pack interruption over a "data" change so
			// pausing for export doesn't discard a pending BP/RP/extras edit.
			if pendingSource == "" || pendingSource == "data" {
				pendingSource = source
			}
			d.debounce = time.NewTimer(watchDebounceDuration)

		case <-debounceCh:
			d.debounce = nil

		case req := <-d.control:
			switch req.action {
			case "pause":
				req.done <- d.handlePause(&paused, &pendingSource)
			case "restart":
				err := d.handleRestart(&paused)
				if err != nil && pendingErr == nil {
					pendingErr = err
				}
				req.done <- err
			case "close":
				req.done <- d.handleClose()
				return
			default:
				req.done <- nil
			}

		case action, ok := <-d.stage:
			if !ok {
				_ = d.handleClose()
				return
			}
			switch action {
			case "pause":
				_ = d.handlePause(&paused, &pendingSource)
			case "restart":
				if err := d.handleRestart(&paused); err != nil && pendingErr == nil {
					pendingErr = err
				}
			}
		}
	}
}

func (d *DirWatcher) handlePause(paused *bool, pendingSource *string) error {
	var closeErr error
	if d.backend != nil {
		closeErr = d.backend.Close()
		d.backend = nil
	}
	*paused = true
	d.stopDebounce()
	d.drainBackendQueues()
	// Data exports can modify packs/data right as export begins; clear any
	// pending "data" interruption so exportData filters don't self-trigger.
	if *pendingSource == "data" {
		*pendingSource = ""
	}
	return closeErr
}

func (d *DirWatcher) handleRestart(paused *bool) error {
	if !*paused && d.backend != nil {
		return nil
	}
	d.drainBackendQueues()
	backend, err := startWatchBackend(d.roots, d.events, d.backendErrs)
	if err != nil {
		*paused = true
		return err
	}
	d.backend = backend
	*paused = false
	return nil
}

func (d *DirWatcher) handleClose() error {
	d.stopDebounce()
	var closeErr error
	if d.backend != nil {
		closeErr = d.backend.Close()
		d.backend = nil
	}
	d.drainBackendQueues()
	return closeErr
}

func (d *DirWatcher) stopDebounce() {
	if d.debounce != nil {
		d.debounce.Stop()
		d.debounce = nil
	}
}

func (d *DirWatcher) drainBackendQueues() {
	for {
		select {
		case <-d.events:
		case <-d.backendErrs:
		default:
			return
		}
	}
}

func isInDir(path, root string) bool {
	if root == "" {
		return false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = filepath.Clean(root)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = filepath.Clean(path)
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false
	}
	return rel != ".." &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator)) &&
		!filepath.IsAbs(rel)
}
