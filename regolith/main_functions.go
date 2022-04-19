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
	if len(filters) == 0 {
		return WrappedError("No filters specified.")
	}
	Logger.Info("Installing filters...")
	if !hasGit() {
		Logger.Warn(gitNotInstalled)
	}
	for _, filter := range filters {
		if err := addFilter(filter, force); err != nil {
			return WrapErrorf(err, "Failed to install filter %q.", filter)
		}
	}
	Logger.Info("Successfully installed the filters.")
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
	Logger.Info("Installing filters...")
	if !hasGit() {
		Logger.Warn(gitNotInstalled)
	}
	configJson, err := LoadConfigAsMap()
	if err != nil {
		return WrapError(err, "Failed to load config.json.")
	}
	config, err := ConfigFromObject(configJson)

	if err != nil {
		return WrapError(err, "Failed to parse \"config.json\" file.")
	}
	err = config.InstallFilters(force)
	if err != nil {
		return WrapError(err, "Could not install filters.")
	}
	Logger.Info("Successfully installed the filters.")
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
	if len(filters) == 0 {
		return WrappedError("No filters specified.")
	}
	Logger.Info("Updating filters...")
	if !hasGit() {
		Logger.Warn(gitNotInstalled)
	}
	configMap, err1 := LoadConfigAsMap()
	config, err2 := ConfigFromObject(configMap)
	if err := firstErr(err1, err2); err != nil {
		return WrapError(err, "Failed to load config.json.")
	}
	for _, filterName := range filters {
		filterDefinition, ok := config.FilterDefinitions[filterName]
		if !ok {
			Logger.Warnf(
				"Filter %q is not installed and therefore cannot be updated.",
				filterName)
			continue
		}
		remoteFilter, ok := filterDefinition.(*RemoteFilterDefinition)
		if !ok {
			Logger.Warnf(
				"Filter %q is not a remote filter and therefore cannot be updated.",
				filterName)
			continue
		}
		if err := remoteFilter.Update(); err != nil {
			Logger.Error(
				WrapErrorf(err, "Failed to update filter %q.", filterName))
		}
	}
	Logger.Info("Successfully updated the filters.")
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
	Logger.Info("Updating filters...")
	if !hasGit() {
		Logger.Warn(gitNotInstalled)
	}
	Logger.Infof("Updating filters...")
	configMap, err1 := LoadConfigAsMap()
	config, err2 := ConfigFromObject(configMap)
	if err := firstErr(err1, err2); err != nil {
		return WrapError(err, "Failed to load config.json.")
	}
	for filterName, filterDefinition := range config.FilterDefinitions {
		remoteFilter, ok := filterDefinition.(*RemoteFilterDefinition)
		if !ok { // Skip updating non-remote filters.
			continue
		}
		if err := remoteFilter.Update(); err != nil {
			Logger.Error(
				WrapErrorf(
					err, "Failed to update filter %q.", filterName))
		}
	}
	Logger.Info("Successfully updated the filters.")
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
	Logger.Infof("Running %q profile...", profile)
	err := RunProfile(profile)
	if err != nil {
		return WrapErrorf(err, "Failed to run profile %q", profile)
	}
	Logger.Infof("Successfully ran the %q profile.", profile)
	return nil
}

// Init handles the "regolith init" command. It initializes a new Regolith
// project in the current directory.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func Init(debug bool) error {
	InitLogging(debug)
	Logger.Info("Initializing Regolith project...")

	wd, err := os.Getwd()
	if err != nil {
		return WrapError(
			err, "Unable to get working directory to initialize project.")
	}
	if isEmpty, err := isDirEmpty(wd); err != nil {
		return WrapErrorf(
			err, "Failed to check if %s is an empty directory.", wd)
	} else if !isEmpty {
		return WrappedErrorf(
			"Cannot initialze the project, because %s is not an empty "+
				"directory.\n\"regolith init\" can be used only in empty "+
				"directories.", wd)
	}
	ioutil.WriteFile(".gitignore", []byte(GitIgnore), 0666)
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
	err = ioutil.WriteFile(ConfigFilePath, jsonBytes, 0666)
	if err != nil {
		return WrapErrorf(err, "Failed to write data to %q", ConfigFilePath)
	}

	for _, folder := range ConfigurationFolders {
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
func Clean(debug bool, cachedStatesOnly bool) error {
	InitLogging(debug)
	Logger.Infof("Cleaning cache...")
	if cachedStatesOnly {
		err := ClearCachedStates()
		if err != nil {
			return WrapError(err, "Failed to remove cached path states.")
		}
	} else {
		err := os.RemoveAll(".regolith")
		if err != nil {
			return WrapError(err, "failed to remove .regolith folder")
		}
		err = os.Mkdir(".regolith", 0666)
		if err != nil {
			return WrapError(err, "failed to recreate .regolith folder")
		}
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
	Logger.Info("Disabling the safe mode...")
	configMap, err1 := LoadConfigAsMap()
	_, err2 := ConfigFromObject(configMap)
	if err := firstErr(err1, err2); err != nil {
		return WrapError(
			err,
			"This does not appear to be a Regolith project.\nRegolith was "+
				"unable to load the \"config.json\" file.\nEvery regolith"+
				"project requires a valid config file.")
	}

	id, err := GetMachineId()
	if err != nil {
		return WrappedError("Failed to get machine ID for the lock file.")
	}

	lockfilePath := ".regolith/cache/lockfile.txt"
	Logger.Infof("Creating the lock file in %s...", lockfilePath)
	if _, err := os.Stat(lockfilePath); err == nil {
		return WrappedErrorf(
			"Failed to create the lock file because it already exists.\n"+
				"Please remove the file manually.\n"+
				"The lock file is located at %q.", lockfilePath)
	}
	err = ioutil.WriteFile(lockfilePath, []byte(id), 0666)
	if err != nil {
		return WrapError(err, "Failed to write lock file.")
	}
	Logger.Infof("Safe mode disabled.")
	return nil
}
