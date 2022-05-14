package regolith

import (
	"io/ioutil"

	"muzzammil.xyz/jsonc"
)

const StandardLibraryUrl = "github.com/Bedrock-OSS/regolith-filters"
const ConfigFilePath = "config.json"
const GitIgnore = "/build\n/.regolith"

var ConfigurationFolders = []string{
	"packs",
	"packs/data",
	"packs/BP",
	"packs/RP",
	".regolith",
	".regolith/cache",
	".regolith/cache/venvs",
}

// Config represents the full configuration file of Regolith, as saved in
// "config.json".
type Config struct {
	Name            string `json:"name,omitempty"`
	Author          string `json:"author,omitempty"`
	Packs           `json:"packs,omitempty"`
	RegolithProject `json:"regolith,omitempty"`

	// interruptionChannel is a channel that is used to notify about changes
	// in the sourec files, in order to trigger a restart of the program in
	// the watch mode. The string send to the channel is the name of the source
	// of the change ("rp", "bp" or "data"), which may be used to handle
	// some interuptions differently.
	interruptionChannel chan string
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
			"%q not found (use \"regolith init\" to initialize the project).",
			ConfigFilePath)
	}
	var configJson map[string]interface{}
	err = jsonc.Unmarshal(file, &configJson)
	if err != nil {
		return nil, WrapErrorf(
			err, "Could not load %q as a JSON file.", ConfigFilePath)
	}
	return configJson, nil
}

// ConfigFromObject creates a "Config" object from map[string]interface{}
func ConfigFromObject(obj map[string]interface{}) (*Config, error) {
	result := &Config{}
	// Name
	name, ok := obj["name"].(string)
	if !ok {
		return nil, WrappedError("The \"name\" property is missing.")
	}
	result.Name = name
	// Author
	author, ok := obj["author"].(string)
	if !ok {
		return nil, WrappedError("The \"author\" is missing.")
	}
	result.Author = author
	// Packs
	if packs, ok := obj["packs"]; ok {
		packs, ok := packs.(map[string]interface{})
		if !ok {
			return nil, WrappedError("The \"packs\" property not a map.")
		}
		// Packs can be empty, no need to check for errors
		result.Packs = PacksFromObject(packs)
	} else {
		return nil, WrappedError("The \"packs\" property is missing.")
	}
	// Regolith
	if regolith, ok := obj["regolith"]; ok {
		regolith, ok := regolith.(map[string]interface{})
		if !ok {
			return nil, WrappedError("The \"regolith\" property is not a map.")
		}
		regolithProject, err := RegolithProjectFromObject(regolith)
		if err != nil {
			return nil, WrapError(
				err, "Could not parse \"regolith\" property.")
		}
		result.RegolithProject = regolithProject
	} else {
		return nil, WrappedError("Missing \"regolith\" property.")
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
		return result, WrappedError("The \"dataPath\" property is missing.")
	}
	dataPath, ok := obj["dataPath"].(string)
	if !ok {
		return result, WrappedErrorf("The \"dataPath\" is not a string")
	}
	result.DataPath = dataPath
	// Filter definitions
	filterDefinitions, ok := obj["filterDefinitions"].(map[string]interface{})
	if ok { // filter definitions are optional
		for filterDefinitionName, filterDefinition := range filterDefinitions {
			filterDefinitionMap, ok := filterDefinition.(map[string]interface{})
			if !ok {
				return result, WrappedErrorf(
					"The filter definition %q not a map.", filterDefinitionName)
			}
			filterInstaller, err := FilterInstallerFromObject(
				filterDefinitionName, filterDefinitionMap)
			if err != nil {
				return result, WrapError(
					err, "Could not parse the filter definition.")
			}
			result.FilterDefinitions[filterDefinitionName] = filterInstaller
		}
	}
	// Profiles
	profiles, ok := obj["profiles"].(map[string]interface{})
	if !ok {
		return result, WrappedError("Missing \"profiles\" property.")
	}
	for profileName, profile := range profiles {
		profileMap, ok := profile.(map[string]interface{})
		if !ok {
			return result, WrappedErrorf(
				"Profile %q is not a map.", profileName)
		}
		profileValue, err := ProfileFromObject(
			profileMap, result.FilterDefinitions)
		if err != nil {
			return result, WrapErrorf(
				err, "Could not parse %q profile.", profileName)
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
		return result, WrappedError(
			"The\"target\" property in \"config.json\" is missing.")
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

// InstallFilters handles the "regolith install" command (without the --add
// flag). It downloads all of the filters from "filter_definitions"
// of the Config and/or installs their dependencies.
// isForced toggles the force mode. The force mode overwrites existing
// dependencies. Non-force mode only installs dependencies that are not
// already installed.
func (c *Config) InstallFilters(isForced bool) error {
	err := CreateDirectoryIfNotExists(".regolith/cache/filters", true)
	if err != nil {
		return PassError(err)
	}
	err = CreateDirectoryIfNotExists(".regolith/cache/venvs", true)
	if err != nil {
		return PassError(err)
	}

	err = c.DownloadRemoteFilters(isForced)
	if err != nil {
		return WrapError(err, "Downloading remote filters has failed.")
	}
	for filterName, filterDefinition := range c.FilterDefinitions {
		Logger.Infof("Installing %q filter dependencies...", filterName)
		err = filterDefinition.InstallDependencies(nil)
		if err != nil {
			return WrapErrorf(
				err, "Failed to install dependencies for %q filter.",
				filterName)
		}
	}
	return nil
}

// DownloadRemoteFilters downloads all of the remote filters from
// "filter_definitions" of the Confing.
// isForced toggles the force mode described in InstallFilters.
func (c *Config) DownloadRemoteFilters(isForced bool) error {
	err := CreateDirectoryIfNotExists(".regolith/cache/filters", true)
	if err != nil {
		return PassError(err)
	}
	err = CreateDirectoryIfNotExists(".regolith/cache/venvs", true)
	if err != nil {
		return PassError(err)
	}

	for name := range c.FilterDefinitions {
		filterDefinition := c.FilterDefinitions[name]
		Logger.Infof("Downloading %q filter...", name)
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
	return nil
}

// StartWatchingSourceFiles causes the Config to start goroutines that watch
// for changes in the source files and report that to the
func (c *Config) StartWatchingSrouceFiles() error {
	// TODO - if you want to be able to restart the watcher, you need to handle
	// closing the channels somewhere. Currently the watching goroutines yield
	// their messages until the end of the program. Sending to a closed channel
	// would cause panic.
	if c.interruptionChannel != nil {
		return WrappedError("The Config is already watching source files.")
	}
	rpWatcher, err := NewDirWatcher(c.ResourceFolder)
	if err != nil {
		return WrapError(err, "Could not create resource pack watcher.")
	}
	bpWatcher, err := NewDirWatcher(c.BehaviorFolder)
	if err != nil {
		return WrapError(err, "Could not create behavior pack watcher.")
	}
	dataWatcher, err := NewDirWatcher(c.DataPath)
	if err != nil {
		return WrapError(err, "Could not create data watcher.")
	}
	c.interruptionChannel = make(chan string)
	yieldChanges := func(
		watcher *DirWatcher, sourceName string,
	) {
		for {
			err := watcher.WaitForChangeGroup(100)
			if err != nil {
				return
			}
			c.interruptionChannel <- sourceName
		}
	}
	go yieldChanges(rpWatcher, "rp")
	go yieldChanges(bpWatcher, "bp")
	go yieldChanges(dataWatcher, "data")
	return nil
}

// AwaitInterruption locks the goroutine with the interruption channel until
// the Config is interrupted and returns the interruption message.
func (c *Config) AwaitInterruption() string {
	return <-c.interruptionChannel
}

// IsInterrupted returns true if there is a message on the interruptionChannel
// unless the source of the interruption is on the list of ignored sources.
// This function does not block.
func (c *Config) IsInterrupted(ignoredSourece ...string) bool {
	if c.interruptionChannel == nil {
		return false
	}
	select {
	case source := <-c.interruptionChannel:
		for _, ignored := range ignoredSourece {
			if ignored == source {
				return false
			}
		}
		return true
	default:
		return false
	}
}
