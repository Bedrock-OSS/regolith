// Functions related to config.toml
package regolith

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type UserConfig struct {
	Project struct {
		AppDataStorage bool `toml:"app_data_storage"`
	} `toml:"project"`
	User struct {
		Name string `toml:"name"`
	} `toml:"user"`
	Resolvers []string `toml:"resolvers"`
}

func (u *UserConfig) fillDefaults() {
	u.Resolvers = append(u.Resolvers, resolverUrl)
	if u.User.Name == "" {
		u.User.Name = "Your name"
	}
}

var userConfig *UserConfig

func getAppDataConfigTomlPath() (string, error) {
	// App data enabled - use user cache dir
	userCache, err := os.UserCacheDir()
	if err != nil {
		return "", WrappedError(osUserCacheDirError)
	}
	return filepath.Join(userCache, "regolith", "config.toml"), nil
}

// readUserConfig reads the config from .regolith/config.toml and from the
// user app data directory. It returns the merged config and an error if any.
// The result is guaranteed to be a valid config even if there was an error.
func readUserConfig() (*UserConfig, error) {
	// Get the paths to localConfigPath and appDataConfig
	globalConfigPath, err := getAppDataConfigTomlPath()
	result := &UserConfig{}
	defer result.fillDefaults()
	if err != nil {
		return result, WrapError(err, "Failed to get global config.toml path")
	}
	localConfigPath, err := filepath.Abs(".regolith/config.toml")
	if err != nil {
		return result, WrapErrorf(err, filepathAbsError, ".regolith/config.toml")
	}
	// Load the config files
	// First load the global config
	if _, err := os.Stat(globalConfigPath); err == nil {
		if _, err := toml.DecodeFile(globalConfigPath, result); err != nil {
			return result, WrapErrorf(err, tomlUnmarshalError, globalConfigPath)
		}
	} else if !os.IsNotExist(err) {
		return result, WrapErrorf(err, osStatErrorAny, globalConfigPath)
	}
	// Overwrite with the local config
	if _, err := os.Stat(localConfigPath); err == nil {
		if _, err := toml.DecodeFile(localConfigPath, result); err != nil {
			return result, WrapErrorf(err, tomlUnmarshalError, localConfigPath)
		}
	} else if !os.IsNotExist(err) {
		return result, WrapErrorf(err, osStatErrorAny, localConfigPath)
	}
	return result, nil
}

// getUserConfig lazily loads the user config to the global userConfig variable
// and returns it. The pointer returned from this function is guaranteed to be
// non-nil.
func getUserConfig() *UserConfig {
	if userConfig == nil {
		readUserConfig, err := readUserConfig()
		if err != nil {
			Logger.Warn(WrapError(err, "Failed to read user config").Error())
		}
		userConfig = readUserConfig
	}
	return userConfig
}
