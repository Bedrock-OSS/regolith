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
	if len(filters) == 0 {
		return WrappedError("No filters specified.")
	}
	Logger.Info("Installing filters...")
	if !hasGit() {
		Logger.Warn(gitNotInstalled)
	}
	// resolverUpdated is a boolean that determines if the resovler map file
	// was downloaded during the execution of this function.
	resolverUpdated := false
	// resolvedArgs is a map with parsed argumetns, the keys are the filter
	// url and name (as this kind of repetition is not allowed). This map is
	// used to search for duplicates.
	parsedArgs := make(map[[2]string]*parsedInstallFilterArg)
	// Parse the args
	for _, rawFilterArg := range filters {
		parsedArg, err := parseInstallFilterArg(
			rawFilterArg,
			!resolverUpdated) // download resolver only once in a loop
		if err != nil {
			return WrapErrorf(
				err, "Unable to parse filter name and version from %q.",
				rawFilterArg)
		}
		if parsedArg.usedResolver {
			resolverUpdated = true
		}
		key := [2]string{parsedArg.url, parsedArg.name}
		if parsedArgs[key] != nil {
			return WrapErrorf(
				err, "Duplicate filter:\n URL: %s\n name: %s",
				parsedArg.url, parsedArg.name)
		}
		parsedArgs[key] = parsedArg

	}
	// TODO - Possible async download here, I'm not sure if it would actually
	// be faster unless you provide a lot of filters at once. Modifying
	// config.json must be reomved from addFilter before adding that feature.
	for _, parsedArg := range parsedArgs {
		if err := addFilter(*parsedArg, force); err != nil {
			return WrapErrorf(
				err, "Failed to install filter %q.", parsedArg.raw)
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
	resolverUpdated := false
	for _, filterName := range filters {
		filterDefinition, ok := config.FilterDefinitions[filterName]
		if !ok {
			Logger.Warnf(
				"Filter %q is not installed and therefore cannot be updated.",
				filterName)
			continue
		}
		// Only remote filters require resolver, and we only need to download
		// it once
		if _, ok := filterDefinition.(*RemoteFilterDefinition); ok && !resolverUpdated {
			err := DownloadResolverMap()
			if err != nil {
				Logger.Warn("Failed to download resolver map.")
			}
			resolverUpdated = true
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
	resolverUpdated := false
	for filterName, filterDefinition := range config.FilterDefinitions {
		remoteFilter, ok := filterDefinition.(*RemoteFilterDefinition)
		if !ok { // Skip updating non-remote filters.
			continue
		} else if !resolverUpdated {
			err := DownloadResolverMap()
			if err != nil {
				Logger.Warn("Failed to download resolver map.")
			}
			resolverUpdated = true
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
	err = CheckProfileImpl(profile, profileName, *config, nil)
	if err != nil {
		return err
	}
	path, _ := filepath.Abs(".")
	context := RunContext{
		AbsoluteLocation: path,
		Config:           config,
		Parent:           nil,
		Profile:          profileName,
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
