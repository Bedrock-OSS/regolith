package src

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	getter "github.com/hashicorp/go-getter"
)

// UrlToPath returns regolith cache path for given URL.
func UrlToPath(url string) string {
	return ".regolith/cache/filters/" + url
}

// FilterNameToUrl returns the URL of a standard filter based on its name.
func FilterNameToUrl(name string) string {
	return "github.com/Bedrock-OSS/regolith-filters//" + name
}

// IsRemotrFilterCached checks whether the filter of given URL is already saved
// in cache.
func IsRemoteFilterCached(url string) bool {
	_, err := os.Stat(UrlToPath(url))
	return err == nil
}

// InstallDependencies downloads all of the remote filters of every
// profile specified in config.json and recursively downloads the filters
// specified in filter.json of every downloaded filter.
func InstallDependencies() error {
	Logger.Infof("Installing dependencies...")
	Logger.Warnf("This may take a while...")

	err := os.MkdirAll(".regolith/cache/filters", 0777)
	if err != nil {
		return wrapError("Could not create .regolith/cache/filters", err)
	}
	// Special path for virtual environments for python
	err = os.MkdirAll(".regolith/cache/venvs", 0777)
	if err != nil {
		return wrapError("Could not create .regolith/cache/venvs", err)
	}

	project, err := LoadConfig()
	if err != nil {
		return wrapError("Failed to load project config", err)
	}
	for _, profile := range project.Profiles {
		err := InstallDependency(profile)
		if err != nil {
			return wrapError("Could not install dependency", err)
		}
	}

	Logger.Infof("Dependencies installed.")
	return nil
}

// InstallDependency recursively downloads the filters of a profile and the
// filters specifed in other filters.
func InstallDependency(profile Profile) error { // TODO - rename that and split into two functions?
	for _, filter := range profile.Filters {
		// Get the url of the dependency
		var url string
		if filter.Url != "" {
			url = filter.Url
		} else if filter.Filter != "" { // TODO - what if there is both URL and filter?
			url = FilterNameToUrl(filter.Filter)
		} else { // Leaf of profile tree (nothing to install)
			continue
		}
		Logger.Infof("Installing dependency %s...", url)

		// Download the filter into the cache folder
		path := UrlToPath(url)
		ok, err := DownloadGitHubUrl(url, "master", path)
		if err != nil {
			Logger.Debug(err)
		}
		if !ok {
			Logger.Debug("Failed to download filter " + filter.Filter + " without git")
			err := getter.Get(path, url)
			if err != nil {
				return err
			}
		}

		// Check required files
		file, err := ioutil.ReadFile(path + "/filter.json")
		if err != nil {
			return wrapError(fmt.Sprintf("Couldn't find %s/filter.json!", path), err)
		}

		// Load subprofile (remote filter)
		var remoteProfile Profile
		err = json.Unmarshal(file, &remoteProfile)
		if err != nil {
			return wrapError(fmt.Sprintf("Couldn't load %s/filter.json: ", path), err)
		}
		// Propagate venvSlot property
		for f := range remoteProfile.Filters {
			remoteProfile.Filters[f].VenvSlot = filter.VenvSlot
		}
		// Install dependencies of remote filters
		// recursion ends when there is no more nested remote dependencies
		err = InstallDependency(remoteProfile)
		if err != nil {
			return err
		}

		// Install filter dependencies
		for _, filter := range remoteProfile.Filters {
			if filter.RunWith != "" {
				if f, ok := FilterTypes[filter.RunWith]; ok {
					err := f.install(filter, path)
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
