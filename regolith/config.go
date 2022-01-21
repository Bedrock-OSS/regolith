package regolith

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

const StandardLibraryUrl = "github.com/Bedrock-OSS/regolith-filters"
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
	Profiles      map[string]Profile      `json:"profiles,omitempty"`
	Installations map[string]Installation `json:"installations,omitempty"`
}

// LoadConfigAsMap loads the config.json file as a map[string]interface{}
func LoadConfigAsMap() map[string]interface{} {
	file, err := ioutil.ReadFile(ManifestName)
	if err != nil {
		Logger.Fatalf(
			"Couldn't find %s! Consider running 'regolith init'.",
			ManifestName)
	}
	var configJson map[string]interface{}
	err = json.Unmarshal(file, &configJson)
	if err != nil {
		Logger.Fatalf(
			"Couldn't load %s! Does the file contain correct json?",
			ManifestName)
	}
	return configJson
}

func ConfigFromObject(obj map[string]interface{}) *Config {
	result := &Config{}
	// Name
	name, ok := obj["name"].(string)
	if !ok {
		Logger.Fatal("Could not find name in config.json")
	}
	result.Name = name
	// Author
	author, ok := obj["author"].(string)
	if !ok {
		Logger.Fatal("Could not find author in config.json")
	}
	result.Author = author
	// Packs
	packs, ok := obj["packs"].(map[string]interface{})
	if !ok {
		Logger.Fatal("Could not find packs in config.json")
	}
	result.Packs = PacksFromObject(packs)
	// Regolith
	regolith, ok := obj["regolith"].(map[string]interface{})
	if !ok {
		Logger.Fatal("Could not find regolith in config.json")
	}
	result.RegolithProject = RegolithProjectFromObject(regolith)
	return result
}

func PacksFromObject(obj map[string]interface{}) Packs {
	result := Packs{}
	// BehaviorPack
	behaviorPack, _ := obj["behaviorPack"].(string)
	result.BehaviorFolder = behaviorPack
	// ResourcePack
	resourcePack, _ := obj["resourcePack"].(string)
	result.ResourceFolder = resourcePack
	return result
}

func RegolithProjectFromObject(obj map[string]interface{}) RegolithProject {
	result := RegolithProject{
		Profiles:      make(map[string]Profile),
		Installations: make(map[string]Installation),
	}
	profiles, ok := obj["profiles"].(map[string]interface{})
	if !ok {
		Logger.Fatal("Could not find profiles in config.json")
	}
	installations, ok := obj["installations"].(map[string]interface{})
	if ok { // Installations are optional
		for installationName, installation := range installations {
			installationMap, ok := installation.(map[string]interface{})
			if !ok {
				Logger.Fatal("invalid format of installation %s in config.json", installationName)
			}
			result.Installations[installationName] = InstallationFromObject(
				installationName, installationMap)
		}
	}
	for profileName, profile := range profiles {
		profileMap, ok := profile.(map[string]interface{})
		if !ok {
			Logger.Fatal("Could not find profile in config.json")
		}
		result.Profiles[profileName] = ProfileFromObject(
			profileName, profileMap, result.Installations)

	}
	return result
}

func ExportTargetFromObject(obj map[string]interface{}) ExportTarget {
	// TODO - implement in a proper way
	result := ExportTarget{}
	// Target
	target, ok := obj["target"].(string)
	if !ok {
		Logger.Fatal("Could not find target in config.json")
	}
	result.Target = target
	// RpPath
	rpPath, _ := obj["rpPath"].(string)
	result.RpPath = rpPath
	// BpPath
	bpPath, _ := obj["bpPath"].(string)
	result.BpPath = bpPath
	// WorldName
	worldName, _ := obj["worldName"].(string)
	result.WorldName = worldName
	// WorldPath
	worldPath, _ := obj["worldPath"].(string)
	result.WorldPath = worldPath
	// ReadOnly
	readOnly, _ := obj["readOnly"].(bool)
	result.ReadOnly = readOnly
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
							Filters: []FilterRunner{
								&RemoteFilter{Id: "hello_world"},
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

// Recursively install dependencies for the entire config.
//  - Force mode will overwrite existing dependencies.
//  - Non-force mode will only install dependencies that are not already installed.
func (c *Config) InstallFilters(isForced bool, updateFilters bool) error {
	CreateDirectoryIfNotExists(".regolith/cache/filters", true)
	CreateDirectoryIfNotExists(".regolith/cache/venvs", true)

	c.DownloadInstallations(isForced, updateFilters)
	for profileName, profile := range c.Profiles {
		Logger.Infof("Installing profile %s...", profileName)
		err := profile.Install(isForced)
		if err != nil {
			return wrapError("Could not install profile!", err)
		}
	}
	Logger.Infof("All filters installed.")
	return nil
}

func (c *Config) DownloadInstallations(isForced bool, updateFilters bool) error {
	CreateDirectoryIfNotExists(".regolith/cache/filters", true)
	CreateDirectoryIfNotExists(".regolith/cache/venvs", true)

	for name := range c.Installations {
		item := c.Installations[name]
		Logger.Infof("Downloading %q...", name)
		err := item.Download(isForced)
		if err != nil {
			return wrapError(
				fmt.Sprintf("Could not download %q!", name),
				err)
		}
	}
	Logger.Infof("All remote filters installed.")
	return nil
}
