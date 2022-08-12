//go:build !windows
// +build !windows

package regolith

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	return nil, fmt.Errorf("Not implemented for this system.")
}

func (d *DirWatcher) WaitForChange() error {
	return fmt.Errorf("Not implemented for this system.")
}

func (d *DirWatcher) WaitForChangeGroup(
	groupTimeout uint32, interruptionChannel chan string,
	interruptionMessage string,
) error {
	return fmt.Errorf("Not implemented for this system.")
}

func (d *DirWatcher) Close() error {
	return fmt.Errorf("Not implemented for this system.")
}

/*
FindMojangDir locates the com.mojang folder on non-windows platforms.

If the platform is not WSL, this function will return an error.

If WslUser is an empty string, this function will return an error.
*/
func FindMojangDir(WslUser string) (string, error) {
	winUserDir := filepath.Join("/", "mnt", "c", "Users")
	if _, err := os.Stat(winUserDir); err != nil {
		return "", WrappedErrorf(
			"Unsupported operating system: '%s'", runtime.GOOS)
	}

	if WslUser == "" {
		return "", WrappedErrorf(
			"This platform appears to be WSL but the WslUser option was not set.")
	}

	result := filepath.Join(
		winUserDir, WslUser, "AppData", "Local", "Packages",
		"Microsoft.MinecraftUWP_8wekyb3d8bbwe", "LocalState", "games",
		"com.mojang")
	if _, err := os.Stat(result); err != nil {
		if os.IsNotExist(err) {
			return "", WrapErrorf(
				err, "The \"com.mojang\" folder is not at \"%s\".\n"+
					"Does your system have multiple user accounts?", result)
		}
		return "", WrapErrorf(
			err, "Unable to access \"%s\".\n"+
				"Are your user permissions correct?", result)
	}
	return result, nil
}
