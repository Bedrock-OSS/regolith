package regolith

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	getter "github.com/hashicorp/go-getter"
	"github.com/otiai10/copy"
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

// IsRemoteFilterCached checks whether the filter of given URL is already saved
// in cache.
func IsRemoteFilterCached(url string) bool {
	_, err := os.Stat(UrlToPath(url))
	return err == nil
}

/*
Recursively install dependencies for the entire config.
 - Force mode will overwrite existing dependencies.
 - Non-force mode will only install dependencies that are not already installed.
*/
func InstallDependencies(isForced bool) error {
	Logger.Infof("Installing dependencies...")

	err := os.MkdirAll(".regolith/cache/filters", 0666)
	if err != nil {
		return wrapError("Could not create .regolith/cache/filters", err)
	}

	// Special path for virtual environments for python
	err = os.MkdirAll(".regolith/cache/venvs", 0666)
	if err != nil {
		return wrapError("Could not create .regolith/cache/venvs", err)
	}

	project, err := LoadConfig()
	if err != nil {
		return wrapError("Failed to load project config", err)
	}

	for _, profile := range project.Profiles {
		err := InstallDependency(profile, isForced)
		if err != nil {
			return wrapError("Could not install dependency", err)
		}
	}

	Logger.Infof("Dependencies installed.")
	return nil
}

// InstallDependency recursively downloads the filters of a profile and the
// filters specified in other filters.
func InstallDependency(profile Profile, isForced bool) error {
	for _, filter := range profile.Filters {
		// Get the url of the dependency, which may be constructed
		var url string
		if filter.Url != "" {
			url = FilterNameToUrl(filter.Url, filter.Filter)
		} else if filter.Filter != "" {
			url = FilterNameToUrl(StandardLibraryUrl, filter.Filter)
		} else { // Leaf of profile tree (nothing to install)
			continue
		}
		Logger.Infof("Installing dependency %s...", url)

		// Download the filter into the cache folder
		downloadPath := UrlToPath(url)

		// If downloadPath already exists, continue
		if _, err := os.Stat(downloadPath); err == nil {
			if !isForced {
				Logger.Infof("Dependency %s already installed, skipping.", url)
				continue
			} else {
				Logger.Infof("Dependency %s already installed, but forcing installation.", url)
				err := os.RemoveAll(downloadPath)
				if err != nil {
					return wrapError("Could not remove installed filter", err)
				}

			}
		}

		// Download the filter fresh
		ok, err := DownloadGitHubUrl(url, downloadPath)
		if err != nil {
			Logger.Debug(err)
		}
		if !ok {
			Logger.Debug("Failed to download filter " + filter.Filter + " without git")
			err := getter.Get(downloadPath, url)
			if err != nil {
				return err
			}
		}

		// Move filters 'data' folder contents into 'data'
		filterName := strings.Split(path.Clean(url), "/")[3]
		filterDataPath := path.Join(profile.DataPath, filterName)

		// If the filterDataPath already exists, print a warning and continue

		if _, err := os.Stat(filterDataPath); err == nil {
			if isForced {
				Logger.Infof("Installation forced. Overwriting existing filter data %s", filterDataPath)
				err := os.RemoveAll(filterDataPath)
				if err != nil {
					return wrapError("Could not remove existing filter data", err)
				}
			} else {
				Logger.Warnf("Filter %s already has data in the 'data' folder. You may re-run with --force to download anyways.", filterName)
			}
			continue
		}

		err = os.MkdirAll(filterDataPath, 0666)
		if err != nil {
			Logger.Error("Could not create filter data folder", err)
		}

		if profile.DataPath != "" {
			err = copy.Copy(path.Join(downloadPath, "data"), filterDataPath, copy.Options{PreserveTimes: false, Sync: false})
			if err != nil {
				Logger.Error("Could not initialize filter data", err)
			}
		}

		// Check required files
		file, err := ioutil.ReadFile(downloadPath + "/filter.json")
		if err != nil {
			return wrapError(fmt.Sprintf("Couldn't find %s/filter.json!", downloadPath), err)
		}

		// Load subprofile (remote filter)
		var remoteProfile Profile
		err = json.Unmarshal(file, &remoteProfile)
		if err != nil {
			return wrapError(fmt.Sprintf("Couldn't load %s/filter.json: ", downloadPath), err)
		}
		// Propagate venvSlot property
		for f := range remoteProfile.Filters {
			remoteProfile.Filters[f].VenvSlot = filter.VenvSlot
		}
		// Install dependencies of remote filters
		// recursion ends when there is no more nested remote dependencies
		err = InstallDependency(remoteProfile, isForced)
		if err != nil {
			return err
		}

		// Install filter dependencies
		for _, filter := range remoteProfile.Filters {
			if filter.RunWith != "" {
				if f, ok := FilterTypes[filter.RunWith]; ok {
					err := f.install(filter, downloadPath)
					if err != nil {
						return wrapError(fmt.Sprintf("Couldn't install filter %s", filter.Name), err)
					}
				} else {
					Logger.Warnf("Filter type '%s' not supported", filter.RunWith)
				}
			}
		}
	}
	return nil
}
