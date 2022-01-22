package regolith

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/go-getter"
)

type FilterDefinition struct {
	Filter  string `json:"-"`
	Url     string `json:"url,omitempty"`
	Version string `json:"version,omitempty"`
}

func FilterDefinitionFromObject(name string, obj map[string]interface{}) FilterDefinition {
	result := FilterDefinition{}
	result.Filter = name
	url, ok := obj["url"].(string)
	if !ok {
		Logger.Fatal("could not find url in filter definition %s", name)
	}
	result.Url = url
	version, ok := obj["version"].(string)
	if !ok {
		Logger.Fatal("could not find version in filter definition %s", name)
	}
	result.Version = version
	return result
}

// Download
func (i *FilterDefinition) Download(isForced bool) error {
	if i.IsInstalled() {
		if !isForced {
			Logger.Warnf("Filter %q already installed, skipping. Run "+
				"with '-f' to force.", i.Filter)
			return nil
		} else {
			// TODO should we print version information here?
			// like "version 1.4.2 uninstalled, version 1.4.3 installed"
			Logger.Warnf("Filter %q already installed, but force mode is enabled.\n"+
				"Filter will be installed, erasing prior contents.", i.Filter)
			i.Uninstall()
		}
	}

	Logger.Infof("Downloading filter %s...", i.Filter)

	// Download the filter using Git Getter
	// TODO:
	// Can we somehow detect whether this is a failure from git being not
	// installed, or a failure from
	// the repo/folder not existing?
	url := i.GetDownloadUrl()
	downloadPath := i.GetDownloadPath()
	err := getter.Get(downloadPath, url)
	if err != nil {
		return wrapError(fmt.Sprintf(
			"Could not download filter from %s.\n"+
				"	Is git installed?\n"+
				"	Does that filter exist?", url), err)
	}

	// Remove 'test' folder, which we never want to use (saves space on disk)
	testFolder := path.Join(downloadPath, "test")
	if _, err := os.Stat(testFolder); err == nil {
		os.RemoveAll(testFolder)
	}

	Logger.Infof("Filter %s downloaded successfully.", i.Filter)
	return nil
}

// GetDownloadUrl creates a download URL, based on the filter definition
func (i *FilterDefinition) GetDownloadUrl() string {
	repoVersion, err := GetRemoteFilterDownloadRef(
		i.Url, i.Filter, i.Version, true)
	if err != nil {
		Logger.Fatal(err)
	}
	return fmt.Sprintf("%s//%s?ref=%s", i.Url, i.Filter, repoVersion)
}

// GetDownloadPath returns the path location where the filter can be found.
func (i *FilterDefinition) GetDownloadPath() string {
	return filepath.Join(".regolith/cache/filters", i.Filter)
}

func (i *FilterDefinition) Uninstall() {
	err := os.RemoveAll(i.GetDownloadPath())
	if err != nil {
		Logger.Error(wrapError(fmt.Sprintf("Could not remove installed filter %s.", i.Filter), err))
	}
}

// IsInstalled eturns whether the filter is currently installed or not.
func (i *FilterDefinition) IsInstalled() bool {
	if _, err := os.Stat(i.GetDownloadPath()); err == nil {
		return true
	}
	return false
}
