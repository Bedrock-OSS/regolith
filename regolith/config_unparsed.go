package regolith

import (
	"os"

	"github.com/Bedrock-OSS/go-burrito/burrito"

	"github.com/muhammadmuzzammil1998/jsonc"
)

// Functions for accessing information from the config file without parsing it
// to a Config object. This is useful for accessing the config information
// for functions that modify the content of the config file, like
// "regolith install" and for accessing the config information when the file
// might have some errors.

// LoadConfigAsMap loads the config.json file as map[string]interface{}
func LoadConfigAsMap() (map[string]any, error) {
	err := CheckSuspiciousLocation()
	if err != nil {
		return nil, burrito.PassError(err)
	}
	file, err := os.ReadFile(ConfigFilePath)
	if err != nil {
		return nil, burrito.WrappedError( // We don't need to pass OS error. It's confusing.
			"Failed to open \"config.json\". This directory is not a Regolith project.\n" +
				"Please make sure to run this command in a Regolith project directory.\n" +
				"If you want to create new Regolith project here, use \"regolith init\".")
	}
	var configJson map[string]any
	err = jsonc.Unmarshal(file, &configJson)
	if err != nil {
		return nil, burrito.WrapErrorf(err, jsonUnmarshalError, ConfigFilePath)
	}
	return configJson, nil
}

// dataPathFromConfigMap returns the value of the data path from the config
// file map, without parsing it to a Config object.
func dataPathFromConfigMap(config map[string]any) (string, error) {
	return FindByJSONPath[string](config, "regolith/dataPath")
}

// filterDefinitionFromConfigMap returns the filter definitions as map from
// the config file map, without parsing it to a Config object.
func filterDefinitionsFromConfigMap(
	config map[string]any,
) (map[string]any, error) {
	return FindByJSONPath[map[string]any](config, "regolith/filterDefinitions")
}
