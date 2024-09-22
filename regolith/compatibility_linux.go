//go:build linux
// +build linux

package regolith

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"golang.org/x/sys/unix"
)

// pythonExeNames is the list of strings with possible names of the Python
// executable. The order of the names determines the order in which they are
// tried.
var pythonExeNames = []string{"python3", "python"}

// venvScriptsPath is a folder name between "venv" and "python" that leads to
// the python executable.
const venvScriptsPath = "bin"

// exeSuffix is a suffix for executable files.
const exeSuffix = ""

// Error used whe os.UserCacheDir fails
const osUserCacheDirError = "Failed to get user cache directory."

// copyFileSecurityInfo placeholder for a function which is necessary only
// on Windows.
func copyFileSecurityInfo(source string, target string) error {
	return nil
}

// DirWatcher is a struct that provides an easy to use methods for watching a
// directory for changes. It uses the inotify API.
//
// Useful links:
// https://www.man7.org/linux/man-pages/man7/inotify.7.html
// https://pkg.go.dev/golang.org/x/sys/unix
type DirWatcher struct {
	fileDescriptor      int
	watchDescriptorList []int
	eventBuffer         [5440]byte
}

// NewDirWatcher creates a new inotify instance and adds watchers for each
// subdirectory recursively.
func NewDirWatcher(path string) (*DirWatcher, error) {
	fileDescriptor, err := unix.InotifyInit1(0)
	if err != nil {
		return nil, burrito.WrapError(err, "Could not create an inotify instance")
	}

	mask := uint32(unix.IN_CREATE | unix.IN_MODIFY | unix.IN_DELETE | unix.IN_MOVED_TO | unix.IN_MOVED_FROM | unix.IN_MOVE_SELF)
	var eventBuffer [(unix.SizeofInotifyEvent + unix.NAME_MAX + 1) * 20]byte
	var watchDescriptorList []int

	err = filepath.WalkDir(path, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return burrito.WrapErrorf(err, "Could not descent into `%s` to initiate file watching", path)
		}
		if !entry.IsDir() {
			return nil
		}
		watchDescriptor, err := unix.InotifyAddWatch(fileDescriptor, path, mask)
		if err != nil {
			return burrito.WrapErrorf(err, "Could not add a new inotify watcher at `%s`", path)
		}
		watchDescriptorList = append(watchDescriptorList, watchDescriptor)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &DirWatcher{fileDescriptor, watchDescriptorList, eventBuffer}, nil
}

// WaitForChange locks the goroutine until an inotify event is received.
func (d *DirWatcher) WaitForChange() error {
	_, err := unix.Read(d.fileDescriptor, d.eventBuffer[:])
	if err != nil {
		return burrito.WrapError(err, "Could not read inotify event")
	}
	return nil
}

// WaitForChangeGroup locks a goroutine until it receives an inotify event.
// When that happens it sends the interruptionMessage to the
// interruptionChannel.
// Then it continues locking as long as other events keep coming with
// intervals less than the given timeout.
func (d *DirWatcher) WaitForChangeGroup(
	groupTimeout uint32,
	interruptionChannel chan string,
	interruptionMessage string,
) error {
	if err := d.WaitForChange(); err != nil {
		return err
	}

	interruptionChannel <- interruptionMessage

	timer := time.NewTimer(time.Duration(groupTimeout))

	for {
		select {
		case <-timer.C:
			return nil
		default:
			if err := d.WaitForChange(); err != nil {
				return err
			}
		}
	}
}

// Close removes all inotify watchers.
func (d *DirWatcher) Close() error {
	for watchDescriptor := range d.watchDescriptorList {
		unix.InotifyRmWatch(d.fileDescriptor, uint32(watchDescriptor))
	}
	return nil
}

func FindStandardMojangDir() (string, error) {
	comMojang := os.Getenv("COM_MOJANG")
	if comMojang == "" {
		return "", burrito.WrappedError(comMojangEnvUnsetError)
	}
	return comMojang, nil
}

func FindPreviewDir() (string, error) {
	comMojangPreview := os.Getenv("COM_MOJANG_PREVIEW")
	if comMojangPreview == "" {
		return "", burrito.WrappedError(comMojangPreviewEnvUnsetError)
	}
	return comMojangPreview, nil
}

func FindEducationDir() (string, error) {
	comMojangEdu := os.Getenv("COM_MOJANG_EDU")
	if comMojangEdu == "" {
		return "", burrito.WrappedError(comMojangEduEnvUnsetError)
	}
	return comMojangEdu, nil
}

func CheckSuspiciousLocation() error {
	return nil
}
