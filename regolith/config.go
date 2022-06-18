package regolith

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
