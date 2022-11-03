// Functions related to user_config.json
package regolith

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

type UserConfig struct {
	ProjectAppDataStorage bool     `json:"project_app_data_storage"`
	Username              string   `json:"username"`
	Resolvers             []string `json:"resolvers"`
}

func (u *UserConfig) fillDefaults() {
	u.Resolvers = append(u.Resolvers, resolverUrl)
	if u.Username == "" {
		u.Username = "Your name"
	}
}

var userConfig *UserConfig

func getAppDataConfigJsonPath() (string, error) {
	// App data enabled - use user cache dir
	userCache, err := os.UserCacheDir()
	if err != nil {
		return "", WrappedError(osUserCacheDirError)
	}
	return filepath.Join(userCache, "regolith", "user_config.json"), nil
}

// readCombinedUserConfig reads the config from .regolith/user_config.json and
// from the user app data directory. It returns the merged config or an error.
func readCombinedUserConfig() (*UserConfig, error) {
	// Get the paths to localConfigPath and appDataConfig
	result := &UserConfig{}
	defer result.fillDefaults()
	readConfigToResult := func(path string) error {
		if _, err := os.Stat(path); err == nil {
			file, err := ioutil.ReadFile(path)
			if err == nil {
				return WrapErrorf(err, fileReadError, path)
			}
			if err = json.Unmarshal(file, result); err != nil {
				return WrapErrorf(err, jsonUnmarshalError, path)
			}
		} else if !os.IsNotExist(err) {
			return WrapErrorf(err, osStatErrorAny, path)
		}
		return nil
	}

	globalConfigPath, err := getAppDataConfigJsonPath()
	if err != nil {
		return nil, WrapError(err, "Failed to get global user_config.json path")
	}
	localConfigPath, err := filepath.Abs(".regolith/user_config.json")
	if err != nil {
		return nil, WrapErrorf(err, filepathAbsError, ".regolith/user_config.json")
	}
	// Load the config files
	// First load the global config
	err = readConfigToResult(globalConfigPath)
	if err != nil {
		return nil, WrapError(err, "Failed to read global user_config.json")
	}
	// Overwrite with the local config
	err = readConfigToResult(localConfigPath)
	if err != nil {
		return nil, WrapError(err, "Failed to read local user_config.json")
	}
	return result, nil
}

// getUserConfig lazily loads the user config to the global userConfig variable
// and returns it. The pointer returned from this function is guaranteed to be
// non-nil.
func getUserConfig() (*UserConfig, error) {
	if userConfig == nil {
		readUserConfig, err := readCombinedUserConfig()
		if err != nil {
			return nil, PassError(err)
		}
		userConfig = readUserConfig
	}
	return userConfig, nil
}
