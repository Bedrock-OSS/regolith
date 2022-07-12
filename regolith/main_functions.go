package regolith

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
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
	Logger.Info("Installing filters...")
	if !hasGit() {
		Logger.Warn(gitNotInstalled)
	}
	// Parse arguments into download tasks
	parsedArgs, err := parseInstallFilterArgs(filters)
	if err != nil {
		return WrapError(err, "Failed to parse arguments.")
	}
	config, err := LoadConfigAsMap()
	if err != nil {
		return WrapError(err, "Unable to load config file.")
	}
	// Get parts of config file required for installation
	dataPath, err := dataPathFromConfigMap(config)
	if err != nil {
		return WrapError(err, "Failed to get data path from config file.")
	}
	filterDefinitions, err := filterDefinitionsFromConfigMap(config)
	if err != nil {
		return WrapError(
			err, "Failed to get filter definitions from config file.")
	}
	// Check if the filters are already installed if force mode is disabled
	if !force {
		for _, parsedArg := range parsedArgs {
			_, ok := filterDefinitions[parsedArg.name]
			if ok {
				return WrappedErrorf(
					"The filter %q is already on the filter definitions list.\n"+
						"Please remove it first before installing it again or use "+
						"the --force option.", parsedArg.name)
			}
		}
	}
	// Convert to filter definitions for download
	filterInstallers := make(map[string]FilterInstaller, 0)
	for _, parsedArg := range parsedArgs {
		// Get the filter definition from the Internet
		remoteFilterDefinition, err := FilterDefinitionFromTheInternet(
			parsedArg.url, parsedArg.name, parsedArg.version)
		if err != nil {
			return WrapErrorf(
				err, "Unable to get filter definition of %q filter.",
				parsedArg.name)
		}
		if parsedArg.version == "HEAD" || parsedArg.version == "latest" {
			// The "HEAD" and "latest" keywords should be the same in the
			// config file don't lock them to the actual versions
			remoteFilterDefinition.Version = parsedArg.version
		}
		if err != nil {
			return WrapErrorf(
				err,
				"Unable to download filter: \"%s\"\n"+
					"Some of the files might still be in cache.\n"+
					"Run \"regolith clean\" to fix invalid cache state.",
				parsedArg.name)
		}
		filterInstallers[parsedArg.name] = remoteFilterDefinition
	}
	// Download the filter definitions
	err = installFilters(filterInstallers, force, dataPath, ".regolith")
	if err != nil {
		return WrapError(err, "Failed to install filters.")
	}
	// Add the filters to the config
	for name, downloadedFilter := range filterInstallers {
		// Add the filter to config file
		filterDefinitions[name] = downloadedFilter
	}
	// Save the config file
	jsonBytes, _ := json.MarshalIndent(config, "", "  ")
	err = ioutil.WriteFile(ConfigFilePath, jsonBytes, 0666)
	if err != nil {
		return WrapErrorf(
			err,
			"Successfully downloaded %v filters"+
				"but failed to update the config file.\n"+
				"Run \"regolith clean\" to fix invalid cache state.",
			len(parsedArgs))
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
	configMap, err1 := LoadConfigAsMap()
	config, err2 := ConfigFromObject(configMap)
	if err := firstErr(err1, err2); err != nil {
		return WrapError(err, "Failed to load config.json.")
	}
	err := installFilters(config.FilterDefinitions, force, config.DataPath, ".regolith")
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
	Logger.Info("Updating filters...")
	if !hasGit() {
		Logger.Warn(gitNotInstalled)
	}
	if len(filters) == 0 {
		return WrappedError("No filters specified.")
	}
	configMap, err1 := LoadConfigAsMap()
	config, err2 := ConfigFromObject(configMap)
	if err := firstErr(err1, err2); err != nil {
		return WrapError(err, "Failed to load config.json.")
	}
	// Filter out the filters that are not present in the 'filters' list
	filterInstallers := make(map[string]FilterInstaller, 0)
	for _, filterName := range filters {
		filterInstaller, ok := config.FilterDefinitions[filterName]
		if !ok {
			Logger.Warnf(
				"Filter %q is not installed and therefore cannot be updated.",
				filterName)
			continue
		}
		filterInstallers[filterName] = filterInstaller
	}
	// Update the filters from the list
	err := updateFilters(filterInstallers, ".regolith")
	if err != nil {
		return WrapError(err, "Could not update filters.")
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
	configMap, err1 := LoadConfigAsMap()
	config, err2 := ConfigFromObject(configMap)
	if err := firstErr(err1, err2); err != nil {
		return WrapError(err, "Failed to load config.json.")
	}
	err := updateFilters(config.FilterDefinitions, ".regolith")
	if err != nil {
		return WrapError(err, "Could not install filters.")
	}
	Logger.Info("Successfully installed the filters.")
	return nil
}

// runOrWatch handles both 'regolith run' and 'regolith watch' commands based
// on the 'watch' parameter. It runs/watches the profile named after
// 'profileName' parameter. The 'debug' argument determines if the debug
// messages should be printed or not.
func runOrWatch(profileName string, recycled, debug, watch bool) error {
	InitLogging(debug)
	// Select the run profile function based on the recycled flag
	rp := RunProfile
	if recycled {
		rp = RecycledRunProfile
	}
	if profileName == "" {
		profileName = "default"
	}
	// Load the Config and the profile
	configJson, err := LoadConfigAsMap()
	if err != nil {
		return WrapError(err, "Could not load \"config.json\".")
	}
	config, err := ConfigFromObject(configJson)
	if err != nil {
		return WrapError(err, "Could not load \"config.json\".")
	}
	profile, ok := config.Profiles[profileName]
	if !ok {
		return WrappedErrorf(
			"Profile %q does not exist in the configuration.", profileName)
	}
	// Check the filters of the profile
	err = CheckProfileImpl(profile, profileName, *config, nil, ".regolith")
	if err != nil {
		return err
	}
	path, _ := filepath.Abs(".")
	context := RunContext{
		AbsoluteLocation: path,
		Config:           config,
		Parent:           nil,
		Profile:          profileName,
		DotRegolithPath:  ".regolith",
	}
	if watch { // Loop until program termination (CTRL+C)
		context.StartWatchingSrouceFiles()
		for {
			err = rp(context)
			if err != nil {
				Logger.Errorf(
					"Failed to run profile %q: %s",
					profileName, PassError(err).Error())
			} else {
				Logger.Infof("Successfully ran the %q profile.", profileName)
			}
			Logger.Info("Press Ctrl+C to stop watching.")
			context.AwaitInterruption()
			Logger.Warn("Restarting...")
		}
		// return nil // Unreachable code
	}
	err = rp(context)
	if err != nil {
		return WrapErrorf(err, "Failed to run profile %q", profileName)
	}
	Logger.Infof("Successfully ran the %q profile.", profileName)
	return nil
}

// Run handles the "regolith run" command. It runs selected profile and exports
// created resource pack and behvaiour pack to the target destination.
func Run(profileName string, recycled, debug bool) error {
	return runOrWatch(profileName, recycled, debug, false)
}

// Watch handles the "regolith watch" command. It watches the project
// directories and it runs selected profile and exports created resource pack
// and behvaiour pack to the target destination when the project changes.
func Watch(profileName string, recycled, debug bool) error {
	return runOrWatch(profileName, recycled, debug, true)
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
				"default": {
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
	var ConfigurationFolders = []string{
		"packs",
		"packs/data",
		"packs/BP",
		"packs/RP",
		filepath.Join(".regolith", "cache/venvs"),
	}
	for _, folder := range ConfigurationFolders {
		err = os.MkdirAll(folder, 0666)
		if err != nil {
			Logger.Error("Could not create folder: %s", folder, err)
		}
	}

	Logger.Info("Regolith project initialized.")
	return nil
}

// Clean handles the "regolith clean" command. It cleans the cache from the
// dotRegolithPath directory.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func Clean(debug bool, cachedStatesOnly bool) error {
	InitLogging(debug)
	Logger.Infof("Cleaning cache...")
	dotRegolithPath := ".regolith"
	if cachedStatesOnly {
		err := ClearCachedStates()
		if err != nil {
			return WrapError(err, "Failed to remove cached path states.")
		}
	} else {
		err := os.RemoveAll(dotRegolithPath)
		if err != nil {
			return WrapErrorf(err, "failed to remove %q folder", dotRegolithPath)
		}
		err = os.MkdirAll(dotRegolithPath, 0666)
		if err != nil {
			return WrapErrorf(err, "failed to recreate %q folder", dotRegolithPath)
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

	lockfilePath := filepath.Join(".regolith", "cache/lockfile.txt")
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
