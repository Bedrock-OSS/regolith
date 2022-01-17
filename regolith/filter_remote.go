package regolith

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-getter"
	"github.com/otiai10/copy"
)

type RemoteFilter struct {
	Filter

	Id string `json:"filter,omitempty"`

	// RemoteFilters can propagate some of the properties unique to other types
	// of filers (like Python's venvSlot).
	VenvSlot int `json:"venvSlot,omitempty"`
}

func RemoteFilterFromObject(obj map[string]interface{}) *RemoteFilter {
	filter := &RemoteFilter{Filter: *FilterFromObject(obj)}
	id, ok := obj["filter"].(string) // filter property is optional
	if !ok {
		Logger.Fatalf(
			"remote filter %q is missing \"filter\" field",
			filter.GetFriendlyName())
	}
	filter.Id = id

	filter.VenvSlot, _ = obj["venvSlot"].(int) // default venvSlot is 0
	return filter
}

func (f *RemoteFilter) Run(absoluteLocation string) error {
	// Disabled filters are skipped
	if f.Disabled {
		Logger.Infof("Filter '%s' is disabled, skipping.", f.GetFriendlyName())
		return nil
	}
	// All other filters require safe mode to be turned off
	if f.GetUrl() != StandardLibraryUrl && !IsUnlocked() {
		return errors.New(
			"safe mode is on, which protects you from potentially unsafe " +
				"code.\nYou may turn it off using 'regolith unlock'",
		)
	}
	Logger.Infof("Running filter %s", f.GetFriendlyName())
	start := time.Now()
	defer Logger.Debugf("Executed in %s", time.Since(start))

	Logger.Debugf("RunRemoteFilter '%s'", f.GetUrl())
	if !f.IsCached() {
		return errors.New("filter is not downloaded! Please run 'regolith install'")
	}

	path := f.GetDownloadPath()
	absolutePath, _ := filepath.Abs(path)
	filterCollection, err := FilterCollectionFromFilterJson(
		filepath.Join(path, "filter.json"))
	if err != nil {
		return err
	}
	for _, filter := range filterCollection.Filters {
		// Overwrite the venvSlot with the parent value
		filter.CopyArguments(f)
		err := filter.Run(absolutePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *RemoteFilter) InstallDependencies(parent *RemoteFilter) error {
	return nil // Remote filters don't install any dependencies
}

func (f *RemoteFilter) Check() error {
	return nil
}

func (f *RemoteFilter) CopyArguments(parent *RemoteFilter) {
	f.Arguments = parent.Arguments
	f.Settings = parent.Settings
	f.VenvSlot = parent.VenvSlot
}

func (f *RemoteFilter) GetFriendlyName() string {
	if f.Name != "" {
		return f.Name
	} else if f.Id != "" {
		return f.Id
	}
	_, end := path.Split(f.GetUrl()) // Return the last part of the URL
	return end
}

// Download ownloads the filter into its own directory and returns the
// download path of the directory.
func (f *RemoteFilter) Download(isForced bool) (string, error) {
	url := f.GetDownloadUrl()
	downloadPath := f.GetDownloadPath()

	if f.IsInstalled() {
		if !isForced {
			Logger.Warnf("Filter %s already installed, skipping. Run "+
				"with '-f' to force.", f.GetFriendlyName())
			return "", nil
		} else {
			// TODO should we print version information here?
			// like "version 1.4.2 uninstalled, version 1.4.3 installed"
			Logger.Warnf("Filter %s already installed, but force mode is enabled.\n"+
				"Filter will be installed, erasing prior contents.", f.GetFriendlyName())
			f.Uninstall()
		}
	}

	Logger.Infof("Downloading filter %s...", f.GetFriendlyName())

	// Download the filter using Git Getter
	// TODO:
	// Can we somehow detect whether this is a failure from git being not installed, or a failure from
	// the repo/folder not existing?
	err := getter.Get(downloadPath, url)
	if err != nil {
		return "", wrapError(fmt.Sprintf("Could not download filter from %s. \n	Is git installed? \n	Does that filter exist?", url), err)
	}

	// Remove 'test' folder, which we never want to use (saves space on disk)
	testFolder := path.Join(downloadPath, "test")
	if _, err := os.Stat(testFolder); err == nil {
		os.RemoveAll(testFolder)
	}

	Logger.Infof("Filter %s downloaded successfully.", f.GetFriendlyName())
	return downloadPath, nil
}

// CopyFilterData copies the filter's data to the data folder.
func (f *RemoteFilter) CopyFilterData(profile *Profile) {
	// Move filters 'data' folder contents into 'data'
	// If the localDataPath already exists, we must not overwrite
	// Additionally, if the remote data path doesn't exist, we don't need
	// to do anything
	filterName := f.GetIdName()
	remoteDataPath := path.Join(f.GetDownloadPath(), "data")
	localDataPath := path.Join(profile.DataPath, filterName)
	if _, err := os.Stat(localDataPath); err == nil {
		Logger.Warnf("Filter %s already has data in the 'data' folder. \n"+
			"You may manually delete this data and reinstall if you "+
			"would like these configuration files to be updated.",
			filterName)
	} else if _, err := os.Stat(remoteDataPath); err == nil {
		// Ensure folder exists
		err = os.MkdirAll(localDataPath, 0666)
		if err != nil {
			Logger.Error("Could not create filter data folder", err) // TODO - I don't think this should break the entire install
		}

		// Copy 'data' to dataPath
		if profile.DataPath != "" {
			err = copy.Copy(
				remoteDataPath, localDataPath,
				copy.Options{PreserveTimes: false, Sync: false})
			if err != nil {
				Logger.Error("Could not initialize filter data", err) // TODO - I don't think this should break the entire install
			}
		} else {
			Logger.Warnf("Filter %s has installation data, but the "+
				"dataPath is not set. Skipping.", filterName)
		}
	}
}

// IsInstalled eturns whether the filter is currently installed or not.
func (f *RemoteFilter) IsInstalled() bool {
	if _, err := os.Stat(f.GetDownloadPath()); err == nil {
		return true
	}
	return false
}

// GetDownloadPath returns the path location where the filter can be found.
func (f *RemoteFilter) GetDownloadPath() string {
	return filepath.Join(".regolith/cache/filters", f.Id)
}

// GetDownloadUrl creates a download URL, based on the filter definition.
func (f *RemoteFilter) GetDownloadUrl() string {
	repoUrl := ""
	if f.GetUrl() == "" {
		repoUrl = StandardLibraryUrl
	} else {
		repoUrl = f.GetUrl()
	}

	repoVersion := ""
	if f.GetVersion() != "" {
		repoVersion = "?ref=" + f.GetVersion()
	}

	return fmt.Sprintf("%s//%s%s", repoUrl, f.Id, repoVersion)
}

// GetIdName returns the name that identifies the filter. This name is used to
// create and access the data folder for the filter. This property only makes
// sense for remote filters. Non-remote filters return empty string.
func (f *RemoteFilter) GetIdName() string {
	if f.Id != "" {
		return f.Id
	}
	return ""
}

func (f *RemoteFilter) Uninstall() {
	err := os.RemoveAll(f.GetDownloadPath())
	if err != nil {
		Logger.Error(wrapError(fmt.Sprintf("Could not remove installed filter %s.", f.GetFriendlyName()), err))
	}
}

// IsCached checks whether the filter of given URL is already saved
// in cache.
func (f *RemoteFilter) IsCached() bool {
	_, err := os.Stat(f.GetDownloadPath())
	return err == nil
}

func (f *RemoteFilter) GetUrl() string {
	installation, ok := installationsMap[f.Id] // evil global variable
	if !ok {
		return StandardLibraryUrl
	}
	return installation.Url
}

func (f *RemoteFilter) GetVersion() string {
	installation, ok := installationsMap[f.Id] // evil global variable
	// If not ok then this means it's a standard filter. Other filters are
	// obligated to have a version or use the "lastest" keyword. This condition
	// is checked when we parse the config.json definition.
	if !ok {
		return "lastest"
	}
	return installation.Version
}
