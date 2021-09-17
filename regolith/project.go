package regolith

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

const ManifestName = "config.json"

// TODO implement the rest of the standard config spec
type Config struct {
	Name            string `json:"name"`
	Author          string `json:"author"`
	Packs           `json:"packs"`
	RegolithProject `json:"regolith"`
}

type Packs struct {
	BehaviorFolder string `json:"behaviorPack"`
	ResourceFolder string `json:"resourcePack"`
}

type RegolithProject struct {
	Profiles map[string]Profile `json:"profiles"`
}

type Profile struct {
	Unsafe       bool         `json:"unsafe"`
	Filters      []Filter     `json:"filters"`
	ExportTarget ExportTarget `json:"export"`
	DataPath     string       `json:"dataPath"`
}

type Filter struct {
	Name      string                 `json:"name"`
	Location  string                 `json:"location"`
	RunWith   string                 `json:"runWith"`
	Command   string                 `json:"command"`
	Arguments []string               `json:"arguments"`
	Url       string                 `json:"url"`
	Filter    string                 `json:"filter"`
	Settings  map[string]interface{} `json:"settings"`
	VenvSlot  int                    `json:"venvSlot"`
}

type ExportTarget struct {
	Target    string `json:"target"` // The mode of exporting. "develop" or "exact"
	RpPath    string `json:"rpPath"` // Relative or absolute path to resource pack for "exact" export target
	BpPath    string `json:"bpPath"` // Relative or absolute path to resource pack for "exact" export target
	WorldName string `json:"worldName"`
	WorldPath string `json:"worldPath"`
	// ComMojangPath string `json:"comMojangPath"`
	// NOT USED, DISABLED FOR NOW.
	// Clean         bool   `json:"clean"`
	// Path          string `json:"path"`
}

func IsProjectConfigured() bool {
	// TODO: Write a better system here, that checks all possible files.
	info, err := os.Stat(ManifestName)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
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

func InitializeRegolithProject(isForced bool) error {
	// Do not attempt to initialize if project is already initialized (can be forced)
	if !isForced && IsProjectConfigured() {
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
		file, err := os.Create(ManifestName)
		if err != nil {
			return wrapError(fmt.Sprintf("Could not create %s: ", ManifestName), err)
		}
		defer func(file *os.File) {
			err = file.Close()
		}(file)
		if err != nil {
			return wrapError(fmt.Sprintf("Could not close %s: ", ManifestName), err)
		}

		// Write default configuration
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
						Unsafe:  false,
						Filters: []Filter{},
						ExportTarget: ExportTarget{
							Target: "development",
						},
					},
				},
			},
		}
		jsonBytes, _ := json.MarshalIndent(jsonData, "", "  ")
		_, err = file.Write(jsonBytes)
		if err != nil {
			return wrapError("Failed to write project file contents", err)
		}

		// Create folders
		err = os.Mkdir("packs", 0777)
		if err != nil {
			Logger.Error("Could not create packs folder", err)
		}

		err = os.Mkdir("./packs/RP", 0777)
		if err != nil {
			Logger.Error("Could not create ./packs/RP folder", err)
		}

		err = os.Mkdir("./packs/BP", 0777)
		if err != nil {
			Logger.Error("Could not create ./packs/BP folder", err)
		}

		err = os.Mkdir(".regolith", 0777)
		if err != nil {
			Logger.Error("Could not create .regolith folder", err)
		}

		Logger.Info("Regolith project initialized.")
		return nil
	}
}
