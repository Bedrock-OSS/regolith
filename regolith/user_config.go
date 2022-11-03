// Functions related to user_config.json
package regolith

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// localUserConfigPath is a path to the user config file
const localUserConfigPath = ".regolith/user_config.json"

// cachedUserConfig is a global variable that stores the combined user config. It
// should not be accessed directly, but instead through the getUserConfig
// function, which will lazily load the config if it hasn't been loaded yet.
var cachedUserConfig *UserConfig

type UserConfig struct {
	// UseProjectAppDataStorage is a flag that determines whether to use the
	// app data for project files (.regolith path). It's a pointer to a boolean
	// to allow for the default value to be nil.
	UseProjectAppDataStorage *bool `json:"use_project_app_data_storage,omitempty"`

	// Username is the name of the user. It's used in the "regolith init" command
	// to fill in the author field of the project config. It's a pointer to a
	// string to allow for the default value to be nil.
	Username *string `json:"username,omitempty"`

	// Resolvers is a list of URLs to resolvers that Regolith will use to find
	// filters for the "regolith install" command.
	Resolvers []string `json:"resolvers,omitempty"`
}

func NewUserConfig() *UserConfig {
	return &UserConfig{
		UseProjectAppDataStorage: nil,
		Username:                 nil,
		Resolvers:                []string{},
	}
}

func (u *UserConfig) String() string {
	result, _ := u.stringPropertyValue("use_project_app_data_storage")
	extra, _ := u.stringPropertyValue("username")
	result += "\n" + extra
	extra, _ = u.stringPropertyValue("resolvers")
	result += "\n" + extra
	return result
}

// stringPropertyValue returns a string with pretty formatted value of a
// specified property of the user config.
func (u *UserConfig) stringPropertyValue(name string) (string, error) {
	switch name {
	case "use_project_app_data_storage":
		value := "null"
		if u.UseProjectAppDataStorage != nil {
			value = fmt.Sprintf("%v", *u.UseProjectAppDataStorage)
		}
		return fmt.Sprintf("use_project_app_data_storage: %v", value), nil
	case "username":
		value := "null"
		if u.Username != nil {
			value = fmt.Sprintf("%v", *u.Username)
		}
		return fmt.Sprintf("username: %v", value), nil
	case "resolvers":
		if len(u.Resolvers) == 0 {
			return "resolvers: []", nil
		}
		result := "resolvers: \n"
		for i, resolver := range u.Resolvers {
			result += fmt.Sprintf("\t- [%v] %s\n", i, resolver)
		}
		return result, nil
	}
	return "", WrapErrorf(nil, invalidUserConfigPropertyError, name)
}

// fillDefaults fills the empty fields in the user config with default values.
func (u *UserConfig) fillDefaults() {
	if u.UseProjectAppDataStorage == nil {
		u.UseProjectAppDataStorage = new(bool)
		*u.UseProjectAppDataStorage = false
	}
	if u.Username == nil {
		u.Username = new(string)
		*u.Username = "Your name"
	}
	// Make sure resolvers is not nil and append the default resolver
	if u.Resolvers == nil {
		u.Resolvers = []string{}
	}
	u.Resolvers = append(u.Resolvers, resolverUrl)
}

// fillWithFileData fills the user config with the data loaded from a file. If
// file doesn't exist, it returns nil, if file exists but is invalid or can't
// be opened, it returns an error. The function doesn't fill the default values
// of the config.
func (u *UserConfig) fillWithFileData(path string) error {
	if _, err := os.Stat(path); err == nil {
		file, err := ioutil.ReadFile(path)
		if err != nil {
			return WrapErrorf(err, fileReadError, path)
		}
		if err = json.Unmarshal(file, u); err != nil {
			return WrapErrorf(err, jsonUnmarshalError, path)
		}
	} else if !os.IsNotExist(err) {
		return WrapErrorf(err, osStatErrorAny, path)
	}
	return nil
}

func (u *UserConfig) dump(path string) error {
	// Save the configuration
	result, _ := json.MarshalIndent(u, "", "\t")
	parentDir := filepath.Dir(path)
	err := os.MkdirAll(parentDir, 0755)
	if err != nil {
		return WrapErrorf(err, osMkdirError, parentDir)
	}
	err = os.WriteFile(path, result, 0644)
	if err != nil {
		return WrapErrorf(err, fileWriteError, path)
	}
	return nil
}

// getGlobalUserConfigPath returns the path to the file where the global user
// config is stored.
func getGlobalUserConfigPath() (string, error) {
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
	result := NewUserConfig()
	defer result.fillDefaults()
	globalConfigPath, err := getGlobalUserConfigPath()
	if err != nil {
		return nil, WrapError(err, getGlobalUserConfigPathError)
	}
	localConfigPath, err := filepath.Abs(localUserConfigPath)
	if err != nil {
		return nil, WrapErrorf(err, filepathAbsError, localUserConfigPath)
	}
	// Load the config files
	// First load the global config
	err = result.fillWithFileData(globalConfigPath)
	if err != nil {
		return nil, WrapError(err, "Failed to read global user_config.json")
	}
	// Overwrite with the local config
	err = result.fillWithFileData(localConfigPath)
	if err != nil {
		return nil, WrapError(err, "Failed to read local user_config.json")
	}
	return result, nil
}

// getUserConfig lazily loads the user config to the global userConfig variable
// and returns it. The pointer returned from this function is guaranteed to be
// non-nil.
func getUserConfig() (*UserConfig, error) {
	if cachedUserConfig == nil {
		readUserConfig, err := readCombinedUserConfig()
		if err != nil {
			return nil, PassError(err)
		}
		cachedUserConfig = readUserConfig
	}
	return cachedUserConfig, nil
}
