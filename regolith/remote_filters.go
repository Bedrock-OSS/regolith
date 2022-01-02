package regolith

import (
	"fmt"
	"os"
	"strings"
)

const StandardLibraryUrl = "github.com/Bedrock-OSS/regolith-filters"

// UrlToPath returns regolith cache path for given URL.
func UrlToPath(url string) string {
	return ".regolith/cache/filters/" + url
}

// FilterNameToUrl returns the URL of a standard filter based on its name.
func FilterNameToUrl(libraryUrl string, name string) string {
	return fmt.Sprintf("%s//%s", libraryUrl, name)
}

func ValidateUrl(url string) error {
	if !strings.HasPrefix(url, "http") {
		return fmt.Errorf("Invalid URL: %s", url)
	}
	return nil
}

// IsRemoteFilterCached checks whether the filter of given URL is already saved
// in cache.
func IsRemoteFilterCached(url string) bool {
	_, err := os.Stat(UrlToPath(url))
	return err == nil
}

// Recursively install dependencies for the entire config.
//  - Force mode will overwrite existing dependencies.
//  - Non-force mode will only install dependencies that are not already installed.
func InstallDependencies(isForced bool) error {
	Logger.Infof("Installing dependencies...")

	project := LoadConfig()

	CreateDirectoryIfNotExists(".regolith/cache/filters", true)
	CreateDirectoryIfNotExists(".regolith/cache/venvs", true)

	wd, err := os.Getwd()
	if err != nil {
		return wrapError("Could not get working directory", err)
	}
	for _, profile := range project.Profiles {
		err := profile.Install(isForced, wd)
		if err != nil {
			return wrapError("Could not install dependency", err)
		}
	}

	Logger.Infof("Dependencies installed.")
	return nil
}
