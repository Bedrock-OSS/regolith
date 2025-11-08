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

func findSomeMojangDir(
	comMojangWordsVar,
	comMojangPacksVar,
	comMojangVar,
	comMojangEnvUnsetError string,
	pathType ComMojangPathType,
) (string, error) {
	// Try specific path environment variables first
	switch pathType {
	case WorldPath:
		comMojang := os.Getenv(comMojangWordsVar)
		if comMojang != "" {
			return comMojang, nil
		}
	case PacksPath:
		comMojang := os.Getenv(comMojangPacksVar)
		if comMojang != "" {
			return comMojang, nil
		}
	}
	// Try general environment variable
	comMojang := os.Getenv(comMojangVar)
	if comMojang == "" {
		return "", burrito.WrappedError(comMojangEnvUnsetError)
	}
	return comMojang, nil
}

func FindStandardMojangDir(pathType ComMojangPathType) (string, error) {
	return findSomeMojangDir(
		"COM_MOJANG_WORLDS",
		"COM_MOJANG_PACKS",
		"COM_MOJANG",
		comMojangEnvUnsetError,
		pathType,
	)
}

func FindPreviewDir(pathType ComMojangPathType) (string, error) {
	return findSomeMojangDir(
		"COM_MOJANG_WORLDS_PREVIEW",
		"COM_MOJANG_PACKS_PREVIEW",
		"COM_MOJANG_PREVIEW",
		comMojangPreviewEnvUnsetError,
		pathType,
	)
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
