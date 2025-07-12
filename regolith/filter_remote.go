package regolith

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bedrock-OSS/go-burrito/burrito"

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
		return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "version")
	}
	version, ok := versionObj.(string)
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyTypeError, "version", "string")
	}
	result.Version = version
	venvSlot64, _ := obj["venvSlot"].(float64) // default venvSlot is 0.0
	result.VenvSlot = int(venvSlot64)

	return result, nil
}

// run executes all subfilters of the remote filter. It returns true if the
// execution was interrupted via the RunContext.
func (f *RemoteFilter) run(context RunContext) (bool, error) {
	Logger.Debugf("RunRemoteFilter \"%s\"", f.Definition.Url)
	if !f.IsCached(context.DotRegolithPath) {
		return false, burrito.WrappedErrorf(
			"Filter is not downloaded. "+
				"You can download filter files using command:\n"+
				"regolith install %s", f.Id)
	}

	version, err := f.GetCachedVersion(context.DotRegolithPath)
	if err != nil {
		return false, burrito.WrapErrorf(
			err, "Failed check the version of the filter in cache."+
				"\nFilter: %s\n"+
				"You can try to force reinstallation fo the filter using command:"+
				"regolith install --force %s", f.Id, f.Id)
	}
	if f.Definition.Version != "HEAD" && f.Definition.Version != "latest" && f.Definition.Version != *version {
		return false, burrito.WrappedErrorf(
			"Filter version saved in cache doesn't match the version declared"+
				" in the config file.\n"+
				"Filter: %s\n"+
				"Installed version: %s\n"+
				"Required version: %s\n"+
				"You update all of the filters by running:\n"+
				"regolith install-all",
			// id, cached, required
			f.Id, *version, f.Definition.Version)
	}

	path := f.GetDownloadPath(context.DotRegolithPath)
	absolutePath, _ := filepath.Abs(path)
	filterCollection, err := f.subfilterCollection(context.DotRegolithPath)
	if err != nil {
		return false, burrito.WrapErrorf(err, remoteFilterSubfilterCollectionError)
	}
	for i, filter := range filterCollection.Filters {
		runContext := RunContext{
			Config:           context.Config,
			AbsoluteLocation: absolutePath,
			Profile:          context.Profile,
			Parent:           context.Parent,
			DotRegolithPath:  context.DotRegolithPath,
			Settings:         filter.GetSettings(),
		}
		// Disabled filters are skipped
		disabled, err := filter.IsDisabled(runContext)
		if err != nil {
			return false, burrito.WrapErrorf(err, "Failed to check if filter is disabled")
		}
		if disabled {
			Logger.Debugf(
				"The %s subfilter of \"%s\" filter is disabled, skipping.",
				nth(i), f.Id)
			continue
		}
		// Overwrite the venvSlot with the parent value
		// TODO - remote filters can contain multiple filters, the interruption
		// check should be performed after every subfilter
		_, err = filter.Run(runContext)
		if err != nil {
			return false, burrito.WrapErrorf(
				err, filterRunnerRunError,
				NiceSubfilterName(f.Id, i))
		}
		if context.IsInterrupted() {
			return true, nil
		}
	}
	return false, nil
}

func (f *RemoteFilter) Run(context RunContext) (bool, error) {
	interrupted, err := f.run(context)
	if err != nil {
		return false, burrito.PassError(err)
	}
	if interrupted {
		return true, nil
	}
	return context.IsInterrupted(), nil
}

func (f *RemoteFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}, id string) (FilterRunner, error) {
	basicFilter, err := filterFromObject(runConfiguration, id)
	if err != nil {
		return nil, burrito.WrapError(err, filterFromObjectError)
	}
	filter := &RemoteFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *RemoteFilterDefinition) InstallDependencies(_ *RemoteFilterDefinition, dotRegolithPath string) error {
	path := filepath.Join(f.GetDownloadPath(dotRegolithPath), "filter.json")
	filterCollection, err := loadFilterConfig(path)
	if err != nil {
		return burrito.PassError(err)
	}

	// Filters
	filtersObj, ok := filterCollection["filters"]
	if !ok {
		return extraFilterJsonErrorInfo(
			path, burrito.WrappedErrorf(jsonPathMissingError, "filters"))
	}
	filters, ok := filtersObj.([]interface{})
	if !ok {
		return extraFilterJsonErrorInfo(
			path, burrito.WrappedErrorf(jsonPathTypeError, "filters", "array"))
	}
	for i, filter := range filters {
		filter, ok := filter.(map[string]interface{})
		jsonPath := fmt.Sprintf("filters->%d", i) // Used for error messages
		if !ok {
			return extraFilterJsonErrorInfo(
				path, burrito.WrappedErrorf(jsonPathTypeError, jsonPath, "object"))
		}
		if runWith, ok := filter["runWith"]; !ok || runWith == "" {
			return burrito.WrappedErrorf(
				"Nested remote filters are not supported.",
				"Filter: %s", f.Id)
		}
		filterInstaller, err := FilterInstallerFromObject(
			fmt.Sprintf("%v:subfilter%v", f.Id, i), filter)
		if err != nil {
			return extraFilterJsonErrorInfo(
				path, burrito.WrapErrorf(err, jsonPathParseError, jsonPath))
		}
		err = filterInstaller.InstallDependencies(f, dotRegolithPath)
		if err != nil {
			// This is not parsing error so extraErrorInfo is not necessary
			return burrito.WrapErrorf(
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
		map[string]interface{}{}, f.Id)
	const shouldntHappenError = "Filter name: %s\n" +
		"This is a bug, please submit a bug report to the Regolith " +
		"project repository:\n" +
		"https://github.com/Bedrock-OSS/regolith/issues"
	if err != nil { // Shouldn't happen but just in case it's better to check
		return burrito.WrapErrorf(
			err, "Failed to create FilterRunner for the filter.\n"+
				shouldntHappenError, f.Id)
	}
	dummyFilterRunnerConverted, ok := dummyFilterRunner.(*RemoteFilter)
	if !ok { // Shouldn't happen but just in case it's better to check
		return burrito.WrappedErrorf(
			"Failed to convert to RemoteFilter.\n"+shouldntHappenError, f.Id)
	}
	filterCollection, err := dummyFilterRunnerConverted.subfilterCollection(
		context.DotRegolithPath)
	if err != nil {
		return burrito.WrapError(err, remoteFilterSubfilterCollectionError)
	}
	for i, filter := range filterCollection.Filters {
		// Overwrite the venvSlot with the parent value
		err := filter.Check(context)
		if err != nil {
			return burrito.WrapErrorf(
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
		Logger.Warn(burrito.WrappedErrorf(
			"Filter already has data in its data folder.\n"+
				"Filter name: %s\n"+
				"Filter data folder: %s\n"+
				"If you want to download the default data from filter's "+
				"repository, remove the data folder manually and reinstall the "+
				"filter.", f.Id, localDataPath))
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
	file, err := os.ReadFile(path)

	if err != nil {
		return nil, burrito.WrapErrorf(err, fileReadError, path)
	}

	var filterCollection map[string]interface{}
	err = json.Unmarshal(file, &filterCollection)
	if err != nil {
		return nil, burrito.WrapErrorf(err, jsonUnmarshalError, file)
	}
	versionObj, ok := filterCollection["version"]
	if !ok {
		return nil, extraFilterJsonErrorInfo(
			path, burrito.WrappedErrorf(jsonPathMissingError, "version"))
	}
	version, ok := versionObj.(string)
	if !ok {
		return nil, extraFilterJsonErrorInfo(
			path, burrito.WrappedErrorf(jsonPathTypeError, "version", "string"))
	}
	return &version, nil
}

func (f *RemoteFilter) IsUsingDataExport(dotRegolithPath string, _ RunContext) (bool, error) {
	// Load the filter.json file
	filterJsonPath := filepath.Join(f.GetDownloadPath(dotRegolithPath), "filter.json")
	file, err := os.ReadFile(filterJsonPath)
	if err != nil {
		return false, burrito.WrappedErrorf(readFilterJsonError, filterJsonPath)
	}
	var filterJsonObj map[string]interface{}
	err = json.Unmarshal(file, &filterJsonObj)
	if err != nil {
		return false, burrito.WrapErrorf(err, jsonUnmarshalError, filterJsonPath)
	}
	// Get the exportData field (default to false)
	exportDataObj, ok := filterJsonObj["exportData"]
	if !ok {
		return false, nil
	}
	exportData, ok := exportDataObj.(bool)
	if !ok {
		return false, burrito.WrappedErrorf(
			jsonPathTypeError, "exportData", "bool")
	}
	return exportData, nil
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
			return nil, burrito.WrapErrorf(
				err, getRemoteFilterDownloadRefError, url, name, version)
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
func (f *RemoteFilterDefinition) Download(
	isForced bool, dotRegolithPath string, refreshFilters bool,
) error {
	if _, err := os.Stat(f.GetDownloadPath(dotRegolithPath)); err == nil {
		if !isForced {
			Logger.Warnf(
				"The download path of the \"%s\" already exists.This should "+
					"be the case only if the filter is installed.\n"+
					"    Skipped the download. You can force the it by "+
					"passing the \"-force\" flag.", f.Id)
			return nil
		} else {
			f.Uninstall(dotRegolithPath)
		}
	}

	Logger.Infof("Downloading filter %s...", f.Id)

	// Download the filter using Git Getter
	MeasureStart("Check git")
	if !hasGit() {
		return burrito.WrappedError(gitNotInstalledWarning)
	}
	MeasureStart("Get remote filter download ref")
	repoVersion, err := GetRemoteFilterDownloadRef(f.Url, f.Id, f.Version)
	if err != nil {
		return burrito.WrapErrorf(
			err, getRemoteFilterDownloadRefError, f.Url, f.Id, f.Version)
	}
	url := fmt.Sprintf("https://%s", f.Url)
	downloadPath := f.GetDownloadPath(dotRegolithPath)

	_, err = os.Stat(downloadPath)
	downloadPathIsNew := os.IsNotExist(err)
	err = downloadFilterRepository(downloadPath, url, repoVersion, f.Id, refreshFilters)
	if err != nil {
		if downloadPathIsNew { // Remove the path created by getter
			os.Remove(downloadPath)
		}
		return burrito.WrapErrorf(
			err, "Could not download filter from %s.\n"+
				"Does that filter exist?", f.Url)
	}
	// Save the version of the filter we downloaded
	MeasureStart("Save version info")
	err = f.SaveVersionInfo(trimFilterPrefix(repoVersion, f.Id), dotRegolithPath)
	if err != nil {
		return burrito.PassError(err)
	}
	MeasureEnd()
	// Remove 'test' folder, which we never want to use (saves space on disk)
	testFolder := path.Join(downloadPath, "test")
	if _, err := os.Stat(testFolder); err == nil {
		os.RemoveAll(testFolder)
	}

	Logger.Infof("Filter \"%s\" downloaded successfully.", f.Id)
	return nil
}

func downloadFilterRepository(downloadPath, url, ref, filter string, forceUpdate bool) error {
	config, err := getCombinedUserConfig()
	if err != nil {
		return burrito.WrapErrorf(err, getUserConfigError)
	}
	cooldown, err := time.ParseDuration(*config.ResolverCacheUpdateCooldown)

	cache, err := getFilterCache(url)
	if err != nil {
		return burrito.WrapErrorf(err, "Could not get cache path for %s", url)
	}
	// Check if exists in cache
clone:
	if _, err := os.Stat(cache); err != nil && os.IsNotExist(err) {
		err := os.MkdirAll(cache, 0755)
		if err != nil {
			return burrito.WrapErrorf(err, osMkdirError, cache)
		}
		// Clone the repository
		MeasureStart("Clone repository %s", url)
		output, err := RunGitProcess([]string{"clone", url, "."}, cache)
		if err != nil {
			Logger.Error(strings.Join(output, "\n"))
			return burrito.WrapErrorf(err, "Failed to clone repository.\nURL: %s", url)
		}
		forceUpdate = false
	} else if err != nil {
		return burrito.WrapErrorf(err, osStatErrorAny, cache)
	}
	info, _ := os.Stat(cache)
	if forceUpdate || info.ModTime().Before(time.Now().Add(cooldown*-1)) {
		// Fetch the repository
		MeasureStart("Fetch repository %s", url)
		output, err := RunGitProcess([]string{"fetch"}, cache)
		if err != nil {
			Logger.Error(strings.Join(output, "\n"))
			Logger.Errorf("Failed to fetch repository.\nURL: %s", url)
			Logger.Infof("Trying to clone the repository instead...")
			err := os.RemoveAll(cache)
			if err != nil {
				return burrito.WrapErrorf(err, osRemoveError, cache)
			}
			goto clone
		}
		err = os.Chtimes(cache, time.Now(), time.Now())
		if err != nil {
			Logger.Debugf(osChtimesError, cache)
		}
		// Fetch the repository
		MeasureStart("Fetch repository tags %s", url)
		output, err = RunGitProcess([]string{"fetch", "--tags"}, cache)
		if err != nil {
			Logger.Error(strings.Join(output, "\n"))
			Logger.Errorf("Failed to fetch repository.\nURL: %s", url)
			Logger.Infof("Trying to clone the repository instead...")
			err := os.RemoveAll(cache)
			if err != nil {
				return burrito.WrapErrorf(err, osRemoveError, cache)
			}
			goto clone
		}
		err = os.Chtimes(cache, time.Now(), time.Now())
		if err != nil {
			Logger.Debugf(osChtimesError, cache)
		}
	}
	// Checkout the specified ref
	MeasureStart("Checkout ref %s", ref)
	output, err := RunGitProcess([]string{"checkout", ref}, cache)
	if err != nil {
		Logger.Error(strings.Join(output, "\n"))
		return burrito.WrapErrorf(err, "Failed to checkout ref.\nURL: %s\nRef: %s", url, ref)
	}
	// Copy to download path
	MeasureStart("Copy to download path %s", downloadPath)
	err = copy.Copy(filepath.Join(cache, filter), downloadPath)
	if err != nil {
		return burrito.WrapErrorf(err, osCopyError, filepath.Join(cache, filter), downloadPath)
	}
	MeasureEnd()
	return nil
}

// SaveVersionInfo saves puts the specified version string into the
// filter.json of the remote filter.
func (f *RemoteFilterDefinition) SaveVersionInfo(version, dotRegolithPath string) error {
	filterJsonMap, err := f.LoadFilterJson(dotRegolithPath)
	if err != nil {
		return burrito.WrapErrorf(
			err, "Could not load filter.json for \"%s\" filter.", f.Id)
	}
	filterJsonMap["version"] = version
	filterJson, _ := json.MarshalIndent(filterJsonMap, "", "\t") // no error
	filterJsonPath := path.Join(f.GetDownloadPath(dotRegolithPath), "filter.json")
	err = os.WriteFile(filterJsonPath, filterJson, 0644)
	if err != nil {
		return burrito.WrapErrorf(
			err, "Unable to write \"filter.json\" for %q filter.", f.Id)
	}
	return nil
}

// LoadFilterJson loads the filter.json file of the remote filter to a map.
func (f *RemoteFilterDefinition) LoadFilterJson(dotRegolithPath string) (map[string]interface{}, error) {
	downloadPath := f.GetDownloadPath(dotRegolithPath)
	filterJsonPath := path.Join(downloadPath, "filter.json")
	filterJson, err1 := os.ReadFile(filterJsonPath)
	var filterJsonMap map[string]interface{}
	err2 := json.Unmarshal(filterJson, &filterJsonMap)
	if err := firstErr(err1, err2); err != nil {
		return nil, burrito.PassError(err)
	}
	return filterJsonMap, nil
}

// InstalledVersion reads the version saved in the filter.json
func (f *RemoteFilterDefinition) InstalledVersion(dotRegolithPath string) (string, error) {
	filterJsonMap, err := f.LoadFilterJson(dotRegolithPath)
	if err != nil {
		return "", burrito.WrapErrorf(
			err, "Could not load filter.json for %q filter.", f.Id)
	}
	version, ok1 := filterJsonMap["version"]
	versionStr, ok2 := version.(string)
	if !ok1 || !ok2 {
		return "", burrito.WrappedErrorf(
			"Could not read \"version\" from filter.json for %q filter",
			f.Id)
	}
	return versionStr, nil
}

func (f *RemoteFilterDefinition) Update(force bool, dotRegolithPath, dataPath string, refreshFilters bool) error {
	installedVersion, err := f.InstalledVersion(dotRegolithPath)
	installedVersion = trimFilterPrefix(installedVersion, f.Id)
	if err != nil && force {
		Logger.Warnf("Unable to get installed version of filter %q.", f.Id)
	}
	MeasureStart("Get remote filter download ref")
	version, err := GetRemoteFilterDownloadRef(f.Url, f.Id, f.Version)
	if err != nil {
		return burrito.WrapErrorf(
			err, getRemoteFilterDownloadRefError, f.Url, f.Id, f.Version)
	}
	MeasureEnd()
	version = trimFilterPrefix(version, f.Id)
	if installedVersion != version || force {
		Logger.Infof(
			"Updating filter %q to new version: %q->%q.",
			f.Id, installedVersion, version)
		err = f.Download(true, dotRegolithPath, refreshFilters)
		if err != nil {
			return burrito.PassError(err)
		}
		// Copy the data of the remote filter to the data path
		f.CopyFilterData(dataPath, dotRegolithPath)
		err = f.InstallDependencies(f, dotRegolithPath)
		if err != nil {
			return burrito.PassError(err)
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
func (f *RemoteFilterDefinition) GetDownloadPath(dotRegolithPath string) string {
	return filepath.Join(filepath.Join(dotRegolithPath, "cache/filters"), f.Id)
}

func (f *RemoteFilterDefinition) Uninstall(dotRegolithPath string) {
	Logger.Debugf("Uninstalling filter %q.", f.Id)
	downloadPath := f.GetDownloadPath(dotRegolithPath)
	err := os.RemoveAll(downloadPath)
	if err != nil {
		Logger.Error(
			burrito.WrapErrorf(err, osRemoveError, downloadPath))
	}
}

// hasGit returns whether git is installed or not.

func hasGit() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// loadFilterConfig loads the remote filter configuration from the given path.
func loadFilterConfig(path string) (map[string]interface{}, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, burrito.WrapErrorf(err, fileReadError, path)
	}
	var filterCollection map[string]interface{}
	err = json.Unmarshal(file, &filterCollection)
	if err != nil {
		return nil, burrito.WrapErrorf(err, jsonUnmarshalError, path)
	}
	return filterCollection, nil
}

// extraFilterJsonErrorInfo is used to wrap errors related to parsing the
// filter.json file. It's common for other functions to handle loading and
// parsing of this file, so using this is necessary to provide both the
// information about the file path and reuse the errors from errors.go
func extraFilterJsonErrorInfo(filterJsonFilePath string, err error) error {
	return burrito.WrapErrorf(
		err, "Failed to load the filter configuration.\n"+
			"Filter configuration file: %s", filterJsonFilePath)
}
