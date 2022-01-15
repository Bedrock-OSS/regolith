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
)

type filterDefinition struct {
	filter              func(filter Filter, settings map[string]interface{}, absoluteLocation string) error
	installDependencies func(filter Filter, path string) error
	check               func() error
	validateDefinition  func(filter Filter) error
}

var FilterTypes = map[string]filterDefinition{}

func RegisterFilters() {
	RegisterPythonFilter(FilterTypes)
	RegisterNodeJSFilter(FilterTypes)
	RegisterShellFilter(FilterTypes)
	RegisterJavaFilter(FilterTypes)
	RegisterNimFilter(FilterTypes)
}

func DefaultValidateDefinition(filter Filter) error {
	if filter.Script == "" {
		return errors.New("Missing 'script' field in filter definition")
	}
	return nil
}

// RunStandardFilter runs a filter from standard Bedrock-OSS library. The
// function doesn't test if the filter passed on input is standard.
func RunStandardFilter(filter Filter) error {
	Logger.Debugf("RunStandardFilter '%s'", filter.Filter)
	return RunRemoteFilter(filter.GetDownloadUrl(), filter)
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
func RunRemoteFilter(url string, parentFilter Filter) error {
	settings := parentFilter.Settings
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
		filter.VenvSlot = parentFilter.VenvSlot
		filter.Arguments = append(filter.Arguments, parentFilter.Arguments...)
		// Join settings from local config to remote definition
		for k, v := range settings {
			filter.Settings[k] = v
		}
		err := filter.Run(absolutePath)
		if err != nil {
			return err
		}
	}
	return nil
}

type Filter struct {
	Name      string                 `json:"name,omitempty"`
	Script    string                 `json:"script,omitempty"`
	Disabled  bool                   `json:"disabled,omitempty"`
	RunWith   string                 `json:"runWith,omitempty"`
	Command   string                 `json:"command,omitempty"`
	Arguments []string               `json:"arguments,omitempty"`
	Url       string                 `json:"url,omitempty"`
	Version   string                 `json:"version,omitempty"`
	Filter    string                 `json:"filter,omitempty"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
	VenvSlot  int                    `json:"venvSlot,omitempty"`
}

// Run determine whether the filter is remote, standard (from standard
// library) or local and executes it using the proper function.
//
// absoluteLocation is an absolute path to the root folder of the filter.
// In case of local filters it's a root path of the project.
func (filter *Filter) Run(absoluteLocation string) error {
	// Disabled filters are skipped
	if filter.Disabled {
		Logger.Infof("Filter '%s' is disabled, skipping.", filter.GetFriendlyName())
		return nil
	}

	Logger.Infof("Running filter %s", filter.GetFriendlyName())
	start := time.Now()

	// Standard Filter is only filter that doesn't require authentication.
	if filter.Filter != "" {

		// Special handling for our hello world filter,
		// which welcomes new users to Regolith
		if filter.Filter == "hello_world" {
			return RunHelloWorldFilter(filter)
		}

		// Otherwise drop down to standard handling
		err := RunStandardFilter(*filter)
		if err != nil {
			return err
		}
	} else {

		// All other filters require safe mode to be turned off
		if !IsUnlocked() {
			return errors.New("Safe mode is on, which protects you from potentially unsafe code. \nYou may turn it off using 'regolith unlock'.")
		}

		if filter.Url != "" {
			err := RunRemoteFilter(filter.Url, *filter)
			if err != nil {
				return err
			}
		} else {
			if f, ok := FilterTypes[filter.RunWith]; ok {
				if f.validateDefinition != nil {
					err := f.validateDefinition(*filter)
					if err != nil {
						return err
					}
				} else {
					err := DefaultValidateDefinition(*filter)
					if err != nil {
						return err
					}
				}
				err := f.filter(*filter, filter.Settings, absoluteLocation)
				if err != nil {
					return err
				}
			} else {
				Logger.Warnf("Filter type '%s' not supported", filter.RunWith)
			}
			Logger.Debugf("Executed in %s", time.Since(start))
		}
	}
	return nil
}

// IsRemote returns whether the filter is a remote filter or not.
// A remote filter requires installation
func (f *Filter) IsRemote() bool {
	return f.Script == ""
}

// IsInstalled eturns whether the filter is currently installed or not.
func (f *Filter) IsInstalled() bool {
	if _, err := os.Stat(f.GetDownloadPath()); err == nil {
		return true
	}
	return false
}

// InstalledVersion returns the currently installed version, or "" for a
// filter that isn't installed
func (f *Filter) InstalledVersion() string {
	if f.IsInstalled() {

		// TODO THIS IS WRONG
		// We need to store the current version into the actual download, somehow
		return f.Version
	}
	return ""
}

func (f *Filter) GetLatestVersion() string {
	// TODO This function needs to be created
	return ""
}

// IsFilterOutdated returns whether the downloaded filter it out of date or not.
func (f *Filter) IsFilterOutdated() bool {
	if f.IsInstalled() {

		// TODO THIS IS WRONG
		// We need to ping the remote repo to test for latest version
		if f.InstalledVersion() != f.Version {
			return true
		}
	}
	return false
}

// GetDownloadPath returns the path location where the filter can be found.
func (f *Filter) GetDownloadPath() string {
	return UrlToPath(f.Url)
}

// GetDownloadUrl creates a download URL, based on the filter definition.
func (f *Filter) GetDownloadUrl() string {
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

	return fmt.Sprintf("%s//%s%s", repoUrl, f.Filter, repoVersion)
}

// GetIdName returns the name that identifies the filter. This name is used to
// create and access the data folder for the filter. This property only makes
// sense for remote filters. Non-remote filters return empty string.
func (f *Filter) GetIdName() string {
	if f.Filter != "" {
		return f.Filter
	} else if f.Url != "" {
		splitUrl := strings.Split(f.Url, "/")
		return splitUrl[len(splitUrl)-1]
	}
	return ""
}

// GetFriendlyName returns the friendly name of the filter that can be used for
// logging.
func (f *Filter) GetFriendlyName() string {
	if f.Name != "" {
		return f.Name
	}
	return f.Filter
}

func (f *Filter) Uninstall() {
	err := os.RemoveAll(f.GetDownloadPath())
	if err != nil {
		Logger.Error(wrapError(fmt.Sprintf("Could not remove installed filter %s.", f.GetFriendlyName()), err))
	}
}

// DownloadDependencies installs all dependencies of the filter.
// The profile directory is the location in which the filter is installed
func (f *Filter) DownloadDependencies() error {
	installLocation := ""
	if f.IsRemote() {
		installLocation = f.GetDownloadPath()
	}
	// Install dependencies
	if f.RunWith == "" {
		return nil // No dependencies to install
	}
	Logger.Infof("Downloading dependencies for %s...", f.GetFriendlyName())

	if filterDefinition, ok := FilterTypes[f.RunWith]; ok {
		scriptPath, err := filepath.Abs(filepath.Join(installLocation, f.Script))
		if err != nil {
			return wrapError(fmt.Sprintf(
				"Unable to resolve path of %s script",
				f.GetFriendlyName()), err)
		}
		err = filterDefinition.installDependencies(*f, filepath.Dir(scriptPath))
		if err != nil {
			return wrapError(fmt.Sprintf(
				"Couldn't install filter dependencies %s",
				f.GetFriendlyName()), err)
		}
	} else {
		Logger.Warnf(
			"Filter type '%s' not supported", f.RunWith)
	}

	Logger.Infof("Dependencies for %s installed successfully", f.GetFriendlyName())
	return nil
}

// Download ownloads the filter into its own directory and returns the
// download path of the directory.
func (f *Filter) Download(isForced bool) (string, error) {
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
