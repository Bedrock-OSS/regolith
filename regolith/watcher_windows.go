//go:build windows
// +build windows

package regolith

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"golang.org/x/sys/windows"
)

const windowsNotifyFilter = windows.FILE_NOTIFY_CHANGE_FILE_NAME |
	windows.FILE_NOTIFY_CHANGE_DIR_NAME |
	windows.FILE_NOTIFY_CHANGE_SIZE |
	windows.FILE_NOTIFY_CHANGE_LAST_WRITE |
	windows.FILE_NOTIFY_CHANGE_CREATION

type fileSnapshot struct {
	exists  bool
	size    int64
	modTime time.Time
}

type windowsRootWatch struct {
	root       watchRoot
	handle     windows.Handle
	singleFile string
	fileState  fileSnapshot
}

type windowsWatchBackend struct {
	stopEvent windows.Handle
	wg        sync.WaitGroup
	closeOnce sync.Once
}

func snapshotFile(path string) fileSnapshot {
	info, err := os.Stat(path)
	if err != nil {
		return fileSnapshot{exists: false}
	}
	return fileSnapshot{
		exists:  true,
		size:    info.Size(),
		modTime: info.ModTime(),
	}
}

func (s fileSnapshot) equal(other fileSnapshot) bool {
	return s.exists == other.exists &&
		s.size == other.size &&
		s.modTime.Equal(other.modTime)
}

func startWatchBackend(
	roots []watchRoot,
	events chan<- string,
	errors chan<- error,
) (watchBackend, error) {
	// Manual-reset event kept nonsignaled until Close() so all root goroutines
	// wake up together and release their notification handles cleanly.
	stopEvent, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return nil, burrito.WrapError(err, "Could not initialize directory watching")
	}

	watches := make([]windowsRootWatch, 0, len(roots))
	cleanupOpened := func() {
		for _, rw := range watches {
			_ = windows.FindCloseChangeNotification(rw.handle)
		}
		_ = windows.CloseHandle(stopEvent)
	}

	for _, root := range roots {
		absPath, err := filepath.Abs(root.path)
		if err != nil {
			cleanupOpened()
			return nil, burrito.WrapErrorf(err, "Could not start watching `%s`", root.path)
		}
		info, err := os.Stat(absPath)
		if err != nil {
			cleanupOpened()
			return nil, burrito.WrapErrorf(err, "Could not start watching `%s`", root.path)
		}

		if info.IsDir() {
			// Hold a single subtree handle on the root directory rather than per-subdir
			// handles, avoiding NTFS delete-pending locks when users delete nested folders.
			handle, err := windows.FindFirstChangeNotification(absPath, true, windowsNotifyFilter)
			if err != nil {
				cleanupOpened()
				return nil, burrito.WrapErrorf(err, "Could not start watching `%s`", root.path)
			}
			watches = append(watches, windowsRootWatch{
				root:   root,
				handle: handle,
			})
		} else {
			// FindFirstChangeNotification only accepts directories, so watch the
			// parent directory non-recursively and filter by file metadata changes.
			handle, err := windows.FindFirstChangeNotification(filepath.Dir(absPath), false, windowsNotifyFilter)
			if err != nil {
				cleanupOpened()
				return nil, burrito.WrapErrorf(err, "Could not start watching `%s`", root.path)
			}
			watches = append(watches, windowsRootWatch{
				root:       root,
				handle:     handle,
				singleFile: absPath,
				fileState:  snapshotFile(absPath),
			})
		}
	}

	backend := &windowsWatchBackend{
		stopEvent: stopEvent,
	}
	for _, rw := range watches {
		backend.wg.Add(1)
		go backend.watchRoot(rw, events, errors)
	}
	return backend, nil
}

func (b *windowsWatchBackend) watchRoot(
	rw windowsRootWatch,
	events chan<- string,
	errors chan<- error,
) {
	defer b.wg.Done()
	defer windows.FindCloseChangeNotification(rw.handle)

	// Place stopEvent first so WaitForMultipleObjects prioritizes shutdown
	// when both stopEvent and the change handle are signaled simultaneously.
	handles := []windows.Handle{b.stopEvent, rw.handle}
	for {
		waitResult, err := windows.WaitForMultipleObjects(handles, false, windows.INFINITE)
		if err != nil {
			if isEventSignaled(b.stopEvent) {
				return
			}
			sendWatchError(errors, burrito.WrapErrorf(err, "Error while watching `%s`", rw.root.path))
			return
		}

		switch waitResult {
		case windows.WAIT_OBJECT_0:
			return
		case windows.WAIT_OBJECT_0 + 1:
			if err := windows.FindNextChangeNotification(rw.handle); err != nil {
				if isEventSignaled(b.stopEvent) {
					return
				}
				sendWatchError(errors, burrito.WrapErrorf(err, "Error while watching `%s`", rw.root.path))
				return
			}
			if isEventSignaled(b.stopEvent) {
				return
			}
			if rw.singleFile != "" {
				current := snapshotFile(rw.singleFile)
				if current.equal(rw.fileState) {
					continue
				}
				rw.fileState = current
			}
			select {
			case events <- rw.root.source:
			default:
			}
		default:
			if isEventSignaled(b.stopEvent) {
				return
			}
			sendWatchError(errors, burrito.WrappedErrorf("Unexpected wait status %d while watching `%s`", waitResult, rw.root.path))
			return
		}
	}
}

func (b *windowsWatchBackend) Close() error {
	var closeErr error
	b.closeOnce.Do(func() {
		if err := windows.SetEvent(b.stopEvent); err != nil {
			closeErr = err
		}
		b.wg.Wait()
		if err := windows.CloseHandle(b.stopEvent); err != nil && closeErr == nil {
			closeErr = err
		}
	})
	return closeErr
}

func isEventSignaled(h windows.Handle) bool {
	res, err := windows.WaitForSingleObject(h, 0)
	return err == nil && res == windows.WAIT_OBJECT_0
}

func sendWatchError(errors chan<- error, err error) {
	select {
	case errors <- err:
	default:
	}
}
