package regolith

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/otiai10/copy"
)

type RemoteFilter struct {
	Filter

	Id      string `json:"filter,omitempty"`
	Url     string `json:"-"`
	Version string `json:"-"`
	// RemoteFilters can propagate some of the properties unique to other types
	// of filers (like Python's venvSlot).
	VenvSlot int `json:"venvSlot,omitempty"`
}

func RemoteFilterFromObject(
	obj map[string]interface{}, installations map[string]Installation,
) *RemoteFilter {
	filter := &RemoteFilter{Filter: *FilterFromObject(obj)}
	id, ok := obj["filter"].(string)
	if !ok {
		Logger.Fatalf(
			"remote filter %q is missing \"filter\" field",
			filter.GetFriendlyName())
	}
	filter.Id = id
	installation, ok := installations[id]
	if ok {
		filter.Url = installation.Url
		filter.Version = installation.Version
	} else {
		filter.Url = StandardLibraryUrl
		filter.Version = "latest"
	}

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
	if f.Url != StandardLibraryUrl && !IsUnlocked() {
		return errors.New(
			"safe mode is on, which protects you from potentially unsafe " +
				"code.\nYou may turn it off using 'regolith unlock'",
		)
	}
	Logger.Infof("Running filter %s", f.GetFriendlyName())
	start := time.Now()
	defer Logger.Debugf("Executed in %s", time.Since(start))

	Logger.Debugf("RunRemoteFilter '%s'", f.Url)
	if !f.IsCached() {
		return errors.New(
			"filter is not downloaded! Please run 'regolith install'")
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
	// We don't support nested remote filters anymore so this function is
	// never called.
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
	_, end := path.Split(f.Url) // Return the last part of the URL
	return end
}

// CopyFilterData copies the filter's data to the data folder.
func (f *RemoteFilter) CopyFilterData(profile *Profile) {
	// Move filters 'data' folder contents into 'data'
	// If the localDataPath already exists, we must not overwrite
	// Additionally, if the remote data path doesn't exist, we don't need
	// to do anything
	remoteDataPath := path.Join(f.GetDownloadPath(), "data")
	localDataPath := path.Join(profile.DataPath, f.Id)
	if _, err := os.Stat(localDataPath); err == nil {
		Logger.Warnf("Filter %s already has data in the 'data' folder. \n"+
			"You may manually delete this data and reinstall if you "+
			"would like these configuration files to be updated.",
			f.Id)
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
				"dataPath is not set. Skipping.", f.Id)
		}
	}
}

// GetDownloadPath returns the path location where the filter can be found.
func (f *RemoteFilter) GetDownloadPath() string {
	return filepath.Join(".regolith/cache/filters", f.Id)
}

// IsCached checks whether the filter of given URL is already saved
// in cache.
func (f *RemoteFilter) IsCached() bool {
	_, err := os.Stat(f.GetDownloadPath())
	return err == nil
}
