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

// TODO implement the rest of the standard config spec
type Config struct {
	Name            string `json:"name,omitempty"`
	Author          string `json:"author,omitempty"`
	Packs           `json:"packs,omitempty"`
	RegolithProject `json:"regolith,omitempty"`
}

func LoadConfig() (*Config, error) {
	file, err := ioutil.ReadFile(ManifestName)
	if err != nil {
		return nil, wrapError(fmt.Sprintf("Couldn't find %s! Consider running 'regolith init'", ManifestName), err)
	}
	var result *Config
	err = json.Unmarshal(file, &result)
	if err != nil {
		return nil, wrapError(fmt.Sprintf("Couldn't load %s: ", ManifestName), err)
	}
	// If settings is nil replace it with empty map
	for _, profile := range result.Profiles {
		for fk := range profile.Filters {
			if profile.Filters[fk].Settings == nil {
				profile.Filters[fk].Settings = make(map[string]interface{})
			}
		}
	}
	return result, nil
}

type Packs struct {
	BehaviorFolder string `json:"behaviorPack,omitempty"`
	ResourceFolder string `json:"resourcePack,omitempty"`
}

type RegolithProject struct {
	Profiles map[string]Profile `json:"profiles,omitempty"`
}

type Profile struct {
	Filters      []Filter     `json:"filters,omitempty"`
	ExportTarget ExportTarget `json:"export,omitempty"`
	DataPath     string       `json:"dataPath,omitempty"`
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

// Install installs all of the dependencies of the profile
func (p *Profile) Install(isForced bool, profilePath string) error {
	for filter := range p.Filters {
		filter := &p.Filters[filter] // Using pointer is faster than creating copies in the loop and gives more options

		downloadPath, err := filter.Download(isForced, profilePath)
		// TODO - we could use type switch to handle different kinds of errors
		// here. Download can fail on downloading or on cleaning the download
		// path. It can also fail when isForced is false and the path already
		// exists.
		if err != nil {
			return wrapError("Could not download filter", err) // TODO - I don't think this should break the entire install
		} else if downloadPath == "" { // filter.RunWith != "" && filter.Script != ""
			continue
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

		// Create profile from filter.json file
		remoteProfile, err := LoadFilterJsonProfile(
			filepath.Join(downloadPath, "filter.json"), *filter, *p)
		if err != nil {
			return err // TODO - I don't think this should break the entire install. Just remove the files and continue.
		}

		// Install dependencies of remote filters. Recursion ends when there
		// is no more nested remote dependencies.
		err = remoteProfile.Install(isForced, downloadPath)
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
	Filter    string                 `json:"filter,omitempty"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
	VenvSlot  int                    `json:"venvSlot,omitempty"`
}

// GetFilterPath returns URL for downloading the filter or empty string
// if the filter is not remote
func (f *Filter) GetDownloadUrl() string {
	if f.Url != "" {
		return FilterNameToUrl(f.Url, f.Filter)
	} else if f.Filter != "" {
		return FilterNameToUrl(StandardLibraryUrl, f.Filter)
	}
	return ""
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

// Download downloads the filter and returns the download path. If the filter
// is not remote, it downloads the dependencies of the filter and returns
// empty string. The profileDir is a path to the directory of the profile that
// owns the filter (the directory of either the config.json or filter.json
// file). The profileDir combined with Script property of the filter gives
// the absolute path to the script.
func (f *Filter) Download(isForced bool, profileDir string) (string, error) {
	url := f.GetDownloadUrl()
	if url == "" {
		// Not a remote filter, download the dependencies
		if filterDefinition, ok := FilterTypes[f.RunWith]; ok {
			scriptPath, err := filepath.Abs(filepath.Join(profileDir, f.Script))
			if err != nil {
				return "", wrapError(fmt.Sprintf(
					"Unable to resolve path of %s script",
					f.GetFriendlyName()), err)
			}
			err = filterDefinition.install(*f, filepath.Dir(scriptPath))
			if err != nil {
				return "", wrapError(fmt.Sprintf(
					"Couldn't install filter dependencies %s",
					f.GetFriendlyName()), err)
			}
		} else {
			Logger.Warnf(
				"Filter type '%s' not supported", f.RunWith)
		}
		return "", nil // The filter is not downloaded (just dependencies)
	}

	// Download the filter into the cache folder
	downloadPath := UrlToPath(url)

	// If downloadPath already exists, we don't need to download it again.
	// Force mode allows overwriting.
	if _, err := os.Stat(downloadPath); err == nil {
		if !isForced {
			Logger.Warnf("Dependency %s already installed, skipping. Run "+
				"with '-f' to force.", url)
			return "", nil
		} else {
			Logger.Warnf("Dependency %s already installed, but forcing "+
				"installation.", url)
			err := os.RemoveAll(downloadPath)
			if err != nil {
				return "", wrapError("Could not remove installed filter", err)
			}
		}
	}

	Logger.Infof("Installing dependency %s...", url)

	// Download the filter fresh
	ok, err := DownloadGitHubUrl(url, downloadPath)
	if err != nil {
		Logger.Debug(err)
	}
	if !ok {
		Logger.Debug(
			"Failed to download filter " + f.Filter + " without git")
		err := getter.Get(downloadPath, url)
		if err != nil {
			return "", err
		}
	}

	// Remove 'test' folder, which may be installed via git-getter library
	// This is a workaround for cases where our own getter library is not
	// able to download the filter.
	testFolder := path.Join(downloadPath, "test")
	if _, err := os.Stat(testFolder); err == nil {
		os.RemoveAll(testFolder)
		if err != nil {
			Logger.Debug("Could not remove test folder", err)
		}
	}
	return downloadPath, nil
}

type ExportTarget struct {
	Target    string `json:"target,omitempty"` // The mode of exporting. "develop" or "exact"
	RpPath    string `json:"rpPath,omitempty"` // Relative or absolute path to resource pack for "exact" export target
	BpPath    string `json:"bpPath,omitempty"` // Relative or absolute path to resource pack for "exact" export target
	WorldName string `json:"worldName,omitempty"`
	WorldPath string `json:"worldPath,omitempty"`
	ReadOnly  bool   `json:"readOnly"` // Whether the exported files should be read-only
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
