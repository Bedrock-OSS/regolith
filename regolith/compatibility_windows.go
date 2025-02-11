//go:build windows
// +build windows

package regolith

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Bedrock-OSS/go-burrito/burrito"

	"golang.org/x/sys/windows"
)

// pythonExeNames is the list of strings with possible names of the Python
// executable. The order of the names determines the order in which they are
// tried.
var pythonExeNames = []string{"python", "python3"}

// venvScriptsPath is a folder name between "venv" and "python" that leads to
// the python executable.
const venvScriptsPath = "Scripts"

// exeSuffix is a suffix for executable files.
const exeSuffix = ".exe"

// Error used whe os.UserCacheDir fails
const osUserCacheDirError = "Failed to resolve %LocalAppData% path."

// copyFileSecurityInfo copies the DACL info from source path to DACL of
// the target path
func copyFileSecurityInfo(source string, target string) error {
	securityInfo, err := windows.GetNamedSecurityInfo(
		source,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return burrito.WrapError(err, "Unable to get security info from the source.")
	}
	dacl, _, err := securityInfo.DACL()
	if err != nil {
		return burrito.WrapErrorf(err, "Unable to get DACL of the source.")
	}
	err = windows.SetNamedSecurityInfo(
		target,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION, nil, nil, dacl, nil,
	)
	if err != nil {
		return burrito.WrapErrorf(err, "Unable to set DACL of the target.")
	}
	return nil
}

// FindStandardMojangDir returns path to the com.mojang folder in the standard
// Minecraft build.
func FindStandardMojangDir() (string, error) {
	comMojang := os.Getenv("COM_MOJANG")
	if comMojang != "" {
		return comMojang, nil
	}
	result := filepath.Join(
		os.Getenv("LOCALAPPDATA"), "Packages",
		"Microsoft.MinecraftUWP_8wekyb3d8bbwe", "LocalState", "games",
		"com.mojang")
	if _, err := os.Stat(result); err != nil {
		if os.IsNotExist(err) {
			return "", burrito.WrapErrorf(err, osStatErrorIsNotExist, result)
		}
		return "", burrito.WrapErrorf(err, osStatErrorAny, result)
	}
	return result, nil
}

// FindPreviewDir returns path to the com.mojang folder in the preview
// Minecraft build.
func FindPreviewDir() (string, error) {
	comMojang := os.Getenv("COM_MOJANG_PREVIEW")
	if comMojang != "" {
		return comMojang, nil
	}
	result := filepath.Join(
		os.Getenv("LOCALAPPDATA"), "Packages",
		"Microsoft.MinecraftWindowsBeta_8wekyb3d8bbwe", "LocalState", "games",
		"com.mojang")
	if _, err := os.Stat(result); err != nil {
		if os.IsNotExist(err) {
			return "", burrito.WrapErrorf(err, osStatErrorIsNotExist, result)
		}
		return "", burrito.WrapErrorf(
			err, osStatErrorAny, result)
	}
	return result, nil
}

// FindEducationDir returns path to the com.mojang folder in the education
// edition Minecraft build.
func FindEducationDir() (string, error) {
	comMojang := os.Getenv("COM_MOJANG_EDU")
	if comMojang != "" {
		return comMojang, nil
	}
	result := filepath.Join(
		os.Getenv("APPDATA"), "Minecraft Education Edition", "games",
		"com.mojang")
	if _, err := os.Stat(result); err != nil {
		if os.IsNotExist(err) {
			return "", burrito.WrapErrorf(err, osStatErrorIsNotExist, result)
		}
		return "", burrito.WrapErrorf(
			err, osStatErrorAny, result)
	}
	return result, nil
}

func CheckSuspiciousLocation() error {
	path, err := os.Getwd()
	if err != nil {
		return burrito.WrapErrorf(err, osGetwdError)
	}
	// Check if project directory is within mojang dir
	dir, err := FindStandardMojangDir()
	if err == nil {
		dir1 := filepath.Join(dir, "development_behavior_packs")
		if isPathWithinDirectory(path, dir1) {
			return burrito.WrappedErrorf(projectInMojangDirError, path, dir1)
		}
		dir1 = filepath.Join(dir, "development_resource_packs")
		if isPathWithinDirectory(path, dir1) {
			return burrito.WrappedErrorf(projectInMojangDirError, path, dir1)
		}
	}
	// Check if project directory is within mojang dir
	dir, err = FindPreviewDir()
	if err == nil {
		dir1 := filepath.Join(dir, "development_behavior_packs")
		if isPathWithinDirectory(path, dir1) {
			return burrito.WrappedErrorf(projectInPreviewDirError, path, dir1)
		}
		dir1 = filepath.Join(dir, "development_resource_packs")
		if isPathWithinDirectory(path, dir1) {
			return burrito.WrappedErrorf(projectInPreviewDirError, path, dir1)
		}
	}
	// Check if project directory is within OneDrive directories
	od := os.Getenv("OneDrive")
	if od != "" && isPathWithinDirectory(path, od) {
		Logger.Warnf("Project directory is within OneDrive directory. Consider moving the project outside of any cloud synced directories.\nPath: %s\nOneDrive: %s", path, od)
	} else {
		od = os.Getenv("OneDriveConsumer")
		if od != "" && isPathWithinDirectory(path, od) {
			Logger.Warnf("Project directory is within OneDrive Consumer directory. Consider moving the project outside of any cloud synced directories.\nPath: %s\nOneDrive: %s", path, od)
		} else {
			od = os.Getenv("OneDriveCommercial")
			if od != "" && isPathWithinDirectory(path, od) {
				Logger.Warnf("Project directory is within OneDrive Commercial directory. Consider moving the project outside of any cloud synced directories.\nPath: %s\nOneDrive: %s", path, od)
			}
		}
	}
	return nil
}

func isPathWithinDirectory(path string, dir string) bool {
	if path == "" || dir == "" {
		return false
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}
