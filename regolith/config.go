package regolith

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

const StandardLibraryUrl = "github.com/Bedrock-OSS/regolith-filters"
const ConfigFilePath = "config.json"
const GitIgnore = `/build
/.regolith`

// Config represents the full configuration file of Regolith, as saved in
// "config.json".
type Config struct {
	Name            string `json:"name,omitempty"`
	Author          string `json:"author,omitempty"`
	Packs           `json:"packs,omitempty"`
	RegolithProject `json:"regolith,omitempty"`
}

// ExportTarget is a part of "config.json" that contains export information
// for a profile, which denotes where compiled files will go.
type ExportTarget struct {
	Target    string `json:"target,omitempty"` // The mode of exporting. "develop" or "exact"
	RpPath    string `json:"rpPath,omitempty"` // Relative or absolute path to resource pack for "exact" export target
	BpPath    string `json:"bpPath,omitempty"` // Relative or absolute path to resource pack for "exact" export target
	WorldName string `json:"worldName,omitempty"`
	WorldPath string `json:"worldPath,omitempty"`
	ReadOnly  bool   `json:"readOnly"` // Whether the exported files should be read-only
}

// Packs is a part of "config.json" that points to the source behavior and
// resource packs.
type Packs struct {
	BehaviorFolder string `json:"behaviorPack,omitempty"`
	ResourceFolder string `json:"resourcePack,omitempty"`
}

// RegolithProject is a part of "config.json" whith the regolith namespace
// within the Minecraft Project Schema
type RegolithProject struct {
	Profiles          map[string]Profile         `json:"profiles,omitempty"`
	FilterDefinitions map[string]FilterInstaller `json:"filterDefinitions"`
	DataPath          string                     `json:"dataPath,omitempty"`
}

// LoadConfigAsMap loads the config.json file as map[string]interface{}
func LoadConfigAsMap() (map[string]interface{}, error) {
	file, err := ioutil.ReadFile(ConfigFilePath)
	if err != nil {
		return nil, WrapErrorf(
			err,
			"%q not found (use 'regolith init' to initialize the project)",
			ConfigFilePath)
	}
	var configJson map[string]interface{}
	err = json.Unmarshal(file, &configJson)
	if err != nil {
		return nil, WrapErrorf(
			err, "could not load %s as a JSON file", ConfigFilePath)
	}
	return configJson, nil
}

// ConfigFromObject creates a "Config" object from map[string]interface{}
func ConfigFromObject(obj map[string]interface{}) (*Config, error) {
	result := &Config{}
	// Name
	name, ok := obj["name"].(string)
	if !ok {
		return nil, WrapError(nil, "missing 'name' property")
	}
	result.Name = name
	// Author
	author, ok := obj["author"].(string)
	if !ok {
		return nil, WrapError(nil, "missing 'author' property")
	}
	result.Author = author
	// Packs
	if packs, ok := obj["packs"]; ok {
		packs, ok := packs.(map[string]interface{})
		if !ok {
			return nil, WrapErrorf(
				nil, "'packs' property is a %T, not a map", packs)
		}
		// Packs can be empty, no need to check for errors
		result.Packs = PacksFromObject(packs)
	} else {
		return nil, WrapError(nil, "missing 'packs' property")
	}
	// Regolith
	if regolith, ok := obj["regolith"]; ok {
		regolith, ok := regolith.(map[string]interface{})
		if !ok {
			return nil, WrapErrorf(
				nil, "'regolith' property is a %T, not a map", regolith)
		}
		regolithProject, err := RegolithProjectFromObject(regolith)
		if err != nil {
			return nil, WrapError(err, "could not parse 'regolith' property")
		}
		result.RegolithProject = regolithProject
	} else {
		return nil, WrapError(nil, "missing 'regolith' property")
	}
	return result, nil
}

// ProfileFromObject creates a "Profile" object from map[string]interface{}
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

// RegolithProjectFromObject creates a "RegolithProject" object from
// map[string]interface{}
func RegolithProjectFromObject(
	obj map[string]interface{},
) (RegolithProject, error) {
	result := RegolithProject{
		Profiles:          make(map[string]Profile),
		FilterDefinitions: make(map[string]FilterInstaller),
	}
	// DataPath
	if _, ok := obj["dataPath"]; !ok {
		return result, WrapError(nil, "missing 'dataPath' property")
	}
	dataPath, ok := obj["dataPath"].(string)
	if !ok {
		return result, WrapErrorf(
			nil, "'dataPath' is a %T, not a string", obj["dataPath"])
	}
	result.DataPath = dataPath
	// Filter definitions
	filterDefinitions, ok := obj["filterDefinitions"].(map[string]interface{})
	if ok { // filter definitions are optional
		for filterDefinitionName, filterDefinition := range filterDefinitions {
			filterDefinitionMap, ok := filterDefinition.(map[string]interface{})
			if !ok {
				return result, WrapErrorf(
					nil, "filter definition %q is a %T not a map",
					filterDefinitionName, filterDefinitions[filterDefinitionName])
			}
			filterInstaller, err := FilterInstallerFromObject(
				filterDefinitionName, filterDefinitionMap)
			if err != nil {
				return result, WrapError(
					err, "could not parse filter definition")
			}
			result.FilterDefinitions[filterDefinitionName] = filterInstaller
		}
	}
	// Profiles
	profiles, ok := obj["profiles"].(map[string]interface{})
	if !ok {
		return result, WrapError(nil, "missing 'profiles' property")
	}
	for profileName, profile := range profiles {
		profileMap, ok := profile.(map[string]interface{})
		if !ok {
			return result, WrapErrorf(
				nil, "profile %q is a %T not a map",
				profileName, profiles[profileName])
		}
		profileValue, err := ProfileFromObject(
			profileMap, result.FilterDefinitions)
		if err != nil {
			return result, WrapErrorf(
				err, "could not parse profile %q", profileName)
		}
		result.Profiles[profileName] = profileValue
	}
	return result, nil
}

// ExportTargetFromObject creates a "ExportTarget" object from
// map[string]interface{}
func ExportTargetFromObject(obj map[string]interface{}) (ExportTarget, error) {
	// TODO - implement in a proper way
	result := ExportTarget{}
	// Target
	target, ok := obj["target"].(string)
	if !ok {
		return result, WrapError(nil, "could not find 'target' in config.json")
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
	return result, nil
}

// IsProjectInitialized checks if the project is initialized by testing if
// the config.json exists.
func IsProjectInitialized() bool {
	info, err := os.Stat(ConfigFilePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// InitializeRegolithProject handles the "regolith init" command. It creates
// "config.json", ".gitignore" and required folders.
func InitializeRegolithProject(isForced bool) error {
	Logger.Info("Initializing Regolith project...")

	if !isForced && IsProjectInitialized() {
		return WrapErrorf(
			nil,
			"%q already exists, suggesting this project is already initialized. You may use --force to override this check.",
			ConfigFilePath)
	} else {
		if isForced {
			Logger.Warn("Initialization forced. Data may be lost.")
		}

		// Delete old configuration if it exists
		if err := os.Remove(ConfigFilePath); !os.IsNotExist(err) {
			if err != nil {
				return WrapErrorf(err, "Failed to remove old %q", ConfigFilePath)
			}
		}

		// Create new default configuration
		jsonData := Config{
			Name:   "Project name",
			Author: "Your name",
			Packs: Packs{
				BehaviorFolder: "./packs/BP",
				ResourceFolder: "./packs/RP",
			},
			RegolithProject: RegolithProject{
				DataPath:          "./packs/data",
				FilterDefinitions: map[string]FilterInstaller{},
				Profiles: map[string]Profile{
					"dev": {
						FilterCollection: FilterCollection{
							Filters: []FilterRunner{},
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

		err := ioutil.WriteFile(ConfigFilePath, jsonBytes, 0666)
		if err != nil {
			return WrapErrorf(err, "Failed to write data to %q", ConfigFilePath)
		}

		ioutil.WriteFile(".gitignore", []byte(GitIgnore), 0666)

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

// CleanCache handles "regolith clean" command it removes all contents of
// ".regolith" folder.
func CleanCache() error {
	Logger.Infof("Cleaning cache...")
	err := os.RemoveAll(".regolith")
	if err != nil {
		return WrapError(err, "failed to remove .regolith folder")
	}
	err = os.Mkdir(".regolith", 0666)
	if err != nil {
		return WrapError(err, "failed to recreate .regolith folder")
	}
	Logger.Infof("Cache cleaned.")
	return nil
}

// InstallFilters handles the "regolith install" command (without the --add
// flag). It downloads all of the filters from "filter_definitions"
// of the Config and/or installs their dependencies.
// isForced toggles the force mode. The force mode overwrites existing
// dependencies. Non-force mode only installs dependencies that are not
// already installed.
func (c *Config) InstallFilters(isForced bool) error {
	err := CreateDirectoryIfNotExists(".regolith/cache/filters", true)
	if err != nil {
		return WrapError(nil, err.Error())
	}
	err = CreateDirectoryIfNotExists(".regolith/cache/venvs", true)
	if err != nil {
		return WrapError(nil, err.Error())
	}

	err = c.DownloadRemoteFilters(isForced)
	if err != nil {
		return WrapError(err, "failed to download filters")
	}
	for filterName, filterDefinition := range c.FilterDefinitions {
		err = filterDefinition.InstallDependencies(nil)
		if err != nil {
			return WrapErrorf(
				err, "failed to install dependencies for filter %q",
				filterName)
		}
	}
	Logger.Infof("All filters installed.")
	return nil
}

// DownloadRemoteFilters downloads all of the remote filters from
// "filter_definitions" of the Confing.
// isForced toggles the force mode described in InstallFilters.
func (c *Config) DownloadRemoteFilters(isForced bool) error {
	err := CreateDirectoryIfNotExists(".regolith/cache/filters", true)
	if err != nil {
		return WrapError(nil, err.Error())
	}
	err = CreateDirectoryIfNotExists(".regolith/cache/venvs", true)
	if err != nil {
		return WrapError(nil, err.Error())
	}

	for name := range c.FilterDefinitions {
		filterDefinition := c.FilterDefinitions[name]
		Logger.Infof("Downloading %q...", name)
		switch remoteFilter := filterDefinition.(type) {
		case *RemoteFilterDefinition:
			err := remoteFilter.Download(isForced)
			if err != nil {
				return WrapErrorf(
					err, "could not download %q!", name)
			}
			remoteFilter.CopyFilterData(c.DataPath)
		}
	}
	Logger.Infof("All remote filters installed.")
	return nil
}
