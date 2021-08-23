package src

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/fatih/color"
)

const ManifestName = "regolith.json"

type Project struct {
	Name     string             `json:"name"`
	Profiles map[string]Profile `json:"profiles"`
}

type Profile struct {
	Unsafe       bool         `json:"unsafe"`
	Filters      []Filter     `json:"filters"`
	ExportTarget ExportTarget `json:"export"`
}

type Filter struct {
	Name      string                 `json:"name"`
	Location  string                 `json:"location"`
	RunWith   string                 `json:"run_with"`
	Command   string                 `json:"command"`
	Arguments []string               `json:"arguments"`
	Url       string                 `json:"url"`
	Filter    string                 `json:"filter"`
	Settings  map[string]interface{} `json:"settings"`
}

type ExportTarget struct {
	Target        string `json:"target"`
	ComMojangPath string `json:"com_mojang_path"`
	WorldName     string `json:"world_name"`
	WorldPath     string `json:"world_path"`
	Path          string `json:"path"`
}

func IsConfigExists() bool {
	info, err := os.Stat(ManifestName)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func LoadConfig() Project {
	file, err := ioutil.ReadFile(ManifestName)
	if err != nil {
		Logger.Fatalf("Couldn't find %s! Consider running 'regolith init'", ManifestName)
	}
	var result Project
	err = json.Unmarshal(file, &result)
	if err != nil {
		Logger.Fatal(fmt.Sprintf("Couldn't load %s: ", ManifestName), err)
	}
	return result
}

func InitializeRegolithProject(isForced bool) bool {

	// Do not attempt to initialize if project is already initialized (can be forced)
	if !isForced && IsConfigExists() {
		Logger.Errorf("Could not initialize Regolith project. File %s already exists.", ManifestName)
		return false
	} else {
		Logger.Info("Initializing Regolith project...")

		if isForced {
			Logger.Warn("Warning: Initialization forced. Data may be lost.")
		}

		// Delete old configuration
		err := os.Remove(ManifestName)

		// Create new configuration
		file, err := os.Create(ManifestName)
		if err != nil {
			log.Fatal(color.RedString("Could not create %s: ", ManifestName), err)
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				Logger.Fatal("Failed to close the file")
			}
		}(file)

		// Write default configuration
		jsonData := Project{
			Profiles: map[string]Profile{
				"default": {
					Unsafe:  false,
					Filters: []Filter{},
					ExportTarget: ExportTarget{
						Target: "development",
					},
				},
			},
		}
		jsonBytes, _ := json.MarshalIndent(jsonData, "", "  ")
		_, err = file.Write(jsonBytes)
		if err != nil {
			Logger.Fatal("Failed to write project file contents")
		}
		Logger.Info("Regolith project initialized.")

		// Create .regolith folder
		err = os.Mkdir(".regolith", 0777)
		if err != nil {
			Logger.Fatal("Could not create .regolith folder", err)
		}

		return true
	}
}
