//go:build linux
// +build linux

package regolith

import (
	"os"
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
//
// https://pkg.go.dev/golang.org/x/sys/unix
type DirWatcher struct {
	fd  int
	wd  int
	buf [5440]byte
}

// NewDirWatcher creats a new inotify watcher.
func NewDirWatcher(path string) (*DirWatcher, error) {
	fd, err := unix.InotifyInit1(0)
	if err != nil {
		return nil, err
	}
	wd, err := unix.InotifyAddWatch(fd, path, unix.IN_CREATE|unix.IN_DELETE|unix.IN_MOVED_TO|unix.FAN_MOVED_FROM|unix.IN_MOVE_SELF)
	if err != nil {
		return nil, err
	}
	var buf [(unix.SizeofInotifyEvent + unix.NAME_MAX + 1) * 20]byte
	return &DirWatcher{fd, wd, buf}, nil
}

// WaitForChange locks the goroutine until an inotify event is received.
func (d *DirWatcher) WaitForChange() error {
	_, err := unix.Read(d.fd, d.buf[:])
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
	defer timer.Stop()

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

// Close removes the inotify watcher.
func (d *DirWatcher) Close() error {
	unix.InotifyRmWatch(d.fd, uint32(d.wd))
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
