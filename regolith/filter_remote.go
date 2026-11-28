package regolith

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
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

func RemoteFilterDefinitionFromObject(id string, obj map[string]any) (*RemoteFilterDefinition, error) {
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
			UnsafeMode:       context.UnsafeMode,
			interruption:     context.interruption,
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
		interrupted, err := filter.Run(runContext)
		if err != nil {
			return false, burrito.WrapErrorf(
				err, filterRunnerRunError,
				NiceSubfilterName(f.Id, i))
		}
		if interrupted || context.IsInterrupted() {
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

func (f *RemoteFilterDefinition) CreateFilterRunner(runConfiguration map[string]any, id string) (FilterRunner, error) {
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
	filters, ok := filtersObj.([]any)
	if !ok {
		return extraFilterJsonErrorInfo(
			path, burrito.WrappedErrorf(jsonPathTypeError, "filters", "array"))
	}
	for i, filter := range filters {
		filter, ok := filter.(map[string]any)
		jsonPath := fmt.Sprintf("filters->%d", i) // Used for error messages
		if !ok {
			return extraFilterJsonErrorInfo(
				path, burrito.WrappedErrorf(jsonPathTypeError, jsonPath, "object"))
		}
		if runWith, ok := filter["runWith"]; !ok || runWith == "" {
			return burrito.WrappedErrorf(
				"Nested remote filters are not supported.\n"+
					"Filter: %s", f.Id)
		}
		filterInstaller, err := FilterInstallerFromObject(
			fmt.Sprintf("%v:subfilter%v", f.Id, i), f.Id, filter)
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
		map[string]any{}, f.Id)
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

	var filterCollection map[string]any
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
	var filterJsonObj map[string]any
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

// FilterDefinitionFromTheInternet creates a RemoteFilterDefinition for a
// filter hosted in a Git repository. If version is empty, it resolves the
// latest tag (falling back to the HEAD SHA) using "git ls-remote".
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

	downloadPath := f.GetDownloadPath(dotRegolithPath)
	_, err := os.Stat(downloadPath)
	downloadPathIsNew := os.IsNotExist(err)

	MeasureStart("Get remote filter download ref")
	version, err := GetRemoteFilterDownloadRef(f.Url, f.Id, f.Version)
	if err != nil {
		return burrito.WrapErrorf(
			err, getRemoteFilterDownloadRefError, f.Url, f.Id, f.Version)
	}
	trimmedVersion := trimFilterPrefix(version, f.Id)
	isReleaseVersion := trimmedVersion != version
	// Download using gh
	MeasureStart("Try download with gh")
	ghRepoName := f.getGhRepoName()
	ghDownloadSuccessful := false
	if ghRepoName != "" && hasGhCli() && isReleaseVersion {
		unavailable, err := f.downloadFilterGhRelease(ghRepoName, version, downloadPath, refreshFilters)
		if err != nil {
			// Most filters don't publish release files, so a missing
			// release isn't worth a warning
			logFunc := Logger.Warn
			if unavailable {
				logFunc = Logger.Debug
			}
			logFunc(burrito.WrapErrorf(
				err,
				"Failed to download filter release using 'gh'. Using git instead.\n"+
					"GH repo name: %s\n"+
					"Version: %s",
				ghRepoName, version,
			).Error())
			// Don't let a partially extracted release mix with the files
			// copied by the git fallback
			if err := os.RemoveAll(downloadPath); err != nil {
				return burrito.WrapErrorf(err, osRemoveError, downloadPath)
			}
		} else {
			ghDownloadSuccessful = true
		}
	}
	// Download the filter using Git Getter
	if !ghDownloadSuccessful {
		MeasureStart("Try download with git")
		if !hasGit() {
			return burrito.WrappedError(gitNotInstalledWarning)
		}
		err = f.downloadFilterRepository(downloadPath, version, refreshFilters)
		if err != nil {
			if downloadPathIsNew { // Remove the path created by getter
				os.RemoveAll(downloadPath)
			}
			return burrito.WrapErrorf(
				err, "Could not download filter from %s.\n"+
					"Does that filter exist?", f.Url)
		}
	}
	// Save the version of the filter we downloaded
	MeasureStart("Save version info")
	err = f.SaveVersionInfo(trimmedVersion, dotRegolithPath)
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

// downloadFilterGhRelease downloads the ZIP file of the filter from the
// GitHub release (or takes it from the cache) and extracts it to extractDir.
// The returned bool is true when the error only means that the release or its
// file isn't available (no release, gh not logged in, no matching file).
func (f *RemoteFilterDefinition) downloadFilterGhRelease(
	ghRepoName, versionTag, extractDir string, forceDownload bool,
) (bool, error) {
	// Get cache path
	userCache, err := os.UserCacheDir()
	if err != nil {
		return false, burrito.WrappedError(osUserCacheDirError)
	}
	fullGhReleaseCachePath := filepath.Join(
		userCache, appDataGhReleaseCachePath, ghRepoName)
	// Check if matching release file already exists in cache
	zipPath := ""
	matchingFileNames := ghReleaseFileNames(versionTag, runtime.GOOS, runtime.GOARCH)
	if !forceDownload {
		if stat, err := os.Stat(fullGhReleaseCachePath); err == nil && stat.IsDir() {
			for _, matchingFileName := range matchingFileNames {
				p := filepath.Join(fullGhReleaseCachePath, matchingFileName)
				if stat, err := os.Stat(p); err == nil && !stat.IsDir() {
					zipPath = p
					Logger.Infof(
						"Using GitHub release file from cache: %s",
						zipPath,
					)
					break
				}
			}
		}
	}
	// If file not found, try to download it
	if zipPath == "" {
		// Get available downloads
		downloadsList, err := ghListDownloads(ghRepoName, versionTag)
		if err != nil {
			return true, burrito.WrapError(
				err, "Failed to get the list of downloadable files from GitHub release.")
		}
		foundDownload := pickGhReleaseFile(downloadsList, matchingFileNames)
		if foundDownload == "" {
			return true, burrito.WrappedErrorf(
				"GitHub release of the filter doesn't provide a downloadable file for this system.\n"+
					"OS: %s\n"+
					"Architecture: %s\n"+
					"Checked: %s\n"+
					"Found: %s",
				runtime.GOOS, runtime.GOARCH,
				strings.Join(matchingFileNames, ", "),
				strings.Join(downloadsList, ", "),
			)
		}
		zipPath = filepath.Join(fullGhReleaseCachePath, foundDownload)
		err = ghDownloadRelease(ghRepoName, versionTag, foundDownload, zipPath)
		if err != nil {
			return false, burrito.PassError(err)
		}
	}
	// Extract the ZIP
	err = os.MkdirAll(extractDir, 0755)
	if err != nil {
		return false, burrito.WrapErrorf(err, osMkdirError, extractDir)
	}
	err = unzip(zipPath, extractDir)
	if err != nil {
		// The cached file may be broken (e.g. an interrupted download).
		// Remove it, so the next attempt downloads it again.
		os.Remove(zipPath)
		return false, burrito.WrapErrorf(
			err,
			"Failed unzip file:\nSource: %s\nTarget: %s",
			zipPath, extractDir,
		)
	}
	filterJsonPath := filepath.Join(extractDir, "filter.json")
	if _, err := os.Stat(filterJsonPath); err != nil {
		os.Remove(zipPath)
		return false, burrito.WrappedErrorf(
			"The GitHub release file doesn't have \"filter.json\" in its root "+
				"folder.\nFile: %s", zipPath)
	}
	return false, nil
}

// ghDownloadRelease downloads the file of a GitHub release to targetPath.
// The file is downloaded to a temporary folder first, so an interrupted
// download never leaves a broken file in targetPath.
func ghDownloadRelease(repository, tag, fileName, targetPath string) error {
	targetDir := filepath.Dir(targetPath)
	err := os.MkdirAll(targetDir, 0755)
	if err != nil {
		return burrito.WrapErrorf(err, osMkdirError, targetDir)
	}
	tmpDir, err := os.MkdirTemp(targetDir, ".download-")
	if err != nil {
		return burrito.WrapErrorf(err, osMkdirError, targetDir)
	}
	defer os.RemoveAll(tmpDir)
	cmd := exec.Command(
		"gh", "release", "download", tag, "--repo", repository,
		"--pattern", fileName, "--dir", tmpDir,
	)
	Logger.Debugf("Exec: %s", strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return burrito.WrapErrorf(
			err, "Failed to download the GitHub release file.\n"+
				"Command: %s\nOutput: %s",
			strings.Join(cmd.Args, " "), strings.TrimSpace(string(output)))
	}
	downloaded := filepath.Join(tmpDir, fileName)
	err = os.Rename(downloaded, targetPath)
	if err != nil {
		return burrito.WrapErrorf(err, osRenameError, downloaded, targetPath)
	}
	return nil
}

func (f *RemoteFilterDefinition) downloadFilterRepository(downloadPath, ref string, forceUpdate bool) error {
	config, err := getCombinedUserConfig()
	if err != nil {
		return burrito.WrapErrorf(err, getUserConfigError)
	}
	cooldown, err := time.ParseDuration(*config.ResolverCacheUpdateCooldown)
	url := fmt.Sprintf("https://%s", f.Url)
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
	err = copy.Copy(filepath.Join(cache, f.Id), downloadPath)
	if err != nil {
		return burrito.WrapErrorf(err, osCopyError, filepath.Join(cache, f.Id), downloadPath)
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
func (f *RemoteFilterDefinition) LoadFilterJson(dotRegolithPath string) (map[string]any, error) {
	downloadPath := f.GetDownloadPath(dotRegolithPath)
	filterJsonPath := path.Join(downloadPath, "filter.json")
	filterJson, err1 := os.ReadFile(filterJsonPath)
	var filterJsonMap map[string]any
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

// getGhRepoName returns the name of the GitHub repository of this remote
// filter in format <user>/<project> based on its URL propert or an empty
// string if it's not hosted on GitHub
func (f *RemoteFilterDefinition) getGhRepoName() string {
	parts := strings.SplitN(f.Url, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	if parts[0] != "github.com" {
		return ""
	}
	return parts[1]
}

// hasGit returns whether git is installed or not.

func hasGit() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func hasGhCli() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// ghListDownloads uses GitHub CLI (gh) command to list downloadable files
// on given repository for given tag.
func ghListDownloads(repository, tag string) ([]string, error) {
	cmd := exec.Command(
		"gh", "release", "view", tag, "--json", "assets",
		"--jq", ".assets[].name", "--repo", repository,
	)
	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr
	stdOut, err := cmd.Output()
	if err != nil {
		return []string{}, burrito.WrapErrorf(
			err, "Failed to run command.\nCommand: %s\nOutput: %s",
			strings.Join(cmd.Args, " "), strings.TrimSpace(stdErr.String()))
	}
	result := make([]string, 0)
	for _, line := range strings.Split(string(stdOut), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}

// ghReleaseFileNames returns the names of the ZIP files that can provide
// a filter for given tag, OS and architecture, from the most to the least
// specific.
func ghReleaseFileNames(tag, goos, goarch string) []string {
	return []string{
		tag + "-" + goos + "-" + goarch + ".zip",
		tag + "-" + goos + ".zip",
		tag + ".zip",
	}
}

// pickGhReleaseFile returns the first name from 'wanted' that is present in
// 'available' or an empty string if there is no match.
func pickGhReleaseFile(available, wanted []string) string {
	for _, name := range wanted {
		if slices.Contains(available, name) {
			return name
		}
	}
	return ""
}

// loadFilterConfig loads the remote filter configuration from the given path.
func loadFilterConfig(path string) (map[string]any, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, burrito.WrapErrorf(err, fileReadError, path)
	}
	var filterCollection map[string]any
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

// unzip unpacks a .zip file to target folder. This function:
// - Doesn't create targetDir
// - Doesn't remove the unziped files if it fails in the middle of the process
// - Doesn't write outside of targetDir
// - Fails if any of the paths of the extracted files already contains a file
func unzip(zipPath, targetDir string) error {
	// Check destination
	targetDirAbs, err := filepath.Abs(targetDir)
	if err != nil {
		return burrito.WrapErrorf(err, filepathAbsError, targetDir)
	}
	if stat, err := os.Stat(targetDirAbs); err != nil {
		if os.IsNotExist(err) {
			return burrito.WrappedErrorf(osStatErrorIsNotExist, targetDirAbs)
		}
		return burrito.WrapErrorf(err, osStatErrorAny, targetDirAbs)
	} else if !stat.IsDir() {
		return burrito.WrappedErrorf(isDirNotADirError, targetDirAbs)
	}

	// Unpack
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return burrito.WrapError(err, "Failed to open .zip reader.")
	}
	defer r.Close()
	for _, f := range r.File {
		// Extracted file must be in 'targetDirPath' folder
		if !filepath.IsLocal(f.Name) {
			return burrito.WrappedError("Zip contains non-local paths")
		}

		// Handle folders
		currTarget := filepath.Join(targetDirAbs, filepath.FromSlash(f.Name))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(currTarget, 0755); err != nil {
				return burrito.WrapErrorf(err, osMkdirError, currTarget)
			}
			continue
		}

		// Handle files
		currTargetParent := filepath.Dir(currTarget)
		if err := os.MkdirAll(currTargetParent, 0755); err != nil {
			return burrito.WrapErrorf(err, osMkdirError, currTargetParent)
		}
		err = func() error { // Use closure to for deferred file closing
			in, err := f.Open()
			if err != nil {
				return err
			}
			defer in.Close()

			out, err := os.OpenFile(
				currTarget,
				// create file | write only | fail if file exists
				os.O_CREATE|os.O_WRONLY|os.O_EXCL,
				// Keep the executable bits, but make sure the file is readable
				f.Mode().Perm()|0600,
			)
			if err != nil {
				return err
			}

			_, err = io.Copy(out, in)
			if closeErr := out.Close(); closeErr != nil {
				// Some copy errors can be reported on closing
				return burrito.WrapErrorf(closeErr, osCopyError, f.Name, currTarget)
			}
			if err != nil {
				return burrito.WrapErrorf(err, osCopyError, f.Name, currTarget)
			}
			return nil
		}()
		if err != nil {
			return burrito.PassError(err)
		}
	}

	return nil
}
