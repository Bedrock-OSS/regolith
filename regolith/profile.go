package regolith

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// RecycledSetupTmpFiles set up the workspace for the filters. The function
// uses cached data about the state of the project files to reduce the number
// of file system operations.
func RecycledSetupTmpFiles(config Config, profile Profile) error {
	start := time.Now()
	err := os.MkdirAll(".regolith/tmp", 0666)
	if err != nil {
		return WrapError(
			err, "Unable to prepare temporary directory: \"./regolith/tmp\".")
	}
	// Copy the contents of the 'regolith' folder to '.regolith/tmp'
	if config.ResourceFolder != "" {
		Logger.Debug("Copying project files to .regolith/tmp")
		err = FullRecycledMoveOrCopy(
			config.ResourceFolder, ".regolith/tmp/RP",
			RecycledMoveOrCopySettings{
				canMove:                 false,
				saveSourceHashes:        false,
				saveTargetHashes:        false,
				copyTargetAclFromParent: false,
				reloadSourceHashes:      true,
			})
		if err != nil {
			return WrapErrorf(
				err, "Failed to setup resource folder in the temporary directory.")
		}
	}
	if config.BehaviorFolder != "" {
		err = FullRecycledMoveOrCopy(
			config.BehaviorFolder, ".regolith/tmp/BP",
			RecycledMoveOrCopySettings{
				canMove:                 false,
				saveSourceHashes:        false,
				saveTargetHashes:        false,
				copyTargetAclFromParent: false,
				reloadSourceHashes:      true,
			})
		if err != nil {
			return WrapErrorf(
				err, "Failed to setup behavior folder in the temporary directory.")
		}
	}
	if config.DataPath != "" {
		err = FullRecycledMoveOrCopy(
			config.DataPath, ".regolith/tmp/data",
			RecycledMoveOrCopySettings{
				canMove:                 false,
				saveSourceHashes:        false,
				saveTargetHashes:        false,
				copyTargetAclFromParent: false,
				reloadSourceHashes:      true,
			})
		if err != nil {
			return WrapErrorf(
				err, "Failed to setup data folder in the temporary directory.")
		}
	}

	Logger.Debug("Setup done in ", time.Since(start))
	return nil
}

func CheckProfileImpl(profile Profile, profileName string, config Config, parentContext *RunContext) error {
	// Check whether every filter, uses a supported filter type
	for _, f := range profile.Filters {
		err := f.Check(RunContext{Config: &config, Parent: parentContext, Profile: profileName})
		if err != nil {
			return WrapErrorf(err, "Filter check failed.")
		}
	}
	return nil
}

func RunProfileImpl(profile Profile, profileName string, config Config, parentContext *RunContext) error {
	// Run the filters!
	for filter := range profile.Filters {
		filter := profile.Filters[filter]
		path, _ := filepath.Abs(".")
		// Disabled filters are skipped
		if filter.IsDisabled() {
			Logger.Infof("Filter \"%s\" is disabled, skipping.", filter.GetId())
			continue
		}
		// Skip printing if the filter ID is empty (most likely a nested profile)
		if filter.GetId() != "" {
			Logger.Infof("Running filter %s", filter.GetId())
		}
		start := time.Now()
		err := filter.Run(RunContext{AbsoluteLocation: path, Config: &config, Parent: parentContext, Profile: profileName})
		Logger.Debugf("Executed in %s", time.Since(start))
		if err != nil {
			err1 := ClearCachedStates() // Just to be safe clear cached states
			if err1 != nil {
				err = WrapError(
					err1, "Failed to clear cached file path states while "+
						"handling another error.")
			}
			return WrapError(err, "Failed to run filter.")
		}
	}
	return nil
}

// RunProfile loads the profile from config.json and runs it. The profileName
// is the name of the profile which should be loaded from the configuration.
func RunProfile(profileName string) error {
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

	err = CheckProfileImpl(profile, profileName, *config, nil)
	if err != nil {
		return err
	}

	// Prepare tmp files
	err = RecycledSetupTmpFiles(*config, profile)
	if err != nil {
		err1 := ClearCachedStates() // Just to be safe clear cached states
		if err1 != nil {
			err = WrapError(
				err1, "Failed to clear cached file path states whil handling"+
					" another error.")
		}
		return WrapError(err, "Unable to setup profile.")
	}

	err = RunProfileImpl(profile, profileName, *config, nil)
	if err != nil {
		return PassError(err)
	}

	// Export files
	Logger.Info("Moving files to target directory.")
	start := time.Now()
	err = RecycledExportProject(profile, config.Name, config.DataPath)
	if err != nil {
		err1 := ClearCachedStates() // Just to be safe clear cached states
		if err1 != nil {
			err = WrapError(
				err1, "Failed to clear cached file path states while "+
					"handling another error.")
		}
		return WrapError(err, "Exporting project failed.")
	}
	Logger.Debug("Done in ", time.Since(start))
	return nil
}

// WatchProfile is used by "regolith watch". It is the same as RunProfile, but
// it can be interrupted through the use of the interrupt channel. On various
// stages of the execution, the function checks the channel, and if it is
// sending a message, it restarts.
func WatchProfile(profileName string, profile *Profile, config *Config, interrupt chan string) error {
	// Checks if the interrupt channel is sending a message.
	isInterrupted := func(ignoreData bool) bool {
		select {
		case source := <-interrupt:
			Logger.Warn(
				"A new change was detected while running profile. " +
					"Restarting...")
			if ignoreData && source == "data" {
				return false
			}
			return true
		default:
			return false
		}
	}
	// Saves the state of the tmp files
	saveTmp := func() error {
		err1 := SaveStateInDefaultCache(".regolith/tmp/RP")
		err2 := SaveStateInDefaultCache(".regolith/tmp/BP")
		err3 := SaveStateInDefaultCache(".regolith/tmp/data")
		if err := firstErr(err1, err2, err3); err != nil {
			err1 := ClearCachedStates() // Just to be safe - clear cached states
			if err1 != nil {
				err = WrapError(
					err1, "Failed to clear cached file path states while "+
						"handling another error.")
			}
			return WrapError(err, "Failed to run filter.")
		}
		return nil
	}
	// LOL XD - it could be a loop and interruptions could cause continue
	// instead of using a label and goto bu honestly I think this is more
	// readable.
	// Obviously this need to be changed. Don't judge me. I know goto is
	// considered the worst programming practice in the world but I never used
	// it for anything and it was really exciting to write it like that :D
	//
	// I probably won't change it. I wonder if anyone will ever read this...
start:
	// Prepare tmp files
	err := RecycledSetupTmpFiles(*config, *profile)
	if err != nil {
		err1 := ClearCachedStates() // Just to be safe clear cached states
		if err1 != nil {
			err = WrapError(
				err1, "Failed to clear cached file path states whil handling"+
					" another error.")
		}
		return WrapError(err, "Unable to setup profile.")
	}
	if isInterrupted(false) {
		if err := saveTmp(); err != nil {
			return PassError(err)
		}
		goto start
	}

	interrupted, err := WatchProfileImpl(
		*profile, profileName, *config, nil, func() bool { return isInterrupted(false) })

	if interrupted { // Save the current target state before rerun
		if err := saveTmp(); err != nil {
			return PassError(err)
		}
		goto start
	}
	if err != nil {
		return PassError(err)
	}

	// Export files
	Logger.Info("Moving files to target directory.")
	start := time.Now()
	err = RecycledExportProject(*profile, config.Name, config.DataPath)
	if err != nil {
		err1 := ClearCachedStates() // Just to be safe clear cached states
		if err1 != nil {
			err = WrapError(
				err1, "Failed to clear cached file path states while "+
					"handling another error.")
		}
		return WrapError(err, "Exporting project failed.")
	}
	if isInterrupted(true) { // Ignore the interruptions from the data path
		if err := saveTmp(); err != nil {
			return PassError(err)
		}
		goto start
	}
	Logger.Debug("Done in ", time.Since(start))
	return nil
}

// TODO - refactor this code, write proper comment
// WatchProfileImpl ...
/// isInterrupted - a function that checks the interruptions.
// ... returns whether there were any interruptions and an error
func WatchProfileImpl(
	profile Profile, profileName string, config Config,
	parentContext *RunContext, isInterrupted func() bool,
) (bool, error) {
	// Run the filters!
	for filter := range profile.Filters {
		filter := profile.Filters[filter]
		path, _ := filepath.Abs(".")
		// Disabled filters are skipped
		if filter.IsDisabled() {
			Logger.Infof("Filter \"%s\" is disabled, skipping.", filter.GetId())
			continue
		}
		// Skip printing if the filter ID is empty (most likely a nested profile)
		if filter.GetId() != "" {
			Logger.Infof("Running filter %s", filter.GetId())
		}
		start := time.Now()
		// TODO - This function should propagate the interruptions for the
		// profile filter currently it doesn't do that so nested profiles can
		// handle interruptions only after the execution of the entire prfoile.
		err := filter.Run(
			RunContext{
				AbsoluteLocation: path, Config: &config, Parent: parentContext,
				Profile: profileName})
		Logger.Debugf("Executed in %s", time.Since(start))
		if err != nil {
			err1 := ClearCachedStates() // Just to be safe clear cached states
			if err1 != nil {
				err = WrapError(
					err1, "Failed to clear cached file path states while "+
						"handling another error.")
			}
			return false, WrapError(err, "Failed to run filter.")
		}
		if isInterrupted() {
			return true, nil
		}
	}
	return false, nil
}

// SubfilterCollection returns a collection of filters from a
// "filter.json" file of a remote filter.
func (f *RemoteFilter) SubfilterCollection() (*FilterCollection, error) {
	path := filepath.Join(f.GetDownloadPath(), "filter.json")
	result := &FilterCollection{Filters: []FilterRunner{}}
	file, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, WrapErrorf(err, "Couldn't read %q.", path)
	}

	var filterCollection map[string]interface{}
	err = json.Unmarshal(file, &filterCollection)
	if err != nil {
		return nil, WrapErrorf(
			err, "Couldn't load %s! Does the file contain correct json?", path)
	}
	// Filters
	filters, ok := filterCollection["filters"].([]interface{})
	if !ok {
		return nil, WrappedErrorf("Could not parse filters of %q.", path)
	}
	for i, filter := range filters {
		filter, ok := filter.(map[string]interface{})
		if !ok {
			return nil, WrappedErrorf(
				"Could not parse filter %v of %q.", i, path)
		}
		// Using the same JSON data to create both the filter
		// definiton (installer) and the filter (runner)
		filterId := fmt.Sprintf("%v:subfilter%v", f.Id, i)
		filterInstaller, err := FilterInstallerFromObject(filterId, filter)
		if err != nil {
			return nil, WrapErrorf(
				err, "Could not parse filter %v of %q.", i, path)
		}
		// Remote filters don't have the "filter" key but this would break the
		// code as it's required by local filters. Adding it here to make the
		// code work.
		// TODO - this is a hack, fix it!
		filter["filter"] = filterId
		filterRunner, err := filterInstaller.CreateFilterRunner(filter)
		if err != nil {
			return nil, WrapErrorf(
				err, "Could not parse filter %v of %q.", i, path)
		}
		if _, ok := filterRunner.(*RemoteFilter); ok {
			// TODO - we could possibly implement recursive filters here
			return nil, WrappedErrorf(
				"remote filters are not allowed in subfilters. Remote filter"+
					" %q subfilter %v", f.Id, i)
		}
		filterRunner.CopyArguments(f)
		result.Filters = append(result.Filters, filterRunner)
	}
	return result, nil
}

// FilterCollection is a list of filters
type FilterCollection struct {
	Filters []FilterRunner `json:"filters"`
}

type Profile struct {
	FilterCollection
	ExportTarget ExportTarget `json:"export,omitempty"`
}

func ProfileFromObject(
	obj map[string]interface{}, filterDefinitions map[string]FilterInstaller,
) (Profile, error) {
	result := Profile{}
	// Filters
	if _, ok := obj["filters"]; !ok {
		return result, WrappedError("Missing \"filters\" property.")
	}
	filters, ok := obj["filters"].([]interface{})
	if !ok {
		return result, WrappedError("Could not parse \"filters\" property.")
	}
	for i, filter := range filters {
		filter, ok := filter.(map[string]interface{})
		if !ok {
			return result, WrappedErrorf(
				"The %s filter from the list is not a map.", nth(i))
		}
		filterRunner, err := FilterRunnerFromObjectAndDefinitions(
			filter, filterDefinitions)
		if err != nil {
			return result, WrapErrorf(
				err, "Could not parse the %v filter of the profile.", nth(i))
		}
		result.Filters = append(result.Filters, filterRunner)
	}
	// ExportTarget
	if _, ok := obj["export"]; !ok {
		return result, WrappedError("Missing \"export\" property.")
	}
	export, ok := obj["export"].(map[string]interface{})
	if !ok {
		return result, WrappedError(
			"The \"export\" property is not a map.")
	}
	exportTarget, err := ExportTargetFromObject(export)
	if err != nil {
		return result, WrapError(err, "Could not parse \"export\".")
	}
	result.ExportTarget = exportTarget
	return result, nil
}
