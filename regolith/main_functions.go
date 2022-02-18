package regolith

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

// Install handles the "regolith install" command. It installs specific filters
// from the Internet and adds them to the filtersDefinitions list in the
// config.json file.
//
// The "filters" parameter is a list of filters to install in the format
// <filter-url>==<filter-version> or <filter-url>.
// "filter-url" is the URL of the filter to install.
// "filter-version" is the version of the filter. It can be semver, git commit
//  hash, "HEAD", or "latest". "HEAD" means that the filter will be
// updated to lastest SHA commit and "latest" updates the filter to the latest
// version tag. If "filter-version" is not specified, the filter will be
// installed with the latest version or HEAD if there is no valid version tags.
//
// The "force" parameter is a boolean that determines if the installation
// should be forced even if the filter is already installed.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func Install(filters []string, force, debug bool) error {
	InitLogging(debug)
	for _, filter := range filters {
		if err := addFilter(filter, force); err != nil {
			return WrapErrorf(err, "Failed to install filter %q.", filter)
		}
	}
	return nil
}

// InstallAll handles the "regolith install-all" command. It installs all of
// filters and their dependencies from the filtersDefinitions list in the
// config.json file.
//
// The "force" parameter is a boolean that determines if the installation
// should be forced even if the filter is already installed.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func InstallAll(force, debug bool) error {
	InitLogging(debug)
	configJson, err := LoadConfigAsMap()
	if err != nil {
		return WrapError(err, "Failed to load config.json.")
	}
	config, err := ConfigFromObject(configJson)

	if err != nil {
		return WrapError(err, "Failed to parse 'config.json' file.")
	}
	err = config.InstallFilters(force)
	if err != nil {
		return WrapError(err, "Could not install filters.")
	}
	return nil
}

// Update handles the "regolith update" command. It updates filters listed in
// "filters" parameter. The names of the filters must be already present in the
// filtersDefinitions list in the config.json file.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func Update(filters []string, debug bool) error {
	InitLogging(debug)
	// TODO - implement
	return nil
}

// UpdateAll handles the "regolith update-all" command. It updates all of the
// filters from the filtersDefinitions list in the config.json file which
// aren't version locked.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func UpdateAll(debug bool) error {
	InitLogging(debug)
	// TODO - implement
	return nil
}

// Run handles the "regolith run" command. It runs selected profile and exports
// created resource pack and behvaiour pack to the target destination.
//
// The "profile" parameter is the name of the profile to run. If the profile
// is an empty string, the "dev" profile will be used.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func Run(profile string, debug bool) error {
	InitLogging(debug)
	if profile == "" {
		profile = "dev"
	}
	err := RunProfile(profile)
	if err != nil {
		return WrapErrorf(err, "Failed to run profile %q", profile)
	}
	return nil
}

// Init handles the "regolith init" command. It initializes a new Regolith
// project in the current directory.
//
// The "force" parameter is a boolean that determines if the initialization
// overriding potential safeguards.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func Init(force, debug bool) error {
	InitLogging(debug)
	Logger.Info("Initializing Regolith project...")
	if !force && IsProjectInitialized() {
		return WrapErrorf(
			nil,
			"%q already exists, suggesting this project is already initialized. You may use --force to override this check.",
			ConfigFilePath)
	}
	if force {
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

// Clean handles the "regolith clean" command. It cleans the cache from the
// ".regolith" directory.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func Clean(debug bool) error {
	InitLogging(debug)
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

// Unlock handles the "regolith unlock". It unlocks safe mode, by signing the
// machine ID into lockfile.txt.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func Unlock(debug bool) error {
	InitLogging(debug)
	if !IsProjectInitialized() {
		return WrapError(nil, "this does not appear to be a Regolith project")
	}

	id, err := GetMachineId()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(".regolith/cache/lockfile.txt", []byte(id), 0666)
	if err != nil {
		return WrapError(err, "Failed to write lock file.")
	}

	return nil
}
