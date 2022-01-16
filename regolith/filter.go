package regolith

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-getter"
	"github.com/otiai10/copy"
)

type FilterRunner interface {
	Run(absoluteLocation string) error
	InstallDependencies(parent *RemoteFilter) error
	Check() error

	CopyArguments(parent *RemoteFilter)
	GetFriendlyName() string
}

func RunnableFilterFromObject(obj map[string]interface{}) FilterRunner {
	runWith, _ := obj["runWith"].(string)
	switch runWith {
	case "java":
		return JavaFilterFromObject(obj)
	case "nim":
		return NimFilterFromObject(obj)
	case "nodejs":
		return NodeJSFilterFromObject(obj)
	case "python":
		return PythonFilterFromObject(obj)
	case "shell":
		return ShellFilterFromObject(obj)
	case "":
		return RemoteFilterFromObject(obj)
	}
	Logger.Fatalf("Unknown runWith '%s'", runWith)
	return nil
}

type RemoteFilter struct {
	Filter

	Id      string `json:"filter,omitempty"`
	Url     string `json:"url,omitempty"`
	Version string `json:"version,omitempty"`

	// RemoteFilters can propagate some of the properties unique to other types
	// of filers (like Python's venvSlot).
	VenvSlot int `json:"venvSlot,omitempty"`
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
			"Safe mode is on, which protects you from potentially unsafe " +
				"code.\nYou may turn it off using 'regolith unlock'.",
		)
	}
	Logger.Infof("Running filter %s", f.GetFriendlyName())
	start := time.Now()
	defer Logger.Debugf("Executed in %s", time.Since(start))
	return RunRemoteFilter(f.Url, f)
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
	_, end := path.Split(f.Url) // Return the last part of the URL
	return end
}

// copyFilterData copies the filter's data to the data folder.
func (f *RemoteFilter) copyFilterData(profile *Profile) {
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

func RemoteFilterFromObject(obj map[string]interface{}) *RemoteFilter {
	filter := &RemoteFilter{Filter: *FilterFromObject(obj)}
	id, _ := obj["filter"].(string) // filter property is optional
	filter.Id = id

	url, ok := obj["url"].(string)
	if !ok {
		filter.Url = StandardLibraryUrl
	} else {
		filter.Url = url
	}
	filter.Version, _ = obj["version"].(string) // Version is optional
	filter.VenvSlot, _ = obj["venvSlot"].(int)  // default venvSlot is 0
	return filter
}

func RunHelloWorldFilter(filter *Filter) error {
	Logger.Info(
		"Hello world!\n" +
			"===========================================================\n" +
			" Welcome to Regolith!\n" +
			"\n" +
			" This message is generated from the 'hello_world' filter.\n" +
			" You can delete this filter when you're ready, and replace it with" +
			" Something more useful!\n" +
			"===========================================================\n",
	)

	return nil
}

// IsRemoteFilterCached checks whether the filter of given URL is already saved
// in cache.
func IsRemoteFilterCached(url string) bool {
	_, err := os.Stat(UrlToPath(url))
	return err == nil
}

// RunRemoteFilter loads and runs the content of filter.json from in
// regolith cache. The url is the URL of the filter from which the filter
// was downloaded (used to specify its path in the cache). The parentFilter
// is a filter that caused the downloading. Some properties of
// parentFilter are propagated to its children.
func RunRemoteFilter(url string, parentFilter *RemoteFilter) error {
	Logger.Debugf("RunRemoteFilter '%s'", url)
	if !IsRemoteFilterCached(url) {
		return errors.New("filter is not downloaded! Please run 'regolith install'")
	}

	path := UrlToPath(url)
	absolutePath, _ := filepath.Abs(path)
	filterCollection, err := FilterCollectionFromFilterJson(path)
	if err != nil {
		return err
	}
	for _, filter := range filterCollection.Filters {
		// Overwrite the venvSlot with the parent value
		filter.CopyArguments(parentFilter)
		err := filter.Run(absolutePath)
		if err != nil {
			return err
		}
	}
	return nil
}

type Filter struct {
	Name string `json:"name,omitempty"`
	// Script    string                 `json:"script,omitempty"`
	Disabled bool `json:"disabled,omitempty"`
	// RunWith   string                 `json:"runWith,omitempty"`
	// Command   string                 `json:"command,omitempty"`
	Arguments []string `json:"arguments,omitempty"`
	// Url       string                 `json:"url,omitempty"`
	// Version   string                 `json:"version,omitempty"`
	// Filter    string                 `json:"filter,omitempty"`
	Settings map[string]interface{} `json:"settings,omitempty"`
	// VenvSlot  int                    `json:"venvSlot,omitempty"`
}

func FilterFromObject(obj map[string]interface{}) *Filter {
	filter := &Filter{}
	// Name
	name, _ := obj["name"].(string)
	filter.Name = name
	// Disabled
	disabled, _ := obj["disabled"].(bool)
	filter.Disabled = disabled
	// Arguments
	arguments, _ := obj["arguments"].([]string)
	filter.Arguments = arguments
	// Settings
	settings, _ := obj["settings"].(map[string]interface{})
	filter.Settings = settings
	return filter
}

// IsInstalled eturns whether the filter is currently installed or not.
func (f *RemoteFilter) IsInstalled() bool {
	if _, err := os.Stat(f.GetDownloadPath()); err == nil {
		return true
	}
	return false
}

func (f *Filter) GetLatestVersion() string {
	// TODO This function needs to be created
	return ""
}

// GetDownloadPath returns the path location where the filter can be found.
func (f *RemoteFilter) GetDownloadPath() string {
	return UrlToPath(f.Url)
}

// GetDownloadUrl creates a download URL, based on the filter definition.
func (f *RemoteFilter) GetDownloadUrl() string {
	repoUrl := ""
	if f.Url == "" {
		repoUrl = StandardLibraryUrl
	} else {
		repoUrl = f.Url
	}

	repoVersion := ""
	if f.Version != "" {
		repoVersion = "?ref=" + f.Version
	}

	return fmt.Sprintf("%s//%s%s", repoUrl, f.Id, repoVersion)
}

// GetIdName returns the name that identifies the filter. This name is used to
// create and access the data folder for the filter. This property only makes
// sense for remote filters. Non-remote filters return empty string.
func (f *RemoteFilter) GetIdName() string {
	if f.Id != "" {
		return f.Id
	} else if f.Url != "" {
		splitUrl := strings.Split(f.Url, "/")
		return splitUrl[len(splitUrl)-1]
	}
	return ""
}

func (f *RemoteFilter) Uninstall() {
	err := os.RemoveAll(f.GetDownloadPath())
	if err != nil {
		Logger.Error(wrapError(fmt.Sprintf("Could not remove installed filter %s.", f.GetFriendlyName()), err))
	}
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
