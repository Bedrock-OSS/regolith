package regolith

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/otiai10/copy"
)

type RemoteFilterDefinition struct {
	FilterDefinition
	Url     string `json:"url,omitempty"`
	Version string `json:"version,omitempty"`
	// RemoteFilters can propagate some of the properties unique to other types
	// of filers (like Python's venvSlot).
	VenvSlot int `json:"venvSlot,omitempty"`
}

type RemoteFilter struct {
	Filter
	Definition RemoteFilterDefinition `json:"-"`
}

func RemoteFilterDefinitionFromObject(id string, obj map[string]interface{}) *RemoteFilterDefinition {
	result := &RemoteFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
	url, ok := obj["url"].(string)
	if !ok {
		Logger.Fatal("could not find url in filter definition %s", id)
	}
	result.Url = url
	version, ok := obj["version"].(string)
	if !ok {
		Logger.Fatal("could not find version in filter definition %s", id)
	}
	result.Version = version
	result.VenvSlot, _ = obj["venvSlot"].(int) // default venvSlot is 0
	return result
}

func RemoteFilterFromObject(
	obj map[string]interface{}, definition RemoteFilterDefinition,
) *RemoteFilter {
	filter := &RemoteFilter{
		Filter:     *FilterFromObject(obj),
		Definition: definition,
	}
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

func (f *RemoteFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) FilterRunner {
	return RemoteFilterFromObject(runConfiguration, *f)
}

func (f *RemoteFilterDefinition) InstallDependencies(parent *RemoteFilterDefinition) error {
	return nil // Remote filters don't install any dependencies
}

func (f *RemoteFilterDefinition) Check() error {
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
func (f *RemoteFilter) CopyFilterData(profile *Profile, dataPath string) {
	// Move filters 'data' folder contents into 'data'
	// If the localDataPath already exists, we must not overwrite
	// Additionally, if the remote data path doesn't exist, we don't need
	// to do anything
	remoteDataPath := path.Join(f.GetDownloadPath(), "data")
	localDataPath := path.Join(dataPath, f.Id)
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
		if dataPath != "" {
			err = copy.Copy(
				remoteDataPath, localDataPath,
				copy.Options{PreserveTimes: false, Sync: false})
			if err != nil {
				Logger.Error("Could not initialize filter data", err) // TODO - I don't think this should break the entire install
			}
		} else {
			Logger.Warnf(
				"Filter %s has installation data, but the "+
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
