package regolith

import (
	"os"
	"strings"
)

const STANDARD_LIBRARY_URL = "github.com/Bedrock-OSS/regolith-filters"

// UrlToPath returns regolith cache path for given URL.
// Version is ignored, implying that all versions of a filter are installed
// into the same location
func UrlToPath(url string) string {
	// Strip version from url
	url = strings.Split(url, "?")[0]
	return ".regolith/cache/filters/" + url
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
func InstallFilters(isForced bool, updateFilters bool) error {
	project := LoadConfig()

	CreateDirectoryIfNotExists(".regolith/cache/filters", true)
	CreateDirectoryIfNotExists(".regolith/cache/venvs", true)

	for profileName, profile := range project.Profiles {
		Logger.Infof("Installing profile %s...", profileName)
		err := profile.Install(isForced)
		if err != nil {
			return wrapError("Could not install profile!", err)
		}
	}

	Logger.Infof("Profile installation complete.")
	return nil
}
