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
	"golang.org/x/mod/semver"

	"github.com/hashicorp/go-getter"
	"github.com/otiai10/copy"
)

type RemoteFilterDefinition struct {
	FilterDefinition
	Url     string `json:"url,omitempty"`
	Version string `json:"version,omitempty"`
	// RemoteFilters can propagate some of the properties unique to other types
	// of filers (like Python's venvSlot).
	VenvSlot          int                 `json:"venvSlot,omitempty"`
	RepoManifest      *RepositoryManifest `json:"-"`
	PreResolveVersion string              `json:"-"` // This is slightly janky but it basically represets the version information before its resolved. So it would be `latest` instead of some hash/version
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
	result.VenvSlot, _ = obj["venvSlot"].(int) // default venvSlot is 0

	return result, nil
}

// UrlBased checks if the filter is URL based. A filter is assumed to be URL
// based if it can't be found in the repository manifest, or is in the manifest
// but uses the "versions" property with list of URLs to its releases.
func (f *RemoteFilterDefinition) UrlBased() bool {
	if f.RepoManifest == nil {
		return false
	}
	val, _ := f.RepoManifest.IsUrlBased(f.Id)
	return val
}

// CreateResolver creates a PathResolver for the filter.
func (f *RemoteFilterDefinition) CreateResolver() (PathResolver, error) {
	if f.RepoManifest == nil {
		return SimpleResolver{}, nil
	}

	path, err := f.RepoManifest.FindPath(f.Id)
	if err != nil {
		return nil, burrito.WrapErrorf(
			err, "Failed to find the path to the filter on the repository.")
	}
	return &ComplexResolver{path: path}, nil
}

// SimpleResolver always returns the name of the filter as the path to its files.
type SimpleResolver struct{}

// ComplexResolver returns the path to the filter's files based on the information
// specified in the repository manifest (always returns the same value, the
// path is cached during its creation).
type ComplexResolver struct {
	path *string
}

func (n SimpleResolver) FetchPathsForFilter(filter string) string {
	return filter
}

func (f *ComplexResolver) FetchPathsForFilter(filter string) string {
	return *f.path
}

func (f *RemoteFilter) run(context RunContext) error {
	Logger.Debugf("RunRemoteFilter \"%s\"", f.Definition.Url)
	if !f.IsCached(context.DotRegolithPath) {
		return burrito.WrappedErrorf(
			"Filter is not downloaded. "+
				"You can download filter files using command:\n"+
				"regolith install %s", f.Id)
	}

	version, err := f.GetCachedVersion(context.DotRegolithPath)
	if err != nil {
		return burrito.WrapErrorf(
			err, "Failed check the version of the filter in cache."+
				"\nFilter: %s\n"+
				"You can try to force reinstallation fo the filter using command:"+
				"regolith install --force %s", f.Id, f.Id)
	}
	if f.Definition.Version != "HEAD" && f.Definition.Version != "latest" && f.Definition.Version != *version {
		return burrito.WrappedErrorf(
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
		return burrito.WrapErrorf(err, remoteFilterSubfilterCollectionError)
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
			return burrito.WrapErrorf(err, "Failed to check if filter is disabled")
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
			return burrito.WrapErrorf(
				err, filterRunnerRunError,
				NiceSubfilterName(f.Id, i))
		}
	}
	return nil
}

func (f *RemoteFilter) Run(context RunContext) (bool, error) {
	if err := f.run(context); err != nil {
		return false, burrito.PassError(err)
	}
	return context.IsInterrupted(), nil
}

func (f *RemoteFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) (FilterRunner, error) {
	basicFilter, err := filterFromObject(runConfiguration)
	if err != nil {
		return nil, burrito.WrapError(err, filterFromObjectError)
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
	file, err := os.ReadFile(path)

	if err != nil {
		return burrito.WrapErrorf(err, fileReadError, path)
	}

	var filterCollection map[string]interface{}
	err = json.Unmarshal(file, &filterCollection)
	if err != nil {
		return burrito.WrapErrorf(err, jsonUnmarshalError, path)
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
		map[string]interface{}{"filter": f.Id})
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
			return nil, burrito.WrapError(
				err, "Failed to get download link for the filter.")
		}
		version = trimFilterPrefix(version, name)
	}

	manifest, err := ManifestForRepo(url)
	if err != nil {
		// Not reporting the details, because the handling of the
		// FilterDefinitionFromTheInternet should do that.
		return nil, burrito.WrapError(
			err, "Failed to get manifest for the filter")
	}

	return &RemoteFilterDefinition{
		FilterDefinition: FilterDefinition{Id: name},
		Version:          version,
		Url:              url,
		RepoManifest:     manifest,
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

	var err error
	if f.RepoManifest == nil {
		err = f.downloadRemoteFromGit(dotRegolithPath, refreshFilters)
		if err != nil {
			return burrito.PassError(err)
		}
		return nil
	} else if urlBased, err := f.RepoManifest.IsUrlBased(f.Id); err == nil && !urlBased {
		err = f.downloadRemoteFromGit(dotRegolithPath, refreshFilters)
		if err != nil {
			return burrito.PassError(err)
		}
		return nil
	} else if err != nil {
		// We don't need to report the f.Id because this is handled outside
		// of Download function.
		return burrito.WrapError(
			err, "Failed to determine if the filter is URL-based.")
	} else {
		err := f.downloadFromRelease(dotRegolithPath, f.Version)

		if err != nil {
			return burrito.PassError(err)
		}
		return nil
	}
}

func (f *RemoteFilterDefinition) downloadFromRelease(dotRegolithPath string, expectedVersion string) error {
	var url *string
	var version *string
	var err error

	url, version, err = f.RepoManifest.ResolveUrlForFilter(f.Id, expectedVersion)

	if err != nil {
		return burrito.PassError(err)
	}

	if url == nil {
		return burrito.WrappedErrorf(
			"Unable to find a matching version of the filter for the current host.\n"+
				"Version: %q", f.Version)
	}

	downloadPath := f.GetDownloadPath(dotRegolithPath)
	err = os.MkdirAll(downloadPath, 0775)
	if err != nil {
		return burrito.WrapErrorf(err, osMkdirError, downloadPath)
	}

	err = getter.GetAny(f.GetDownloadPath(dotRegolithPath), *url)
	if err != nil {
		return burrito.WrapErrorf(
			err, "Failed to download the filter from the reoslved URL:\n"+
				"URL: %s", *url)
	}

	// We can deref version without a nil check because any time URL is populated so is version
	f.Version = *version
	f.SaveVersionInfo(f.Version, dotRegolithPath)
	return nil
}

func manfiestForLocation(location string) (*RepositoryManifest, error) {
	bytes, err := os.ReadFile(filepath.Join(location, "regolith_filter_manifest.json"))

	// The called is expected to add more information since this is a very generic function
	// Meaning we wont add any kind of context to errors

	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, burrito.PassError(err)
	}

	object := make(map[string]interface{})
	err = json.Unmarshal(bytes, &object)

	if err != nil {
		return nil, burrito.PassError(err)
	}

	manifest, err := RepositoryManifestFromObject(object)

	if err != nil {
		return nil, burrito.PassError(err)
	}

	return manifest, nil
}

func (f *RemoteFilterDefinition) installFromResolver(downloadLocation, baseLocation string, resolver PathResolver) error {

	err := copy.Copy(filepath.Join(baseLocation, resolver.FetchPathsForFilter(f.Id)), downloadLocation)

	if err != nil {
		return burrito.WrapErrorf(err, osCopyError, filepath.Join(baseLocation, resolver.FetchPathsForFilter(f.Id)), downloadLocation)
	}

	return nil
}

func (f *RemoteFilterDefinition) installFilterFully(dotRegolithPath, downloadPath, rawLocation, url, ref string) (*string, error) {
	var err error

	err = checkoutRepository(url, ref, rawLocation)

	var manifest *RepositoryManifest

	if f.RepoManifest == nil {
		if err == nil {
			manifest, err = manfiestForLocation(rawLocation) // There is a chance that it may just not have a manifest

			if err != nil && !os.IsNotExist(err) {
				return nil, burrito.WrapErrorf(err, "Failed to parse the manifest for: %s", url)
			}

		} else {
			// This means that the ref didn't exist meaning it may be a filter version so we need to check for a manifest
			prefix := trimFilterPrefix(f.Version, f.Id)

			manifest, err = manfiestForLocation(rawLocation)

			if err != nil {
				return nil, burrito.WrapErrorf(err, "Failed to checkout ref.\nURL: %s\nRef: %s", url, ref)
			}

			// Means we are looking at a requested version from the manifest
			if !semver.IsValid("v" + prefix) {
				return nil, burrito.WrappedErrorf("Invalid version %s, It must be a semver!", ref)
			}
		}
	} else {
		manifest = f.RepoManifest
	}

	if manifest != nil {
		if !manifest.Exists(f.Id) {
			return nil, burrito.WrappedErrorf("Filter %s is not located in the repository manifest for %s", f.Id, url)
		}

		f.RepoManifest = manifest

		// We can drop the error here since we are checking it just above
		if urlBased, _ := manifest.IsUrlBased(f.Id); urlBased {

			// The reason we rely on PreSearchVersion here is because if someone uses HEAD or latest as the version it will get resolved into a hash
			err := f.downloadFromRelease(dotRegolithPath, f.PreResolveVersion)

			if err != nil {
				return nil, burrito.PassError(err)
			}

			return &f.PreResolveVersion, nil

		}
	}

	resolver, err := f.CreateResolver()

	if err != nil {
		return nil, burrito.WrapErrorf(err, "Failed to resolve path for: %s, URL: %s, version: %s", f.Id, f.Url, f.Version)
	}

	err = f.installFromResolver(downloadPath, rawLocation, resolver)

	if err != nil {
		return nil, burrito.WrapErrorf(err, "Failed to install %s from a resolver!", f.Id)
	}

	return nil, nil
}

func (f *RemoteFilterDefinition) downloadRemoteFromGit(dotRegolithPath string, refreshFilters bool) error {
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

	cacheLocation, err := downloadFilterRepository(url, refreshFilters)
	if err != nil {
		if downloadPathIsNew { // Remove the path created by getter
			os.Remove(downloadPath)
		}
		return burrito.WrapErrorf(
			err, "Could not download filter from %s.\n"+
				"Does that filter exist?", f.Url)
	}

	versionOverride, err := f.installFilterFully(dotRegolithPath, downloadPath, *cacheLocation, url, repoVersion)

	if err != nil {
		return burrito.PassError(err)
	}

	// Save the version of the filter we downloaded
	MeasureStart("Save version info")
	if versionOverride == nil {
		err = f.SaveVersionInfo(trimFilterPrefix(repoVersion, f.Id), dotRegolithPath)
	}

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

// PathResolver is an interface that returns the path to the filter's file
// relative to the root of the repository based on the filter's name.
type PathResolver interface {
	// FetchPathsForFilter returns the path to the filter's files.
	FetchPathsForFilter(filter string) string
}

func checkoutRepository(url, ref, cache string) error {
	// Checkout the specified ref
	MeasureStart("Checkout ref %s", ref)
	output, err := RunGitProcess([]string{"checkout", ref}, cache)
	if err != nil {
		Logger.Error(strings.Join(output, "\n"))
		return burrito.WrapErrorf(err, "Failed to checkout ref.\nURL: %s\nRef: %s", url, ref)
	}
	MeasureEnd()
	return nil
}

func downloadFilterRepository(url string, forceUpdate bool) (*string, error) {
	config, err := getCombinedUserConfig()
	if err != nil {
		return nil, burrito.WrapErrorf(err, getUserConfigError)
	}
	cooldown, err := time.ParseDuration(*config.ResolverCacheUpdateCooldown)
	if err != nil {
		return nil, burrito.WrapErrorf(err, resolverParseDurationError, *config.ResolverCacheUpdateCooldown)
	}

	cache, err := getFilterCache(url)
	if err != nil {
		return nil, burrito.WrapErrorf(err, "Could not get cache path for %s", url)
	}
	// Check if exists in cache
clone:
	if _, err := os.Stat(cache); err != nil && os.IsNotExist(err) {
		err := os.MkdirAll(cache, 0755)
		if err != nil {
			return nil, burrito.WrapErrorf(err, osMkdirError, cache)
		}
		// Clone the repository
		MeasureStart("Clone repository %s", url)
		output, err := RunGitProcess([]string{"clone", url, "."}, cache)
		if err != nil {
			Logger.Error(strings.Join(output, "\n"))
			return nil, burrito.WrapErrorf(err, "Failed to clone repository.\nURL: %s", url)
		}
		forceUpdate = false
	} else if err != nil {
		return nil, burrito.WrapErrorf(err, osStatErrorAny, cache)
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
				return nil, burrito.WrapErrorf(err, osRemoveError, cache)
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
				return nil, burrito.WrapErrorf(err, osRemoveError, cache)
			}
			goto clone
		}
		err = os.Chtimes(cache, time.Now(), time.Now())
		if err != nil {
			Logger.Debugf(osChtimesError, cache)
		}
	}

	return &cache, nil
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

	var version string

	if f.UrlBased() {
		version = f.Version

	} else {

		if f.PreResolveVersion == "" {
			version = "latest"
			f.PreResolveVersion = "latest"
		} else if f.PreResolveVersion != "latest" && f.PreResolveVersion != "HEAD" && !semver.IsValid("v"+f.PreResolveVersion) {
			f.PreResolveVersion = ""
		} else if f.PreResolveVersion == f.Version && !VersionIsLatest(f.Version) {
			f.Version = "HEAD"
		} else if f.Version == "latest" && f.PreResolveVersion == "latest" {
			f.Version = "HEAD"
		}

		if version != "latest" {
			version, err = GetRemoteFilterDownloadRef(f.Url, f.Id, f.Version)

			if err != nil {
				return burrito.WrapErrorf(
					err, getRemoteFilterDownloadRefError, f.Url, f.Id, f.Version)
			}
		}

	}

	MeasureEnd()
	version = trimFilterPrefix(version, f.Id)
	if installedVersion != version || force {
		Logger.Infof(
			"Updating filter %q to new version: %q->%q.",
			f.Id, installedVersion, version)
		err = f.Download(true, dotRegolithPath, refreshFilters)
		if err != nil {
			return burrito.WrapErrorf(err, remoteFilterDownloadError, f.Id)
		}
		// Copy the data of the remote filter to the data path
		f.CopyFilterData(dataPath, dotRegolithPath)
		err = f.InstallDependencies(f, dotRegolithPath)
		if err != nil {
			return burrito.WrapErrorf(
				err, "Failed to install filter dependencies.\nFilter: %s", f.Id)
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

// extraFilterJsonErrorInfo is used to wrap errors related to parsing the
// filter.json file. It's common for other functions to handle loading and
// parsing of this file, so using this is necessary to provide both the
// information about the file path and reuse the errors from errors.go
//
// TODO - this is an ugly solution, perhaps we should have a separate
// function for loading the filter.json file. Currently it's always build into
// other functions.
func extraFilterJsonErrorInfo(filterJsonFilePath string, err error) error {
	return burrito.WrapErrorf(
		err, "Failed to load the filter configuration.\n"+
			"Filter configuration file: %s", filterJsonFilePath)
}
