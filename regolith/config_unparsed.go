// Functions for accessing informations from the config file without parsing it
// to a Config object. This is useful for accessing the config information
// for functions that modify the content of the config file, like
// "regolith install" and for accessing the config information when the file
// might have some errors.
package regolith

import (
	"github.com/Bedrock-OSS/go-burrito/burrito"
	"io/ioutil"

	"muzzammil.xyz/jsonc"
)

// LoadConfigAsMap loads the config.json file as map[string]interface{}
func LoadConfigAsMap() (map[string]interface{}, error) {
	file, err := ioutil.ReadFile(ConfigFilePath)
	if err != nil {
		return nil, burrito.WrappedError( // We don't need to pass OS error. It's confusing.
			"Failed to open \"config.json\". This directory is not a Regolith project.\n" +
				"Please make sure to run this command in a Regolith project directory.\n" +
				"If you want to create new Regolith project here, use \"regolith init\".")
	}
	var configJson map[string]interface{}
	err = jsonc.Unmarshal(file, &configJson)
	if err != nil {
		return nil, burrito.WrapErrorf(err, jsonUnmarshalError, ConfigFilePath)
	}
	return configJson, nil
}

// dataPathFromConfigMap returns the value of the data path from the config
// file map, without parsing it to a Config object.
func dataPathFromConfigMap(config map[string]interface{}) (string, error) {
	regolith, ok := config["regolith"].(map[string]interface{})
	if !ok {
		return "", burrito.WrappedErrorf(jsonPathMissingError, "regolith")
	}
	dataPath, ok := regolith["dataPath"].(string)
	if !ok {
		return "", burrito.WrappedErrorf(jsonPathMissingError, "regolith->dataPath")
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
		return nil, burrito.WrappedErrorf(jsonPathMissingError, "regolith")
	}
	filterDefinitions, ok := regolith["filterDefinitions"].(map[string]interface{})
	if !ok {
		return nil, burrito.WrappedErrorf(
			jsonPathMissingError, "regolith->filterDefinitions")
	}
	return filterDefinitions, nil
}

// useAppDataFromConfigMap returns the useAppData value from the config file
// map, without parsing it to a Config object.
func useAppDataFromConfigMap(config map[string]interface{}) (bool, error) {
	regolith, ok := config["regolith"].(map[string]interface{})
	if !ok {
		return false, burrito.WrappedErrorf(jsonPathMissingError, "regolith")
	}
	filterDefinitionsInterface, ok := regolith["useAppData"]
	if !ok { // false by default
		return false, nil
	}
	filterDefinitions, ok := filterDefinitionsInterface.(bool)
	if !ok {
		return false, burrito.WrappedErrorf(
			jsonPathTypeError, "regolith->useAppData", "bool")
	}
	return filterDefinitions, nil
}
