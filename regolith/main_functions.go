package regolith

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Bedrock-OSS/go-burrito/burrito"
)

var disallowedFiles = []string{
	"config.json",
	"packs",
	".regolith",
	".gitignore",
}

// Install handles the "regolith install" command. It installs specific filters
// from the internet and adds them to the filtersDefinitions list in the
// config.json file.
//
// The "filters" parameter is a list of filters to install in the format
// <filter-url>==<filter-version> or <filter-url>.
// "filter-url" is the URL of the filter to install.
// "filter-version" is the version of the filter. It can be semver, git commit
// hash, "HEAD", or "latest". "HEAD" means that the filter will be
// updated to the latest SHA commit and "latest" updates the filter to the latest
// version tag. If "filter-version" is not specified, the filter will be
// installed with the latest version or HEAD if there is no valid version tags.
//
// The "force" parameter is a boolean that determines if the installation
// should be forced even if the filter is already installed.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func Install(filters []string, force, refreshResolvers, refreshFilters bool, profiles []string, debug bool) error {
	InitLogging(debug)
	Logger.Info("Installing filters...")
	if !hasGit() {
		Logger.Warn(gitNotInstalledWarning)
	}
	config, err := LoadConfigAsMap()
	if err != nil {
		return burrito.WrapError(err, "Unable to load config file.")
	}
	// Check if selected profiles exist
	for _, profile := range profiles {
		_, err := FindByJSONPath[map[string]interface{}](config, "regolith/profiles/"+EscapePathPart(profile))
		if err != nil {
			return burrito.WrapErrorf(
				err, "Profile %s does not exist or is invalid.", profile)
		}
	}
	// Get parts of config file required for installation
	dataPath, err := dataPathFromConfigMap(config)
	if err != nil {
		return burrito.WrapError(err, "Failed to get data path from config file.")
	}
	filterDefinitions, err := filterDefinitionsFromConfigMap(config)
	if err != nil {
		return burrito.WrapError(
			err,
			"Failed to get the list of filter definitions from config file.")
	}
	// Get dotRegolithPath
	dotRegolithPath, err := GetDotRegolith(".")
	if err != nil {
		return burrito.WrapError(
			err, "Unable to get the path to regolith cache folder.")
	}
	// Lock the session
	unlockSession, sessionLockErr := acquireSessionLock(dotRegolithPath)
	if sessionLockErr != nil {
		return burrito.WrapError(sessionLockErr, acquireSessionLockError)
	}
	defer func() { sessionLockErr = unlockSession() }()
	// Parse arguments into download tasks (requires downloading resolvers)
	parsedArgs, err := parseInstallFilterArgs(filters, refreshResolvers)
	if err != nil {
		return burrito.WrapError(err, "Failed to parse arguments.")
	}
	// Check if the filters are already installed if force mode is disabled
	if !force {
		for _, parsedArg := range parsedArgs {
			_, ok := filterDefinitions[parsedArg.name]
			if ok {
				return burrito.WrappedErrorf(
					"The filter is already on the filter definitions list.\n"+
						"Filter: %s\n"+
						"If you want to force the installation of the filter, "+
						"please add \"--update\" flag to your "+
						"\"regolith install\" command", parsedArg.name)
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
			return burrito.WrapErrorf(
				err,
				"Unable to download the filter definition from the internet.\n"+
					"Filter repository Url: %s\n"+
					"Filter name: %s\n"+
					"Filter version: %s\n",
				parsedArg.url, parsedArg.name, parsedArg.version)
		}
		if parsedArg.version == "HEAD" || parsedArg.version == "latest" {
			// The "HEAD" and "latest" keywords should be the same in the
			// config file don't lock them to the actual versions
			remoteFilterDefinition.Version = parsedArg.version
		}
		filterInstallers[parsedArg.name] = remoteFilterDefinition
	}
	// Download the filter definitions
	err = installFilters(
		filterInstallers, force, dataPath, dotRegolithPath, refreshFilters)
	if err != nil {
		return burrito.WrapError(err, "Failed to install filters.")
	}
	// Add the filters to the config
	for name, downloadedFilter := range filterInstallers {
		// Add the filter to config file
		filterDefinitions[name] = downloadedFilter
		// Add the filter to the profile
		for _, profile := range profiles {
			profileMap, err := FindByJSONPath[map[string]interface{}](config, "regolith/profiles/"+EscapePathPart(profile))
			// This check here is not necessary, because we have identical one at the beginning, but better to be safe
			if err != nil {
				return burrito.WrapErrorf(
					err, "Profile %s does not exist or is invalid.", profile)
			}
			if profileMap["filters"] == nil {
				profileMap["filters"] = make([]interface{}, 0)
			}
			// Add the filter to the profile
			profileMap["filters"] = append(
				profileMap["filters"].([]interface{}), map[string]interface{}{
					"filter": name,
				})
		}
	}
	// Save the config file
	jsonBytes, _ := json.MarshalIndent(config, "", "\t")
	err = os.WriteFile(ConfigFilePath, jsonBytes, 0644)
	if err != nil {
		return burrito.WrapErrorf(
			err,
			"Successfully downloaded %v filters"+
				"but failed to update the config file.\n"+
				"Run \"regolith clean\" to fix invalid cache state.",
			len(parsedArgs))
	}
	Logger.Info("Successfully installed the filters.")
	return sessionLockErr // Return the error from the defer function
}

// InstallAll handles the "regolith install-all" command. It installs all of
// the filters and their dependencies from the filtersDefinitions list in the
// config.json file.
//
// The "force" parameter is a boolean that determines if the installation
// should be forced even if the filter is already installed.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func InstallAll(force, debug, refreshFilters bool) error {
	InitLogging(debug)
	Logger.Info("Installing filters...")
	if !hasGit() {
		Logger.Warn(gitNotInstalledWarning)
	}
	configMap, err1 := LoadConfigAsMap()
	config, err2 := ConfigFromObject(configMap)
	if err := firstErr(err1, err2); err != nil {
		return burrito.WrapError(err, "Failed to load config.json.")
	}
	// Get dotRegolithPath
	dotRegolithPath, err := GetDotRegolith(".")
	if err != nil {
		return burrito.WrapError(
			err, "Unable to get the path to regolith cache folder.")
	}
	// Lock the session
	unlockSession, sessionLockErr := acquireSessionLock(dotRegolithPath)
	if sessionLockErr != nil {
		return burrito.WrapError(sessionLockErr, acquireSessionLockError)
	}
	defer func() { sessionLockErr = unlockSession() }()
	// Install the filters
	err = installFilters(
		config.FilterDefinitions, force, config.DataPath, dotRegolithPath, refreshFilters)
	if err != nil {
		return burrito.WrapError(err, "Could not install filters.")
	}
	Logger.Info("Successfully installed the filters.")
	return sessionLockErr // Return the error from the defer function
}

// prepareRunContext prepares the context for the "regolith run" and
// "regolith watch" commands.
func prepareRunContext(profileName string, debug, watch bool) (*RunContext, error) {
	InitLogging(debug)
	if profileName == "" {
		profileName = "default"
	}
	// Load the Config and the profile
	configJson, err := LoadConfigAsMap()
	if err != nil {
		return nil, burrito.WrapError(err, "Could not load \"config.json\".")
	}
	config, err := ConfigFromObject(configJson)
	if err != nil {
		return nil, burrito.WrapError(err, "Could not load \"config.json\".")
	}
	profile, ok := config.Profiles[profileName]
	if !ok {
		return nil, burrito.WrappedErrorf(
			"Profile %q does not exist in the configuration.", profileName)
	}
	// Get dotRegolithPath
	dotRegolithPath, err := GetDotRegolith(".")
	if err != nil {
		return nil, burrito.WrapError(
			err, "Unable to get the path to regolith cache folder.")
	}
	err = os.MkdirAll(dotRegolithPath, 0755)
	if err != nil {
		return nil, burrito.WrapErrorf(err, osMkdirError, dotRegolithPath)
	}
	// Check the filters of the profile
	err = CheckProfileImpl(profile, profileName, *config, nil, dotRegolithPath)
	if err != nil {
		return nil, err
	}
	path, _ := filepath.Abs(".")
	return &RunContext{
		AbsoluteLocation: path,
		Config:           config,
		Parent:           nil,
		Profile:          profileName,
		DotRegolithPath:  dotRegolithPath,
		Settings:         map[string]interface{}{},
	}, nil
}

// Run handles the "regolith run" command. It runs selected profile and exports
// created resource pack and behavior pack to the target destination.
func Run(profileName string, debug bool) error {
	// Get the context
	context, err := prepareRunContext(profileName, debug, false)
	if err != nil {
		return burrito.PassError(err)
	}
	// Lock the session
	unlockSession, sessionLockErr := acquireSessionLock(context.DotRegolithPath)
	if sessionLockErr != nil {
		return burrito.WrapError(sessionLockErr, acquireSessionLockError)
	}
	defer func() { sessionLockErr = unlockSession() }()
	// Run the profile
	err = RunProfile(*context)
	if err != nil {
		return burrito.WrapErrorf(err, "Failed to run profile %q", profileName)
	}
	Logger.Infof("Successfully ran the %q profile.", profileName)
	return sessionLockErr // Return the error from the defer function
}

// Watch handles the "regolith watch" command. It watches the project
// directories, and it runs selected profile and exports created resource pack
// and behavior pack to the target destination when the project changes.
func Watch(profileName string, debug bool) error {
	// Get the context
	context, err := prepareRunContext(profileName, debug, false)
	if err != nil {
		return burrito.PassError(err)
	}
	// Lock the session
	unlockSession, sessionLockErr := acquireSessionLock(context.DotRegolithPath)
	if sessionLockErr != nil {
		return burrito.WrapError(sessionLockErr, acquireSessionLockError)
	}
	defer func() { sessionLockErr = unlockSession() }()
	// Setup the channel for stopping the watching
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run the profile
	context.StartWatchingSourceFiles()
	for { // Loop until program termination (CTRL+C)
		err = RunProfile(*context)
		if err != nil {
			Logger.Errorf(
				"Failed to run profile %q: %s",
				profileName, burrito.PassError(err).Error())
		} else {
			Logger.Infof("Successfully ran the %q profile.", profileName)
		}
		Logger.Info("Press Ctrl+C to stop watching.")
		select {
		case <-context.interruptionChannel:
			// AwaitInterruption locks the goroutine with the interruption channel until
			// the Config is interrupted and returns the interruption message.
			Logger.Warn("Restarting...")
		case <-sigChan:
			return sessionLockErr // Return the error from the defer function
		}
	}
}

// ApplyFilter handles the "regolith apply-filter" command.
// ApplyFilter mode modifies RP and BP file in place (using source). The config and
// properties of the filter are passed via commandline.
func ApplyFilter(filterName string, filterArgs []string, debug bool) error {
	InitLogging(debug)
	// Load the Config and the profile
	configJson, err := LoadConfigAsMap()
	if err != nil {
		return burrito.WrapError(err, "Could not load \"config.json\".")
	}
	config, err := ConfigFromObject(configJson)
	if err != nil {
		return burrito.WrapError(err, "Could not load \"config.json\".")
	}
	filterDefinition, ok := config.FilterDefinitions[filterName]
	if !ok {
		return burrito.WrappedErrorf(
			"Unable to find the filter on the \"filterDefinitions\" list "+
				"of the \"config.json\" file.\n"+
				"Filter name: %s", filterName)
	}
	// Get dotRegolithPath
	dotRegolithPath, err := GetDotRegolith(".")
	if err != nil {
		return burrito.WrapError(
			err, "Unable to get the path to regolith cache folder.")
	}
	err = os.MkdirAll(dotRegolithPath, 0755)
	if err != nil {
		return burrito.WrapErrorf(err, osMkdirError, dotRegolithPath)
	}
	// Lock the session
	unlockSession, sessionLockErr := acquireSessionLock(dotRegolithPath)
	if sessionLockErr != nil {
		return burrito.WrapError(sessionLockErr, acquireSessionLockError)
	}
	defer func() {
		// WARNING: sessionLockError is not reported in case of different errors.
		// This error is minor and other errors are way more important.
		sessionLockErr = unlockSession()
	}()

	// Create the filter
	runConfiguration := map[string]interface{}{
		"filter":    filterName,
		"arguments": filterArgs,
	}
	filterRunner, err := filterDefinition.CreateFilterRunner(runConfiguration)
	if err != nil {
		return burrito.WrapErrorf(err, createFilterRunnerError, filterName)
	}
	// Create run context
	path, _ := filepath.Abs(".")
	runContext := RunContext{
		Config:              config,
		Parent:              nil,
		Profile:             "[dynamic profile]",
		DotRegolithPath:     dotRegolithPath,
		interruptionChannel: nil,
		AbsoluteLocation:    path,
		Settings:            filterRunner.GetSettings(),
	}
	// Check the filter
	err = filterRunner.Check(runContext)
	if err != nil {
		return burrito.WrapErrorf(err, filterRunnerCheckError, filterName)
	}
	// Setup tmp directory
	err = SetupTmpFiles(*config, dotRegolithPath)
	if err != nil {
		return burrito.WrapErrorf(err, setupTmpFilesError, dotRegolithPath)
	}
	// Run the filter
	Logger.Infof("Running the \"%s\" filter.", filterName)
	_, err = filterRunner.Run(runContext)
	if err != nil {
		return burrito.WrapErrorf(err, filterRunnerRunError, filterName)
	}
	// Export files to the source files
	Logger.Info("Overwriting the source files.")
	err = InplaceExportProject(config, dotRegolithPath)
	if err != nil {
		return burrito.WrapError(
			err, "Failed to overwrite the source files with generated files.")
	}
	Logger.Infof("Successfully ran the \"%s\" filter.", filterName)
	return sessionLockErr
}

// Init handles the "regolith init" command. It initializes a new Regolith
// project in the current directory.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func Init(debug, force bool) error {
	InitLogging(debug)
	Logger.Info("Initializing Regolith project...")

	wd, err := os.Getwd()
	if err != nil {
		return burrito.WrapError(
			err, osGetwdError)
	}
	if files, err := GetMatchingDirContents(wd, disallowedFiles); err != nil {
		return burrito.WrapErrorf(
			err, "Failed to check if %s is an empty directory.", wd)
	} else if len(files) > 0 && !force {
		return burrito.WrappedErrorf(
			"Cannot initialize the project, because %s contains files, that will be overwritten on init.\n"+
				"If you want to proceed, use --force flag\n"+
				"Disallowed files and directories found: %s", wd, strings.Join(files, ", "))
	}
	err = CheckSuspiciousLocation()
	if err != nil {
		return burrito.WrapError(err, projectSuspiciousDirError)
	}
	os.WriteFile(".gitignore", []byte(GitIgnore), 0644)
	// Create new default configuration
	userConfig, err := getCombinedUserConfig()
	if err != nil {
		return burrito.WrapError(err, getUserConfigError)
	}
	jsonData := Config{
		Name:   "Project name",
		Author: *userConfig.Username,
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
	jsonBytes, _ := json.MarshalIndent(jsonData, "", "")
	// Add the schema property, this is a little hacky
	rawJsonData := make(map[string]interface{}, 0)
	json.Unmarshal(jsonBytes, &rawJsonData)
	rawJsonData["$schema"] = "https://raw.githubusercontent.com/Bedrock-OSS/regolith-schemas/main/config/v1.2.json"
	jsonBytes, _ = json.MarshalIndent(rawJsonData, "", "\t")

	err = os.WriteFile(ConfigFilePath, jsonBytes, 0644)
	if err != nil {
		return burrito.WrapErrorf(err, "Failed to write data to %q", ConfigFilePath)
	}
	var ConfigurationFolders = []string{
		"packs",
		"packs/data",
		"packs/BP",
		"packs/RP",
		filepath.Join(".regolith", "cache/venvs"),
	}
	for _, folder := range ConfigurationFolders {
		err = os.MkdirAll(folder, 0755)
		if err != nil {
			Logger.Error("Could not create folder: %s", folder, err)
		}
	}

	Logger.Info("Regolith project initialized.")
	return nil
}

// Cleans the cache folder of regolith (.regolith in normal mode or a path in
// AppData). The path to clean is determined by the dotRegolithPath parameter.
// leaveEmptyPath determines if regolith should leave an empty folder at
// dotRegolithPath
func clean(dotRegolithPath string) error {
	err := os.RemoveAll(dotRegolithPath)
	if err != nil {
		return burrito.WrapErrorf(err, "failed to remove %q folder", dotRegolithPath)
	}
	return nil
}

func CleanCurrentProject() error {
	Logger.Infof("Cleaning cache...")

	// Clean .regolith
	Logger.Infof("Cleaning \".regolith\"...")
	err := clean(".regolith")
	if err != nil {
		return burrito.WrapErrorf(
			err, "Failed to clean the cache from \".regolith\".")
	}
	// Clean cache from AppData
	Logger.Infof("Cleaning the cache in application data folder...")
	dotRegolithPath, err := getAppDataDotRegolith(".")
	if err != nil {
		return burrito.WrapError(
			err, "Unable to get the path to regolith cache folder.")
	}
	err = clean(dotRegolithPath)
	if err != nil {
		return burrito.WrapErrorf(
			err, "Failed to clean the cache from %q.", dotRegolithPath)
	}
	Logger.Infof("Cache cleaned.")
	return nil
}

func CleanUserCache() error {
	Logger.Infof("Cleaning all Regolith cache files from user app data...")
	// App data enabled - use user cache dir
	userCache, err := os.UserCacheDir()
	if err != nil {
		return burrito.WrappedError(osUserCacheDirError)
	}
	regolithCacheFiles := filepath.Join(userCache, appDataProjectCachePath)
	Logger.Infof("Regolith cache files are located in: %s", regolithCacheFiles)
	err = os.RemoveAll(regolithCacheFiles)
	if err != nil {
		return burrito.WrapErrorf(err, "failed to remove %q folder", regolithCacheFiles)
	}
	os.MkdirAll(regolithCacheFiles, 0755)
	Logger.Infof("All regolith files cached in user app data cleaned.")
	return nil
}

func CleanFilterCache() error {
	Logger.Infof("Cleaning Regolith filter cache files from user app data...")
	// App data enabled - use user cache dir
	userCache, err := os.UserCacheDir()
	if err != nil {
		return burrito.WrappedError(osUserCacheDirError)
	}
	regolithCacheFiles := filepath.Join(userCache, appDataFilterCachePath)
	Logger.Infof("Regolith cache files are located in: %s", regolithCacheFiles)
	err = os.RemoveAll(regolithCacheFiles)
	if err != nil {
		return burrito.WrapErrorf(err, "failed to remove %q folder", regolithCacheFiles)
	}
	os.MkdirAll(regolithCacheFiles, 0755)
	Logger.Infof("Regolith filter files cached in user app data cleaned.")
	return nil
}

// Clean handles the "regolith clean" command. It cleans the cache from the
// dotRegolithPath directory.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func Clean(debug, userCache, filterCache bool) error {
	InitLogging(debug)
	if userCache {
		return CleanUserCache()
	} else if filterCache {
		return CleanFilterCache()
	} else {
		return CleanCurrentProject()
	}
}

// UpdateResolvers handles the "regolith update-resolvers" command. It updates cached resolver repositories.
//
// The "debug" parameter is a boolean that determines if the debug messages
// should be printed.
func UpdateResolvers(debug bool) error {
	InitLogging(debug)
	_, _, err := DownloadResolverMaps(true)
	return err
}

// manageUserConfigPrint is a helper function for ManageConfig used to print
// the specified value from the user configuration.
func manageUserConfigPrint(debug, full bool, key string) error {
	var err error // prevent shadowing
	configPath := ""
	userConfig := NewUserConfig()
	if full {
		userConfig, err = getCombinedUserConfig() // Combined config
		if err != nil {
			return burrito.WrapError(err, getUserConfigError)
		}
		fmt.Println("\nCOMBINED USER CONFIGURATION (CONFIG FILE + DEFAULTS):")
	} else { // Combined
		configPath, err = getGlobalUserConfigPath()
		if err != nil {
			return burrito.WrapError(err, getGlobalUserConfigPathError)
		}
		fmt.Printf("\nGLOBAL USER CONFIGURATION: %s\n", configPath)
		userConfig.fillWithFileData(configPath)
	}
	result, err := userConfig.stringPropertyValue(key)
	if err != nil {
		return burrito.WrapErrorf(err, invalidUserConfigPropertyError, key)
	}
	result = "\t" + strings.Replace(result, "\n", "\n\t", -1) // Indent
	fmt.Println(result)
	return nil
}

// manageUserConfigPrintAll is a helper function for ManageConfig used to print
// whole user configuration.
func manageUserConfigPrintAll(debug, full bool) error {
	var err error // prevent shadowing
	configPath := ""
	var userConfig *UserConfig
	if full {
		userConfig, err = getCombinedUserConfig() // Combined config
		if err != nil {
			return burrito.WrapError(err, getUserConfigError)
		}
		fmt.Println("\nCOMBINED USER CONFIGURATION (CONFIG FILE + DEFAULTS):")
	} else {
		configPath, err = getGlobalUserConfigPath()
		if err != nil {
			return burrito.WrapError(err, getGlobalUserConfigPathError)
		}
		fmt.Printf("\nUSER CONFIGURATION FROM FILE: %s\n", configPath)
		userConfig, err = getGlobalUserConfig()
		if err != nil {
			return burrito.WrapError(err, getUserConfigError)
		}
	}
	fmt.Println( // Print with additional indentation
		"\t" + strings.Replace(userConfig.String(), "\n", "\n\t", -1))
	return nil
}

// manageUserConfigEdit is a helper function for ManageConfig used to edit
// the specified value from the user configuration.
func manageUserConfigEdit(debug bool, index int, key, value string) error {
	configPath, err := getGlobalUserConfigPath()
	if err != nil {
		return burrito.WrapError(err, getGlobalUserConfigPathError)
	}
	Logger.Infof("Editing user configuration.\n\tPath: %s", configPath)
	userConfig := NewUserConfig()
	userConfig.fillWithFileData(configPath)
	switch key {
	case "use_project_app_data_storage":
		if index != -1 {
			return burrito.WrappedError("Cannot use --index with non-array property.")
		}
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return burrito.WrapErrorf(err, "Invalid value for boolean property.\n"+
				"\tValue: %s", value)
		}
		userConfig.UseProjectAppDataStorage = &boolValue
	case "username":
		if index != -1 {
			return burrito.WrappedError("Cannot use --index with non-array property.")
		}
		userConfig.Username = &value
	case "resolver_cache_update_cooldown":
		if index != -1 {
			return burrito.WrappedError("Cannot use --index with non-array property.")
		}
		_, err = time.ParseDuration(value)
		if err != nil {
			return burrito.WrapErrorf(err, "Invalid value for duration property.\n"+
				"\tValue: %s", value)
		}
		userConfig.ResolverCacheUpdateCooldown = &value
	case "filter_cache_update_cooldown":
		if index != -1 {
			return burrito.WrappedError("Cannot use --index with non-array property.")
		}
		_, err = time.ParseDuration(value)
		if err != nil {
			return burrito.WrapErrorf(err, "Invalid value for duration property.\n"+
				"\tValue: %s", value)
		}
		userConfig.FilterCacheUpdateCooldown = &value
	case "resolvers":
		if index == -1 {
			userConfig.Resolvers = append(userConfig.Resolvers, value)
		} else {
			if len(userConfig.Resolvers) <= index {
				return burrito.WrappedError("Index out of range.")
			}
			userConfig.Resolvers[index] = value
		}
		// Delete duplicates, removing items from the end
		resolversSet := make(map[string]struct{})
		for i := 0; i < len(userConfig.Resolvers); i++ {
			resolver := userConfig.Resolvers[i]
			if _, ok := resolversSet[resolver]; ok {
				userConfig.Resolvers = append(
					userConfig.Resolvers[:i], userConfig.Resolvers[i+1:]...)
				i--
			} else {
				resolversSet[resolver] = struct{}{}
			}
		}
	default:
		return burrito.WrappedErrorf(invalidUserConfigPropertyError, key)
	}
	err = userConfig.dump(configPath)
	if err != nil {
		return burrito.WrapErrorf(err, userConfigDumpError, configPath)
	}
	return nil
}

// manageUserConfigDelete is a helper function for ManageConfig used to delete
// the specified value from the user configuration.
func manageUserConfigDelete(debug bool, index int, key string) error {
	configPath, err := getGlobalUserConfigPath()
	if err != nil {
		return burrito.WrapError(err, getGlobalUserConfigPathError)
	}
	Logger.Infof("Editing user configuration.\n\tPath: %s", configPath)
	userConfig := NewUserConfig()
	userConfig.fillWithFileData(configPath)
	switch key {
	case "use_project_app_data_storage":
		if index != -1 {
			return burrito.WrappedError("Cannot use --index with non-array property.")
		}
		userConfig.UseProjectAppDataStorage = nil
	case "username":
		if index != -1 {
			return burrito.WrappedError("Cannot use --index with non-array property.")
		}
		userConfig.Username = nil
	case "resolvers":
		if index == -1 {
			userConfig.Resolvers = nil
		} else {
			if len(userConfig.Resolvers) <= index {
				return burrito.WrappedError("Index out of range.")
			}
			userConfig.Resolvers = append(
				userConfig.Resolvers[:index],
				userConfig.Resolvers[index+1:]...)
		}
	default:
		return burrito.WrappedErrorf(invalidUserConfigPropertyError, key)
	}
	err = userConfig.dump(configPath)
	if err != nil {
		return burrito.WrapErrorf(err, userConfigDumpError, configPath)
	}
	return nil
}

// ManageConfig handles the "regolith config" command. It can modify or
// print the user configuration
//   - debug - print debug messages
//   - global - modify global configuration
//   - local - modify local configuration
//   - delete - delete the specified value
//   - append - append a value to an array property of the configuration. Applies
//     only to the array properties
//   - index - the index of the value to modify. Applies only to the array
//     properties
//   - args - the arguments of the command, the length of the list must be 0, 1
//     or 2. The length determines the action of the command.
func ManageConfig(debug, full, delete, append bool, index int, args []string) error {
	InitLogging(debug)

	var err error
	// Based on number of arguments, determine what to do
	if len(args) == 0 {
		// 0 ARGUMENTS - Print all

		// Check illegal flags
		if index != -1 {
			return burrito.WrappedError("Cannot use --index without a key.")
		}
		if delete {
			return burrito.WrappedError("Cannot use --delete without a key.")
		}
		if append {
			return burrito.WrappedError("Cannot use --append without a key.")
		}
		// Print all
		err = manageUserConfigPrintAll(debug, full)
		if err != nil {
			return burrito.PassError(err)
		}
		return nil
	} else if len(args) == 1 {
		// 1 ARGUMENT - Print specific or delete

		// Check illegal flags
		if append {
			return burrito.WrappedError("Cannot use --append flag without a value.")
		}

		// Delete or print
		if delete {
			if full {
				return burrito.WrappedError("The --full flag is only valid for printing.")
			}
			err = manageUserConfigDelete(debug, index, args[0])
			if err != nil {
				return burrito.PassError(err)
			}
			return nil
		} else {
			if index != -1 {
				return burrito.WrappedError("The --index flag is not allowed for printing.")
			}
			err = manageUserConfigPrint(debug, full, args[0])
			if err != nil {
				return burrito.PassError(err)
			}
			return nil
		}
	} else if len(args) == 2 {
		// 2 ARGUMENTS - Set or append

		// Check illegal flags
		if delete {
			return burrito.WrappedError("When using --delete, only one argument is allowed.")
		}
		if full {
			return burrito.WrappedError("The --full flag is only valid for printing.")
		}

		// Set or append
		err = manageUserConfigEdit(debug, index, args[0], args[1])
		if err != nil {
			return burrito.PassError(err)
		}
		return nil
	} else {
		return burrito.WrappedError("Too many arguments.")
	}
}
