// Functions related to user_config.json
package regolith

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	// cachedCombinedUserConfig is a global variable that stores the combined user config. It
	// should not be accessed directly, but instead through the getUserConfig
	// function, which will lazily load the config if it hasn't been loaded yet.
	cachedCombinedUserConfig *UserConfig

	// cachedLocalUserConfig is a global variable that stores the local user config. It
	// should not be accessed directly, but instead through the getUserConfig
	// function, which will lazily load the config if it hasn't been loaded yet.
	cachedLocalUserConfig *UserConfig

	// cachedGlobalUserConfig is a global variable that stores the global user config. It
	// should not be accessed directly, but instead through the getUserConfig
	// function, which will lazily load the config if it hasn't been loaded yet.
	cachedGlobalUserConfig *UserConfig
)

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

// loadUserConfigs reads the config from .regolith/user_config.json and
// from the user app data directory and sets the global variables
// cachedCombinedUserConfig, cachedLocalUserConfig and cachedGlobalUserConfig.
func loadUserConfigs() error {
	// Get the paths to localConfigPath and appDataConfig
	cachedGlobalUserConfig = NewUserConfig()
	cachedLocalUserConfig = NewUserConfig()
	cachedCombinedUserConfig = NewUserConfig()
	defer cachedCombinedUserConfig.fillDefaults()

	globalConfigPath, err := getGlobalUserConfigPath()
	if err != nil {
		return WrapError(err, getGlobalUserConfigPathError)
	}
	// Load the config files
	// First load the global config
	err1 := cachedGlobalUserConfig.fillWithFileData(globalConfigPath)
	err2 := cachedCombinedUserConfig.fillWithFileData(globalConfigPath)
	if err = firstErr(err1, err2); err != nil {
		return WrapError(err, "Failed to read global user_config.json")
	}
	return nil
}

// getCombinedUserConfig lazily loads the user config to the global
// cachedCombinedUserConfig variable and returns it.
func getCombinedUserConfig() (*UserConfig, error) {
	if cachedCombinedUserConfig == nil {
		err := loadUserConfigs()
		if err != nil {
			return nil, PassError(err)
		}
	}
	return cachedCombinedUserConfig, nil
}

// getGlobalUserConfig lazily loads the user config to the global
// cachedGlobalUserConfig variable and returns it.
func getGlobalUserConfig() (*UserConfig, error) {
	if cachedGlobalUserConfig == nil {
		err := loadUserConfigs()
		if err != nil {
			return nil, PassError(err)
		}
	}
	return cachedGlobalUserConfig, nil
}
