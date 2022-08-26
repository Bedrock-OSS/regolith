//go:build !windows
// +build !windows

package regolith

import (
	"fmt"
)

// venvScriptsPath is a folder name between "venv" and "python" that leads to
// the python executable.
const venvScriptsPath = "bin"

// exeSuffix is a suffix for executable files.
const exeSuffix = ""

// copyFileSecurityInfo placeholder for a function which is necessary only
// on Windows.
func copyFileSecurityInfo(source string, target string) error {
	return nil
}

type DirWatcher struct{}

func NewDirWatcher(path string) (*DirWatcher, error) {
	return nil, fmt.Errorf(notImplementedOnThisSystemError)
}

func (d *DirWatcher) WaitForChange() error {
	return fmt.Errorf(notImplementedOnThisSystemError)
}

func (d *DirWatcher) WaitForChangeGroup(
	groupTimeout uint32, interruptionChannel chan string,
	interruptionMessage string,
) error {
	return fmt.Errorf(notImplementedOnThisSystemError)
}

func (d *DirWatcher) Close() error {
	return fmt.Errorf(notImplementedOnThisSystemError)
}

func FindMojangDir() (string, error) {
	return "", WrappedErrorf(
		"Unsupported operating system: '%s'", runtime.GOOS)
}
