//go:build !windows
// +build !windows

package regolith

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"github.com/fsnotify/fsnotify"
)

type resolvedWatchRoot struct {
	absPath  string
	realPath string
	source   string
	isDir    bool
}

type unixWatchBackend struct {
	watcher   *fsnotify.Watcher
	stopCh    chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
}

func startWatchBackend(
	roots []watchRoot,
	events chan<- string,
	errCh chan<- error,
) (watchBackend, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, burrito.WrapError(err, "Could not initialize directory watching")
	}

	resolvedRoots := make([]resolvedWatchRoot, 0, len(roots))
	for _, root := range roots {
		absPath, err := filepath.Abs(root.path)
		if err != nil {
			_ = watcher.Close()
			return nil, burrito.WrapErrorf(err, "Could not start watching `%s`", root.path)
		}
		info, err := os.Stat(absPath)
		if err != nil {
			_ = watcher.Close()
			return nil, burrito.WrapErrorf(err, "Could not start watching `%s`", root.path)
		}
		realPath := absPath
		if evaluated, err := filepath.EvalSymlinks(absPath); err == nil {
			realPath = evaluated
		}
		if err := addPathRecursive(watcher, absPath, true); err != nil {
			_ = watcher.Close()
			return nil, err
		}
		resolvedRoots = append(resolvedRoots, resolvedWatchRoot{
			absPath:  absPath,
			realPath: realPath,
			source:   root.source,
			isDir:    info.IsDir(),
		})
	}

	backend := &unixWatchBackend{
		watcher: watcher,
		stopCh:  make(chan struct{}),
	}
	backend.wg.Add(1)
	go backend.run(resolvedRoots, events, errCh)
	return backend, nil
}

func (b *unixWatchBackend) run(
	roots []resolvedWatchRoot,
	events chan<- string,
	errCh chan<- error,
) {
	defer b.wg.Done()

	for {
		select {
		case <-b.stopCh:
			return
		case err, ok := <-b.watcher.Errors:
			if !ok {
				return
			}
			select {
			case <-b.stopCh:
				return
			default:
			}
			select {
			case errCh <- err:
			default:
			}
			return
		case event, ok := <-b.watcher.Events:
			if !ok {
				return
			}
			// Ignore pure chmod events, matching previous watcher behavior.
			if event.Op == fsnotify.Chmod {
				continue
			}
			// Register newly created subdirectories before forwarding the event so
			// nested files created shortly after are not missed.
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if filepath.Base(event.Name) != ".git" {
						if err := addPathRecursive(b.watcher, event.Name, false); err != nil {
							select {
							case errCh <- err:
							default:
							}
							return
						}
					}
				}
			}
			source := matchWatchSource(event.Name, roots)
			if source != "" {
				select {
				case events <- source:
				default:
				}
			}
		}
	}
}

func (b *unixWatchBackend) Close() error {
	var closeErr error
	b.closeOnce.Do(func() {
		close(b.stopCh)
		closeErr = b.watcher.Close()
		b.wg.Wait()
	})
	return closeErr
}

func addPathRecursive(watcher *fsnotify.Watcher, rootPath string, requireRoot bool) error {
	info, err := os.Stat(rootPath)
	if err != nil {
		if !requireRoot && os.IsNotExist(err) {
			return nil
		}
		return wrapUnixWatchError(err, rootPath)
	}
	if !info.IsDir() {
		if err := watcher.Add(rootPath); err != nil {
			if !requireRoot && os.IsNotExist(err) {
				return nil
			}
			return wrapUnixWatchError(err, rootPath)
		}
		return nil
	}

	walkRoot := rootPath
	if linfo, lerr := os.Lstat(rootPath); lerr == nil && linfo.Mode()&os.ModeSymlink != 0 {
		if resolved, rerr := filepath.EvalSymlinks(rootPath); rerr == nil {
			walkRoot = resolved
		}
	}

	err = filepath.WalkDir(walkRoot, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) && (p != walkRoot || !requireRoot) {
				return nil
			}
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		if p != walkRoot && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if err := watcher.Add(p); err != nil {
			if os.IsNotExist(err) && (p != walkRoot || !requireRoot) {
				return nil
			}
			return err
		}
		return nil
	})
	if err != nil {
		if !requireRoot && os.IsNotExist(err) {
			return nil
		}
		return wrapUnixWatchError(err, rootPath)
	}
	return nil
}

func wrapUnixWatchError(err error, path string) error {
	if errors.Is(err, syscall.EMFILE) {
		return burrito.WrapErrorf(
			err,
			"Could not start watching `%s`: too many open files (consider increasing `ulimit -n`)",
			path,
		)
	}
	if errors.Is(err, syscall.ENOSPC) {
		return burrito.WrapErrorf(
			err,
			"Could not start watching `%s`: reached system inotify watch limit (consider increasing fs.inotify.max_user_watches)",
			path,
		)
	}
	return burrito.WrapErrorf(err, "Could not start watching `%s`", path)
}

func matchWatchSource(eventPath string, roots []resolvedWatchRoot) string {
	absEvent, err := filepath.Abs(eventPath)
	if err != nil {
		absEvent = filepath.Clean(eventPath)
	}
	realEvent := absEvent
	if evaluated, err := filepath.EvalSymlinks(absEvent); err == nil {
		realEvent = evaluated
	}

	for _, r := range roots {
		if r.isDir {
			if isInDir(absEvent, r.absPath) ||
				(r.realPath != r.absPath && (isInDir(absEvent, r.realPath) || isInDir(realEvent, r.realPath))) {
				return r.source
			}
		} else {
			if absEvent == r.absPath || realEvent == r.realPath {
				return r.source
			}
		}
	}
	return ""
}
