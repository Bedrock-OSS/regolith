//go:build !windows
// +build !windows

package regolith

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

// comMojangDir returns the path to the com.mojang folder, for either
// Minecraft or Minecraft Preview.
func comMojangDir(preview bool) (string, error) {
	if runtime.GOOS != "linux" {
		return "", WrappedErrorf(
			"Unsupported operating system: '%s'", runtime.GOOS)
	}

	if _, err := exec.LookPath("wslpath"); err != nil {
		return "", WrapErrorf(err, "Could not find wslpath executable (are you running regolith within WSL2?).")
	}
	out, err := exec.Command("cmd.exe", "/c", "echo", "%localappdata%").Output()
	if err != nil {
		return "", WrapErrorf(err, "Could not find localappdata path.")
	}
	localAppData := strings.TrimSpace(string(out))

	packagesDir := "Microsoft.MinecraftUWP_8wekyb3d8bbwe"
	if preview {
		packagesDir = "Microsoft.MinecraftWindowsBeta_8wekyb3d8bbwe"
	}

	result := filepath.Join(
		localAppData, "Packages",
		packagesDir, "LocalState", "games",
		"com.mojang")

	out, err = exec.Command("wslpath", result).Output()
	if err != nil {
		return "", WrapErrorf(err, "Could not find resolve path to WSL path.")
	}
	wslPath := strings.TrimSpace(string(out))

	if _, err := os.Stat(wslPath); err != nil {
		if os.IsNotExist(err) {
			mcName := "Minecraft"
			if preview {
				mcName = "Minecraft Preview"
			}
			return "", WrapErrorf(
				err, "The %s \"com.mojang\" folder is not at \"%s\".\n"+
					"Does your system have multiple user accounts?", mcName, result)
		}
		return "", WrapErrorf(
			err, "Unable to access \"%s\".\n"+
				"Are your user permissions correct?", wslPath)
	}
	return wslPath, nil
}

// FindMojangDir returns path to the com.mojang folder.
func FindMojangDir() (string, error) {
	return comMojangDir(false)
}

// FindMojangDir returns path to the com.mojang folder for Minecraft Preview.
func FindPreviewDir() (string, error) {
	return comMojangDir(true)
}
