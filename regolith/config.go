package regolith

import (
	"encoding/json"
	"io/ioutil"
	"os"
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
