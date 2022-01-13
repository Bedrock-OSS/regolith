package regolith

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-getter"
	"github.com/otiai10/copy"
)

const ManifestName = "config.json"
const GitIgnore = `/build
/.regolith`

// The full configuration file of Regolith, as saved in config.json
type Config struct {
	Name            string `json:"name,omitempty"`
	Author          string `json:"author,omitempty"`
	Packs           `json:"packs,omitempty"`
	RegolithProject `json:"regolith,omitempty"`
}

// The export information for a profile, which denotes where compiled files will go
type ExportTarget struct {
	Target    string `json:"target,omitempty"` // The mode of exporting. "develop" or "exact"
	RpPath    string `json:"rpPath,omitempty"` // Relative or absolute path to resource pack for "exact" export target
	BpPath    string `json:"bpPath,omitempty"` // Relative or absolute path to resource pack for "exact" export target
	WorldName string `json:"worldName,omitempty"`
	WorldPath string `json:"worldPath,omitempty"`
	ReadOnly  bool   `json:"readOnly"` // Whether the exported files should be read-only
}

type Packs struct {
	BehaviorFolder string `json:"behaviorPack,omitempty"`
	ResourceFolder string `json:"resourcePack,omitempty"`
}

// Regolith namespace within the Minecraft Project Schema
type RegolithProject struct {
	Profiles map[string]Profile `json:"profiles,omitempty"`
}

// List of filter definitions
type Profile struct {
	Filters      []Filter     `json:"filters,omitempty"`
	ExportTarget ExportTarget `json:"export,omitempty"`
	DataPath     string       `json:"dataPath,omitempty"`
}

func LoadConfig() *Config {
	file, err := ioutil.ReadFile(ManifestName)
	if err != nil {
		Logger.Fatal("Couldn't find %s! Consider running 'regolith init'.", ManifestName, err)
	}

	var result *Config
	err = json.Unmarshal(file, &result)
	if err != nil {
		Logger.Fatal("Couldn't load %s! Does the file contain correct json?", ManifestName, err)
	}

	// If settings is nil replace it with empty map.
	for _, profile := range result.Profiles {
		for fk := range profile.Filters {
			if profile.Filters[fk].Settings == nil {
				profile.Filters[fk].Settings = make(map[string]interface{})
			}
		}
	}
	return result
}

// LoadFiltersFromPath returns a Profile with list of filters loaded from
// filters.json from input file path. The path should point at a directory
// with filters.json file in it, not at the file itself.
func LoadFiltersFromPath(path string) (*Profile, error) {
	path = path + "/filter.json"
	file, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, wrapError(fmt.Sprintf("Couldn't find %s! Consider running 'regolith install'", path), err)
	}

	var result *Profile
	err = json.Unmarshal(file, &result)
	if err != nil {
		return nil, wrapError(fmt.Sprintf("Couldn't load %s: ", path), err)
	}
	// Replace nil filter settings with empty map
	for fk := range result.Filters {
		if result.Filters[fk].Settings == nil {
			result.Filters[fk].Settings = make(map[string]interface{})
		}
	}
	return result, nil
}

// LoadFilterJsonProfile loads a profile from path to filter.json file of
// a remote filter and propagates the properties of the parent filter (the
// filter in config.json or other remote filter that caused creation of this
// profile).and the parent profile to the returned profile.
func LoadFilterJsonProfile(
	filterJsonPath string, parentFilter Filter, parentProfile Profile,
) (*Profile, error) {
	// Open file
	file, err := ioutil.ReadFile(filterJsonPath)
	if err != nil {
		return nil, wrapError(fmt.Sprintf(
			"Couldn't find %s", filterJsonPath), err)
	}
	// Load data into Profile struct
	var remoteProfile Profile
	err = json.Unmarshal(file, &remoteProfile)
	if err != nil {
		return nil, wrapError(fmt.Sprintf(
			"Couldn't load %s: ", filterJsonPath), err)
	}
	// Propagate venvSlot property
	for subfilter := range remoteProfile.Filters {
		remoteProfile.Filters[subfilter].VenvSlot = parentFilter.VenvSlot
	}
	remoteProfile.DataPath = parentProfile.DataPath
	remoteProfile.ExportTarget = parentProfile.ExportTarget
	return &remoteProfile, nil
}

// Installs a profile, by looping over and installing every filter
func (profile *Profile) Install(isForced bool) error {
	for _, filter := range profile.Filters {
		err := filter.RecursiveInstall(isForced)
		if err != nil {
			return err
		}
	}
	return nil
}

// This function will loop over every filter in the profile, and recursively download it
// and install it's dependencies, including child filters as well as libraries.
func (profile *Profile) Install_OLD(isForced bool, profilePath string) error {
	for filter := range profile.Filters {
		filter := &profile.Filters[filter] // Using pointer is faster than creating copies in the loop and gives more options

		// If filter is remote, download it
		var err error
		downloadPath := UrlToPath(filter.GetDownloadUrl())
		if filter.IsRemote() {
			downloadPath, err = filter.Download(isForced)
			if err != nil {
				Logger.Fatal(wrapError("Could not download filter: ", err))
			}
		}

		// Install dependencies
		err = filter.DownloadDependencies(downloadPath)

		// Move filters 'data' folder contents into 'data'
		filterName := filter.GetIdName()
		localDataPath := path.Join(profile.DataPath, filterName)
		remoteDataPath := path.Join(downloadPath, "data")

		// If the filterDataPath already exists, we must not overwrite
		// Additionally, if the remote data path doesn't exist, we don't need
		// to do anything
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

		// Create profile from filter.json file
		filterPath, err := filepath.Abs(path.Join(downloadPath, "filter.json"))
		if err != nil {
			return wrapError("Could not find filter.json", err)
		}

		remoteProfile, err := LoadFilterJsonProfile(filterPath, *filter, *profile)
		if err != nil {
			return wrapError("Could not read filter.json. Is the json valid?", err)
		}

		// Install dependencies of remote filters. Recursion ends when there
		// is no more nested remote dependencies.
		err = remoteProfile.Install_OLD(isForced, filterPath)
		if err != nil {
			return wrapError("Could not install recursive profile", err)
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

// Returns whether the filter is a remote filter or not.
// A remote filter requires installation
func (filter *Filter) IsRemote() bool {
	return filter.Script == ""
}

// Returns whether the filter is currently installed or not.
func (filter *Filter) IsInstalled() bool {
	if _, err := os.Stat(filter.GetDownloadPath()); err == nil {
		return true
	}
	return false
}

// Returns the currently installed version, or "" for a filter that isn't installed
func (filter *Filter) InstalledVersion() string {
	if filter.IsInstalled() {

		// TODO THIS IS WRONG
		// We need to store the current version into the actual download, somehow
		return filter.Version
	}
	return ""
}

func (filter *Filter) GetLatestVersion() string {
	// TODO This function needs to be created
	return ""
}

// Returns whether the downloaded filter it out of date or not.
func (filter *Filter) IsFilterOutdated() bool {
	if filter.IsInstalled() {

		// TODO THIS IS WRONG
		// We need to ping the remote repo to test for latest version
		if filter.InstalledVersion() != filter.Version {
			return true
		}
	}
	return false
}

// Recursively installs a filter:
// - Downloads the filter if it is remote
// - Installs dependencies
// - Copies the filter's data to the data folder
// - Handles additional filters within the 'filters.json' file
func (filter *Filter) RecursiveInstall(isForced bool) error {
	var err error
	filterDirectory := ""

	if filter.IsRemote() {
		filterDirectory, err = filter.Download(isForced)
		if err != nil {
			return wrapError("Could not download filter: ", err)
		}
		// Create fake profile from filter.json file to check for nested
		// dependencies
		profile, err := LoadFiltersFromPath(filterDirectory)
		if err != nil {
			return fmt.Errorf(
				"could not load \"filter.json\" from path %q, while checking"+
					" for recursive dependencies", filterDirectory,
			)
		}
		profile.Install(isForced)
	}

	// Install dependencies
	if filter.RunWith == "" {
		return nil // No dependencies to install
	}
	err = filter.DownloadDependencies(filterDirectory)
	if err != nil {
		return wrapError("Could not download dependencies: ", err)
	}

	return nil
}

// Returns the path location where the filter can be found.
func (filter *Filter) GetDownloadPath() string {
	return UrlToPath(filter.Url)
}

// Creates a download URL, based on the filter definition.
func (filter *Filter) GetDownloadUrl() string {
	repoUrl := ""
	if filter.Url == "" {
		repoUrl = STANDARD_LIBRARY_URL
	} else {
		repoUrl = filter.Url
	}

	repoVersion := ""
	if filter.Version != "" {
		repoVersion = "?ref=" + filter.Version
	}

	return fmt.Sprintf("%s//%s%s", repoUrl, filter.Filter, repoVersion)
}

// GetIdName returns the name that identifies the filter. This name is used to
// create and access the data folder for the filter. This property only makes
// sense for remote filters. Non-remote filters return empty string.
func (filter *Filter) GetIdName() string {
	if filter.Filter != "" {
		return filter.Filter
	} else if filter.Url != "" {
		splitUrl := strings.Split(filter.Url, "/")
		return splitUrl[len(splitUrl)-1]
	}
	return ""
}

// GetFriendlyName returns the friendly name of the filter that can be used for
// logging.
func (filter *Filter) GetFriendlyName() string {
	if filter.Name != "" {
		return filter.Name
	}
	return filter.Filter
}

func (filter *Filter) Uninstall() {
	err := os.RemoveAll(filter.GetDownloadPath())
	if err != nil {
		Logger.Error(wrapError(fmt.Sprintf("Could not remove installed filter %s.", filter.GetFriendlyName()), err))
	}
}

// Installs all dependencies of the filter.
// The profile directory is the location in which the filter is installed
func (filter *Filter) DownloadDependencies(installLocation string) error {
	Logger.Infof("Downloading dependencies for %s...", filter.GetFriendlyName())

	if filterDefinition, ok := FilterTypes[filter.RunWith]; ok {
		scriptPath, err := filepath.Abs(filepath.Join(installLocation, filter.Script))
		if err != nil {
			return wrapError(fmt.Sprintf(
				"Unable to resolve path of %s script",
				filter.GetFriendlyName()), err)
		}
		err = filterDefinition.installDependencies(*filter, filepath.Dir(scriptPath))
		if err != nil {
			return wrapError(fmt.Sprintf(
				"Couldn't install filter dependencies %s",
				filter.GetFriendlyName()), err)
		}
	} else {
		Logger.Warnf(
			"Filter type '%s' not supported", filter.RunWith)
	}

	Logger.Infof("Dependencies for %s installed successfully", filter.GetFriendlyName())
	return nil
}

// Downloads the filter into its own directory and returns the download path of the directory.
func (filter *Filter) Download(isForced bool) (string, error) {
	url := filter.GetDownloadUrl()
	downloadPath := filter.GetDownloadPath()

	if filter.IsInstalled() {
		if !isForced {
			Logger.Warnf("Filter %s already installed, skipping. Run "+
				"with '-f' to force.", filter.GetFriendlyName())
			return "", nil
		} else {
			// TODO should we print version information here?
			// like "version 1.4.2 uninstalled, version 1.4.3 installed"
			Logger.Warnf("Filter %s already installed, but force mode is enabled.\n"+
				"Filter will be installed, erasing prior contents.", filter.GetFriendlyName())
			filter.Uninstall()
		}
	}

	Logger.Infof("Downloading filter %s...", filter.GetFriendlyName())

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

	Logger.Infof("Filter %s downloaded successfully.", filter.GetFriendlyName())
	return downloadPath, nil
}

func IsProjectInitialized() bool {
	info, err := os.Stat(ManifestName)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func InitializeRegolithProject(isForced bool) error {
	// Do not attempt to initialize if project is already initialized (can be forced)
	if !isForced && IsProjectInitialized() {
		Logger.Errorf("Could not initialize Regolith project. File %s already exists.", ManifestName)
		return nil
	} else {
		Logger.Info("Initializing Regolith project...")

		if isForced {
			Logger.Warn("Initialization forced. Data may be lost.")
		}

		// Delete old configuration if it exists
		if err := os.Remove(ManifestName); !os.IsNotExist(err) {
			if err != nil {
				return err
			}
		}

		// Create new configuration
		jsonData := Config{
			Name:   "Project Name",
			Author: "Your name",
			Packs: Packs{
				BehaviorFolder: "./packs/BP",
				ResourceFolder: "./packs/RP",
			},
			RegolithProject: RegolithProject{
				Profiles: map[string]Profile{
					"dev": {
						DataPath: "./packs/data",
						Filters: []Filter{
							{
								Filter: "hello_world",
							},
						},
						ExportTarget: ExportTarget{
							Target:   "development",
							ReadOnly: false,
						},
					},
				},
			},
		}
		jsonBytes, _ := json.MarshalIndent(jsonData, "", "  ")
		err := ioutil.WriteFile(ManifestName, jsonBytes, 0666)
		if err != nil {
			return wrapError("Failed to write project file contents", err)
		}

		// Create default gitignore file
		err = ioutil.WriteFile(".gitignore", []byte(GitIgnore), 0666)
		if err != nil {
			return wrapError("Failed to write .gitignore file contents", err)
		}

		foldersToCreate := []string{
			"packs",
			"packs/data",
			"packs/BP",
			"packs/RP",
			".regolith",
			".regolith/cache",
			".regolith/venvs",
		}

		for _, folder := range foldersToCreate {
			err = os.Mkdir(folder, 0666)
			if err != nil {
				Logger.Error("Could not create folder: %s", folder, err)
			}
		}

		Logger.Info("Regolith project initialized.")
		return nil
	}
}

// CleanCache removes all contents of .regolith folder.
func CleanCache() error {
	Logger.Infof("Cleaning cache...")
	err := os.RemoveAll(".regolith")
	if err != nil {
		return wrapError("Failed to remove .regolith folder", err)
	}
	err = os.Mkdir(".regolith", 0666)
	if err != nil {
		return wrapError("Failed to recreate .regolith folder", err)
	}
	Logger.Infof("Cache cleaned.")
	return nil
}
