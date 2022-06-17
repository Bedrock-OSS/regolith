// Functions for accessing informations from the config file without parsing it
// to a Config object. This is useful for accessing the config information
// for functions that modify the content of the config file, like
// "regolith install" and for accessing the config information when the file
// might have some errors.
package regolith

import (
	"io/ioutil"

	"muzzammil.xyz/jsonc"
)

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

// dataPathFromConfigMap returns the value of the data path from the config
// file map, without parsing it to a Config object.
func dataPathFromConfigMap(config map[string]interface{}) (string, error) {
	regolith, ok := config["regolith"].(map[string]interface{})
	if !ok {
		return "", WrappedError("Missing \"regolith\" property.")
	}
	dataPath, ok := regolith["dataPath"].(string)
	if !ok {
		return "", WrappedError("Missing \"regolith\"->\"dataPath\" property.")
	}
	return dataPath, nil
}

// filterDefinitionFromConfigMap returns the filter definitions as map from
// the config file map, without parsing it to a Config object.
func filterDefinitionsFromConfigMap(
	config map[string]interface{},
) (map[string]interface{}, error) {
	regolith, ok := config["regolith"].(map[string]interface{})
	if !ok {
		return nil, WrappedError("Missing \"regolith\" property.")
	}
	filterDefinitions, ok := regolith["filterDefinitions"].(map[string]interface{})
	if !ok {
		return nil, WrappedError("Missing \"regolith\"->\"filterDefinitions\" property.")
	}
	return filterDefinitions, nil
}
