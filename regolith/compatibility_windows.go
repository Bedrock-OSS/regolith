//go:build windows
// +build windows

package regolith

import (
	"golang.org/x/sys/windows"
)

// venvScriptsPath is a folder name between "venv" and "python" that leads to
// the python executable.
const venvScriptsPath = "Scripts"

// exeSuffix is a suffix for executable files.
const exeSuffix = ".exe"

// copyFileSecurityInfo copies the DACL info from source path to DACL of
// the target path
func copyFileSecurityInfo(source string, target string) error {
	securityInfo, err := windows.GetNamedSecurityInfo(
		source,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return WrapErrorf(
			err, "Unable to get security info of %q.", source)
	}
	dacl, _, err := securityInfo.DACL()
	if err != nil {
		return WrapErrorf(
			err, "Unable to get DACL of %q.", source)
	}
	err = windows.SetNamedSecurityInfo(
		target,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION, nil, nil, dacl, nil,
	)
	if err != nil {
		return WrapErrorf(
			err, "Unable to set DACL of %q.", target)
	}
	return nil
}

// DirWatcher is a struct that provides easy to use methods for watching a
// directory for changes. It uses FindFirstChangeNotification instead of
// ReadDirectoryChanges so it doesn't provide any information about the
// changes, only the fact that something changed.
//
// Useful links:
// https://docs.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-findfirstchangenotificationa
//
// https://docs.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-findnextchangenotification
//
// https://pkg.go.dev/golang.org/x/sys@v0.0.0-20220412211240-33da011f77ad/windows
//
// https://docs.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-readdirectorychangesw
type DirWatcher struct {
	handle windows.Handle
}

// NewDirWatcher creates a new DirWatcher for the given path. It filters out
// some of the less interesting events like FILE_NOTIFY_CHANGE_LAST_ACCESS.
func NewDirWatcher(path string) (*DirWatcher, error) {
	var notifyFilter uint32 = (windows.FILE_NOTIFY_CHANGE_FILE_NAME |
		windows.FILE_NOTIFY_CHANGE_DIR_NAME |
		// windows.FILE_NOTIFY_CHANGE_ATTRIBUTES |
		// windows.FILE_NOTIFY_CHANGE_SIZE |
		windows.FILE_NOTIFY_CHANGE_LAST_WRITE |
		// windows.FILE_NOTIFY_CHANGE_LAST_ACCESS |
		// windows.FILE_NOTIFY_CHANGE_SECURITY |
		windows.FILE_NOTIFY_CHANGE_CREATION)
	handle, err := windows.FindFirstChangeNotification(
		path, true, notifyFilter)
	if err != nil {
		return nil, err
	}
	return &DirWatcher{handle: handle}, nil
}

// WaitForChange locks the goroutine until a single change is detected. Note
// that some changes are reported multiple times, for example saving a file
// will cause a change to the file and a change to the directory. If you want
// to report cases like that as one event, see WaitForChangeGroup.
func (d *DirWatcher) WaitForChange() error {
	_, err := windows.WaitForSingleObject(d.handle, windows.INFINITE)
	if err != nil {
		return err
	}
	err = windows.FindNextChangeNotification(d.handle)
	if err != nil {
		return err
	}
	return nil
}

// WaitForChangeGroup locks a goroutine until it recives a change notification.
// When that happens it sends the interruptionMessage to the
// interruptionChannel.
// Then it continues locking as long as other changes keep coming with
// intercals less than the given timeout, to group notifications that come
// in short intervals together.
func (d *DirWatcher) WaitForChangeGroup(
	groupTimeout uint32, interruptionChannel chan string,
	interruptionMessage string,
) error {
	err := d.WaitForChange()
	if err != nil {
		return err
	}
	// Instantly report the change
	interruptionChannel <- interruptionMessage
	// Consume all changes for groupDelay duration
	for {
		event, err := windows.WaitForSingleObject(d.handle, groupTimeout)
		if err != nil {
			return err
		}
		// Possible options: WAIT_OBJECT_0, WAIT_ABANDONED, WAIT_TIMEOUT,
		// WAIT_FAILED
		if event == uint32(windows.WAIT_TIMEOUT) ||
			event == uint32(windows.WAIT_ABANDONED) {
			break
		}
		err = windows.FindNextChangeNotification(d.handle)
		if err != nil {
			return err
		}
	}
	return nil
}

// Close closes DirWatcher handle.
func (d *DirWatcher) Close() error {
	return windows.CloseHandle(d.handle)
}
