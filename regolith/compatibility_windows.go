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
