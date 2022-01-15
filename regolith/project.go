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

// FilterCollection is a list of filters
type FilterCollection struct {
	Filters []Filter `json:"filters,omitempty"`
}

type Profile struct {
	FilterCollection
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

// FilterCollectionFromFilterJson returns a collection of filters from a
// "filter.json" file of a remote filter.
func FilterCollectionFromFilterJson(path string) (*FilterCollection, error) {
	file, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, wrapError(
			fmt.Sprintf("Couldn't read %q", path),
			err,
		)
	}

	var result *FilterCollection
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

// Install installs all of the filters in the profile including the nested ones
func (p *Profile) Install(isForced bool) error {
	return p.installFilters(isForced, p.Filters)
}

// installFilters provides a recursive function to install all filters in the
// profile. This function is not exposed outside of the regolith package. Use
// Install() instead.
func (p *Profile) installFilters(isForced bool, filters []Filter) error {
	for _, filter := range filters {
		err := p.installFilter(isForced, filter)
		if err != nil {
			return err
		}
	}
	return nil
}

// installFilter installs a single filter.
// - Downloads the filter if it is remote
// - Installs dependencies
// - Copies the filter's data to the data folder
// - Handles additional filters within the 'filters.json' file
func (p *Profile) installFilter(isForced bool, filter Filter) error {
	var err error

	// TODO - WTF is filterDirectory and downloadPath?! Why are they different?
	// Why is downloadPath created from URL?
	filterDirectory := ""
	downloadPath := UrlToPath(filter.GetDownloadUrl())
	if filter.IsRemote() {
		filterDirectory, err = filter.Download(isForced)
		downloadPath = filterDirectory
		if err != nil {
			return wrapError("could not download filter: ", err)
		}
		filterCollection, err := FilterCollectionFromFilterJson(
			filepath.Join(filterDirectory, "filter.json"))
		if err != nil {
			return fmt.Errorf(
				"could not load \"filter.json\" from path %q, while checking"+
					" for recursive dependencies", filterDirectory,
			)
		}
		p.installFilters(isForced, filterCollection.Filters)
	}
	// Move filters 'data' folder contents into 'data'
	filterName := filter.GetIdName()
	localDataPath := path.Join(p.DataPath, filterName)
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
		if p.DataPath != "" {
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
		repoUrl = STANDARD_LIBRARY_URL
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
func (f *Filter) DownloadDependencies(installLocation string) error {
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
						FilterCollection: FilterCollection{
							[]Filter{{Filter: "hello_world"}},
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
