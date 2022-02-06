package regolith

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/hashicorp/go-getter"
)

// Download
func (i *RemoteFilterDefinition) Download(isForced bool) error {
	if i.IsInstalled() {
		if !isForced {
			Logger.Warnf("Filter %q already installed, skipping. Run "+
				"with '-f' to force.", i.Id)
			return nil
		} else {
			// TODO should we print version information here?
			// like "version 1.4.2 uninstalled, version 1.4.3 installed"
			Logger.Warnf("Filter %q already installed, but force mode is enabled.\n"+
				"Filter will be installed, erasing prior contents.", i.Id)
			i.Uninstall()
		}
	}

	Logger.Infof("Downloading filter %s...", i.Id)

	// Download the filter using Git Getter
	// TODO:
	// Can we somehow detect whether this is a failure from git being not
	// installed, or a failure from
	// the repo/folder not existing?
	url := i.GetDownloadUrl()
	downloadPath := i.GetDownloadPath()
	err := getter.Get(downloadPath, url)
	if err != nil {
		return wrapErrorf(
			err,
			"Could not download filter from %s.\n"+
				"	Is git installed?\n"+
				"	Does that filter exist?", url)
	}

	// Remove 'test' folder, which we never want to use (saves space on disk)
	testFolder := path.Join(downloadPath, "test")
	if _, err := os.Stat(testFolder); err == nil {
		os.RemoveAll(testFolder)
	}

	Logger.Infof("Filter %s downloaded successfully.", i.Id)
	return nil
}

// GetDownloadUrl creates a download URL, based on the filter definition
func (i *RemoteFilterDefinition) GetDownloadUrl() string {
	repoVersion, err := GetRemoteFilterDownloadRef(
		i.Url, i.Id, i.Version, true)
	if err != nil {
		Logger.Fatal(err)
	}
	return fmt.Sprintf("%s//%s?ref=%s", i.Url, i.Id, repoVersion)
}

// GetDownloadPath returns the path location where the filter can be found.
func (i *RemoteFilterDefinition) GetDownloadPath() string {
	return filepath.Join(".regolith/cache/filters", i.Id)
}

func (i *RemoteFilterDefinition) Uninstall() {
	err := os.RemoveAll(i.GetDownloadPath())
	if err != nil {
		Logger.Error(
			wrapErrorf(err, "Could not remove installed filter %s.", i.Id))
	}
}

// IsInstalled eturns whether the filter is currently installed or not.
func (i *RemoteFilterDefinition) IsInstalled() bool {
	if _, err := os.Stat(i.GetDownloadPath()); err == nil {
		return true
	}
	return false
}
