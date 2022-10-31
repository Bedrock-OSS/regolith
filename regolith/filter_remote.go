package regolith

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/hashicorp/go-getter"
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

func RemoteFilterDefinitionFromObject(id string, obj map[string]interface{}) (*RemoteFilterDefinition, error) {
	result := &RemoteFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
	url, ok := obj["url"].(string)
	if !ok {
		result.Url = StandardLibraryUrl
	} else {
		result.Url = url
	}
	versionObj, ok := obj["version"]
	if !ok {
		return nil, WrappedErrorf(jsonPropertyMissingError, "version")
	}
	version, ok := versionObj.(string)
	if !ok {
		return nil, WrappedErrorf(jsonPropertyTypeError, "version", "string")
	}
	result.Version = version
	result.VenvSlot, _ = obj["venvSlot"].(int) // default venvSlot is 0
	return result, nil
}

func (f *RemoteFilter) run(context RunContext) error {
	Logger.Debugf("RunRemoteFilter \"%s\"", f.Definition.Url)
	if !f.IsCached(context.DotRegolithPath) {
		return WrappedErrorf(
			"Filter is not downloaded. "+
				"You can download filter files using command:\n"+
				"regolith install %s", f.Id)
	}

	version, err := f.GetCachedVersion(context.DotRegolithPath)
	if err != nil {
		return WrapErrorf(
			err, "Failed check the version of the filter in cache."+
				"\nFilter: %s\n"+
				"You can try to force reinstallation fo the filter using command:"+
				"regolith install --force %s", f.Id, f.Id)
	}
	if f.Definition.Version != "HEAD" && f.Definition.Version != "latest" && f.Definition.Version != *version {
		return WrappedErrorf(
			"Filter version saved in cache doesn't match the version declared"+
				" in the config file.\n"+
				"Installed version: %s\n"+
				"Required version: %s\n"+
				"You can update the filter using command:\n"+
				"regolith update %s",
			// cached, required, id
			*version, f.Definition.Version, f.Id)
	}

	path := f.GetDownloadPath(context.DotRegolithPath)
	absolutePath, _ := filepath.Abs(path)
	filterCollection, err := f.subfilterCollection(context.DotRegolithPath)
	if err != nil {
		return WrapErrorf(err, remoteFilterSubfilterCollectionError)
	}
	for i, filter := range filterCollection.Filters {
		// Disabled filters are skipped
		disabled, err := filter.IsDisabled()
		if err != nil {
			return WrapErrorf(err, "Failed to check if filter is disabled")
		}
		if disabled {
			Logger.Infof(
				"The %s subfilter of \"%s\" filter is disabled, skipping.",
				nth(i), f.Id)
			continue
		}
		// Overwrite the venvSlot with the parent value
		// TODO - remote filters can contain multiple filters, the interruption
		// chceck should be performed after every subfilter
		_, err = filter.Run(RunContext{
			Config:           context.Config,
			AbsoluteLocation: absolutePath,
			Profile:          context.Profile,
			Parent:           context.Parent,
			DotRegolithPath:  context.DotRegolithPath,
		})
		if err != nil {
			return WrapErrorf(
				err, filterRunnerRunError,
				NiceSubfilterName(f.Id, i))
		}
	}
	return nil
}

func (f *RemoteFilter) Run(context RunContext) (bool, error) {
	if err := f.run(context); err != nil {
		return false, PassError(err)
	}
	return context.IsInterrupted(), nil
}

func (f *RemoteFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) (FilterRunner, error) {
	basicFilter, err := filterFromObject(runConfiguration)
	if err != nil {
		return nil, WrapError(err, filterFromObjectError)
	}
	filter := &RemoteFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

// TODO - this code is almost a duplicate of the code in the
// (f *RemoteFilter) SubfilterCollection()
func (f *RemoteFilterDefinition) InstallDependencies(_ *RemoteFilterDefinition, dotRegolithPath string) error {
	path := filepath.Join(f.GetDownloadPath(dotRegolithPath), "filter.json")
	file, err := ioutil.ReadFile(path)

	if err != nil {
		return WrapErrorf(err, fileReadError, path)
	}

	var filterCollection map[string]interface{}
	err = json.Unmarshal(file, &filterCollection)
	if err != nil {
		return WrapErrorf(err, jsonUnmarshalError, path)
	}

	// Filters
	filtersObj, ok := filterCollection["filters"]
	if !ok {
		return extraFilterJsonErrorInfo(
			path, WrappedErrorf(jsonPathMissingError, "filters"))
	}
	filters, ok := filtersObj.([]interface{})
	if !ok {
		return extraFilterJsonErrorInfo(
			path, WrappedErrorf(jsonPathTypeError, "filters", "array"))
	}
	for i, filter := range filters {
		filter, ok := filter.(map[string]interface{})
		jsonPath := fmt.Sprintf("filters->%d", i) // Used for error messages
		if !ok {
			return extraFilterJsonErrorInfo(
				path, WrappedErrorf(jsonPathTypeError, jsonPath, "object"))
		}
		filterInstaller, err := FilterInstallerFromObject(
			fmt.Sprintf("%v:subfilter%v", f.Id, i), filter)
		if err != nil {
			return extraFilterJsonErrorInfo(
				path, WrapErrorf(err, jsonPathParseError, jsonPath))
		}
		err = filterInstaller.InstallDependencies(f, dotRegolithPath)
		if err != nil {
			// This is not parsing error so extraErrorInfo is not necessary
			return WrapErrorf(
				err,
				"Failed to install the dependencies of the %s subfilter.\n"+
					"Filter configuration file: %s\n"+
					"JSON path: %s",
				nth(i), path, jsonPath)
		}
	}
	return nil
}

func (f *RemoteFilterDefinition) Check(context RunContext) error {
	dummyFilterRunner, err := f.CreateFilterRunner(
		map[string]interface{}{"filter": f.Id})
	const shouldntHappenError = ("Filter name: %s\n" +
		"This is a bug, please submit a bug report to the Regolith " +
		"project repository:\n" +
		"https://github.com/Bedrock-OSS/regolith/issues")
	if err != nil { // Shouldn't happen but just in case it's better to check
		return WrapErrorf(
			err, "Failed to create FilterRunner for the filter.\n"+
				shouldntHappenError, f.Id)
	}
	dummyFilterRunnerConverted, ok := dummyFilterRunner.(*RemoteFilter)
	if !ok { // Shouldn't happen but just in case it's better to check
		return WrappedErrorf(
			"Failed to convert to RemoteFilter.\n"+shouldntHappenError, f.Id)
	}
	filterCollection, err := dummyFilterRunnerConverted.subfilterCollection(
		context.DotRegolithPath)
	if err != nil {
		return WrapError(err, remoteFilterSubfilterCollectionError)
	}
	for i, filter := range filterCollection.Filters {
		// Overwrite the venvSlot with the parent value
		err := filter.Check(context)
		if err != nil {
			return WrapErrorf(
				err, filterRunnerCheckError, NiceSubfilterName(f.Id, i))
		}
	}
	return nil
}

func (f *RemoteFilter) Check(context RunContext) error {
	return f.Definition.Check(context)
}

// CopyFilterData copies the filter's data to the data folder.
func (f *RemoteFilterDefinition) CopyFilterData(dataPath string, dotRegolithPath string) {
	// Move filters 'data' folder contents into 'data'
	// If the localDataPath already exists, we must not overwrite
	// Additionally, if the remote data path doesn't exist, we don't need
	// to do anything
	remoteDataPath := path.Join(f.GetDownloadPath(dotRegolithPath), "data")
	localDataPath := path.Join(dataPath, f.Id)
	if _, err := os.Stat(localDataPath); err == nil {
		Logger.Warnf(
			"Filter already has data in its data folder.\n"+
				"Filter name: %s\n"+
				"Filter data folder: %s\n"+
				"If you want to download the default data from filter's "+
				"repository, remove the data folder manually and reinstall the "+
				"filter.", f.Id, localDataPath)
	} else if _, err := os.Stat(remoteDataPath); err == nil {
		// Ensure folder exists
		err = os.MkdirAll(localDataPath, 0755)
		if err != nil {
			Logger.Error("Could not create filter data folder.", err)
		}

		// Copy 'data' to dataPath
		if dataPath != "" {
			err = copy.Copy(
				remoteDataPath, localDataPath,
				copy.Options{PreserveTimes: false, Sync: false})
			if err != nil {
				Logger.Error("Could not initialize filter data.", err)
			}
		} else {
			Logger.Warnf(
				"Filter \"%s\" has installation data, but the "+
					"dataPath is not set. Skipping.", f.Id)
		}
	}
}

// GetDownloadPath returns the path location where the filter can be found.
func (f *RemoteFilter) GetDownloadPath(dotRegolithPath string) string {
	return filepath.Join(filepath.Join(dotRegolithPath, "cache/filters"), f.Id)
}

// IsCached checks whether the filter of given URL is already saved
// in cache.
func (f *RemoteFilter) IsCached(dotRegolithPath string) bool {
	_, err := os.Stat(f.GetDownloadPath(dotRegolithPath))
	return err == nil
}

// GetCachedVersion returns cached version of the remote filter.
func (f *RemoteFilter) GetCachedVersion(dotRegolithPath string) (*string, error) {
	path := filepath.Join(f.GetDownloadPath(dotRegolithPath), "filter.json")
	file, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, WrapErrorf(err, fileReadError, path)
	}

	var filterCollection map[string]interface{}
	err = json.Unmarshal(file, &filterCollection)
	if err != nil {
		return nil, WrapErrorf(err, jsonUnmarshalError, file)
	}
	versionObj, ok := filterCollection["version"]
	if !ok {
		return nil, extraFilterJsonErrorInfo(
			path, WrappedErrorf(jsonPathMissingError, "version"))
	}
	version, ok := versionObj.(string)
	if !ok {
		return nil, extraFilterJsonErrorInfo(
			path, WrappedErrorf(jsonPathTypeError, "version", "string"))
	}
	return &version, nil
}

// FilterDefinitionFromTheInternet downloads a filter from the internet and
// returns its data.
func FilterDefinitionFromTheInternet(
	url, name, version string,
) (*RemoteFilterDefinition, error) {
	var err error
	if version == "" { // "" locks the version to the latest
		version, err = GetRemoteFilterDownloadRef(url, name, version)
		if err != nil {
			return nil, WrappedErrorf(
				getRemoteFilterDownloadRefError, url, name, version)
		}
		version = trimFilterPrefix(version, name)
	}
	return &RemoteFilterDefinition{
		FilterDefinition: FilterDefinition{Id: name},
		Version:          version,
		Url:              url,
	}, nil
}

// Download
func (i *RemoteFilterDefinition) Download(
	isForced bool, dotRegolithPath string,
) error {
	if _, err := os.Stat(i.GetDownloadPath(dotRegolithPath)); err == nil {
		if !isForced {
			Logger.Warnf(
				"The download path of the \"%s\" already exists.This should "+
					"be the case only if the filter is installed.\n"+
					"    Skipped the download. You can force the it by "+
					"passing the \"-force\" flag.", i.Id)
			return nil
		} else {
			i.Uninstall(dotRegolithPath)
		}
	}

	Logger.Infof("Downloading filter %s...", i.Id)

	// Download the filter using Git Getter
	if !hasGit() {
		return WrappedError(gitNotInstalledWarning)
	}
	repoVersion, err := GetRemoteFilterDownloadRef(i.Url, i.Id, i.Version)
	if err != nil {
		return WrapErrorf(
			err, getRemoteFilterDownloadRefError, i.Url, i.Id, i.Version)
	}
	url := fmt.Sprintf("%s//%s?ref=%s", i.Url, i.Id, repoVersion)
	downloadPath := i.GetDownloadPath(dotRegolithPath)

	_, err = os.Stat(downloadPath)
	downloadPathIsNew := os.IsNotExist(err)
	err = getter.Get(downloadPath, url)
	if err != nil {
		if downloadPathIsNew { // Remove the path created by getter
			os.Remove(downloadPath)
		}
		return WrapErrorf(
			err, "Could not download filter from %s.\n"+
				"Does that filter exist?", url)
	}
	// Save the version of the filter we downloaded
	i.SaveVerssionInfo(trimFilterPrefix(repoVersion, i.Id), dotRegolithPath)
	// Remove 'test' folder, which we never want to use (saves space on disk)
	testFolder := path.Join(downloadPath, "test")
	if _, err := os.Stat(testFolder); err == nil {
		os.RemoveAll(testFolder)
	}

	Logger.Infof("Filter \"%s\" downloaded successfully.", i.Id)
	return nil
}

// SaveVersionInfo saves puts the specified version string into the
// filter.json of the remote fileter.
func (i *RemoteFilterDefinition) SaveVerssionInfo(version, dotRegolithPath string) error {
	filterJsonMap, err := i.LoadFilterJson(dotRegolithPath)
	if err != nil {
		return WrapErrorf(
			err, "Could not load filter.json for \"%s\" filter.", i.Id)
	}
	filterJsonMap["version"] = version
	filterJson, _ := json.MarshalIndent(filterJsonMap, "", "\t") // no error
	filterJsonPath := path.Join(i.GetDownloadPath(dotRegolithPath), "filter.json")
	err = os.WriteFile(filterJsonPath, filterJson, 0644)
	if err != nil {
		return WrapErrorf(
			err, "Unable to write \"filter.json\" for %q filter.", i.Id)
	}
	return nil
}

// LoadFilterJson loads the filter.json file of the remote filter to a map.
func (f *RemoteFilterDefinition) LoadFilterJson(dotRegolithPath string) (map[string]interface{}, error) {
	downloadPath := f.GetDownloadPath(dotRegolithPath)
	filterJsonPath := path.Join(downloadPath, "filter.json")
	filterJson, err1 := ioutil.ReadFile(filterJsonPath)
	var filterJsonMap map[string]interface{}
	err2 := json.Unmarshal(filterJson, &filterJsonMap)
	if err := firstErr(err1, err2); err != nil {
		return nil, PassError(err)
	}
	return filterJsonMap, nil
}

// GetInstalledVersion reads the version seaved in the filter.json
func (f *RemoteFilterDefinition) InstalledVersion(dotRegolithPath string) (string, error) {
	filterJsonMap, err := f.LoadFilterJson(dotRegolithPath)
	if err != nil {
		return "", WrapErrorf(
			err, "Could not load filter.json for %q filter.", f.Id)
	}
	version, ok1 := filterJsonMap["version"]
	versionStr, ok2 := version.(string)
	if !ok1 || !ok2 {
		return "", WrappedErrorf(
			"Could not read \"version\" from filter.json for %q filter",
			f.Id)
	}
	return versionStr, nil
}

func (f *RemoteFilterDefinition) Update(dotRegolithPath string) error {
	installedVersion, err := f.InstalledVersion(dotRegolithPath)
	installedVersion = trimFilterPrefix(installedVersion, f.Id)
	if err != nil {
		Logger.Warnf("Unable to get installed version of filter %q.", f.Id)
	}
	version, err := GetRemoteFilterDownloadRef(f.Url, f.Id, f.Version)
	if err != nil {
		return WrapErrorf(
			err, getRemoteFilterDownloadRefError, f.Url, f.Id, f.Version)
	}
	version = trimFilterPrefix(version, f.Id)
	if installedVersion != version {
		Logger.Infof(
			"Updating filter %q to new version: %q->%q.",
			f.Id, installedVersion, version)
		err = f.Download(true, dotRegolithPath)
		if err != nil {
			return PassError(err)
		}
		err = f.InstallDependencies(f, dotRegolithPath)
		if err != nil {
			return PassError(err)
		}
		Logger.Infof("Filter %q updated successfully.", f.Id)
	} else {
		Logger.Infof(
			"Filter %q is up to date. Installed version: %q.",
			f.Id, installedVersion)
	}
	return nil
}

// GetDownloadPath returns the path location where the filter can be found.
func (i *RemoteFilterDefinition) GetDownloadPath(dotRegolithPath string) string {
	return filepath.Join(filepath.Join(dotRegolithPath, "cache/filters"), i.Id)
}

func (i *RemoteFilterDefinition) Uninstall(dotRegolithPath string) {
	downloadPath := i.GetDownloadPath(dotRegolithPath)
	err := os.RemoveAll(downloadPath)
	if err != nil {
		Logger.Error(
			WrapErrorf(err, osRemoveError, downloadPath))
	}
}

// hasGit returns whether git is installed or not.
func hasGit() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// extraFilterJsonErrorInfo is used to wrap errors related to parsing the
// filter.json file. It's common for other functions to handle loading and
// parsing of this file, so using this is necessary to provide both the
// finformation about the file path and reuse the errrors from errors.go
//
// TODO - this is an ugly solution, perhaps we should have a separate
// function for loading the filter.json file. Currently it's always build into
// other functions.
func extraFilterJsonErrorInfo(filterJsonFilePath string, err error) error {
	return WrapErrorf(
		err, "Failed to load the filter configuration.\n"+
			"Filter configuration file: %s", filterJsonFilePath)
}
