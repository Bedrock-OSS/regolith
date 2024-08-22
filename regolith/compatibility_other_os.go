//go:build !windows
// +build !windows

package regolith

import (
	"os"

	"github.com/Bedrock-OSS/go-burrito/burrito"
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

type DirWatcher struct{}

func NewDirWatcher(path string) (*DirWatcher, error) {
	return nil, burrito.WrappedError(notImplementedOnThisSystemError)
}

func (d *DirWatcher) WaitForChange() error {
	return burrito.WrappedError(notImplementedOnThisSystemError)
}

func (d *DirWatcher) WaitForChangeGroup(
	groupTimeout uint32, interruptionChannel chan string,
	interruptionMessage string,
) error {
	return burrito.WrappedError(notImplementedOnThisSystemError)
}

func (d *DirWatcher) Close() error {
	return burrito.WrappedError(notImplementedOnThisSystemError)
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
