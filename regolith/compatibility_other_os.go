//go:build !windows
// +build !windows

package regolith

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

type DirWatcher struct{}

func NewDirWatcher(path string) (*DirWatcher, error) {
	return nil, WrappedError(notImplementedOnThisSystemError)
}

func (d *DirWatcher) WaitForChange() error {
	return WrappedError(notImplementedOnThisSystemError)
}

func (d *DirWatcher) WaitForChangeGroup(
	groupTimeout uint32, interruptionChannel chan string,
	interruptionMessage string,
) error {
	return WrappedError(notImplementedOnThisSystemError)
}

func (d *DirWatcher) Close() error {
	return WrappedError(notImplementedOnThisSystemError)
}

func FindMojangDir() (string, error) {
	return "", WrappedError(notImplementedOnThisSystemError)
}

func FindPreviewDir() (string, error) {
	return "", WrappedError(notImplementedOnThisSystemError)
}
