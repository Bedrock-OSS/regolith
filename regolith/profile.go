package regolith

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Bedrock-OSS/go-burrito/burrito"

	"github.com/otiai10/copy"
)

// SetupTmpFiles set up the workspace for the filters.
func SetupTmpFiles(context RunContext) error {
	config := *context.Config
	dotRegolithPath := context.DotRegolithPath
	start := time.Now()
	useSizeTimeCheck := IsExperimentEnabled(SizeTimeCheck)
	useSymlinkExport := IsExperimentEnabled(SymlinkExport)
	tmpPath := filepath.Join(dotRegolithPath, "tmp")
	bpTmpPath := filepath.Join(tmpPath, "BP")
	rpTmpPath := filepath.Join(tmpPath, "RP")

	// Check if should create symlinks, if yes load bp and rp paths
	var bpExportPath, rpExportPath string
	shouldCreateSymlinks := false
	if useSymlinkExport {
		profile, err := context.GetProfile()
		if err != nil {
			return burrito.WrapErrorf(err, runContextGetProfileError)
		}
		bpExportPath, rpExportPath, err = GetExportPaths(profile.ExportTarget, context)
		if err != nil {
			return burrito.WrapError(err, getExportPathsError)
		}
		if profile.ExportTarget.Target == "none" {
			useSymlinkExport = false
		} else {
			bpLink := isSymlinkTo(bpTmpPath, bpExportPath)
			rpLink := isSymlinkTo(rpTmpPath, rpExportPath)
			// If either symlink doesn't exist, create them
			shouldCreateSymlinks = !bpLink || !rpLink
		}
	}
	// If we're not using symlink export make sure there is no symlinks
	if !useSymlinkExport {
		if isSymlink(bpTmpPath) {
			err := os.Remove(bpTmpPath)
			if err != nil {
				return burrito.WrapErrorf(err, osRemoveError, bpTmpPath)
			}
		}
		if isSymlink(rpTmpPath) {
			err := os.Remove(rpTmpPath)
			if err != nil {
				return burrito.WrapErrorf(err, osRemoveError, rpTmpPath)
			}
		}
	}

	// Clean the temporary directory
	isRegularRun := !useSizeTimeCheck && !useSymlinkExport
	if isRegularRun || shouldCreateSymlinks {
		Logger.Debugf("Cleaning \"%s\"", tmpPath)
		err := os.RemoveAll(tmpPath)
		if err != nil {
			return burrito.WrapErrorf(err, osRemoveError, tmpPath)
		}
	}

	// Prepare temp path root
	err := os.MkdirAll(tmpPath, 0755)
	if err != nil {
		return burrito.WrapErrorf(err, osMkdirError, tmpPath)
	}

	// Create symlinks
	if shouldCreateSymlinks {
		// Check deletion safety
		editedFiles := LoadEditedFiles(dotRegolithPath)
		err := editedFiles.CheckDeletionSafety(rpExportPath, bpExportPath)
		if err != nil {
			return burrito.WrapErrorf(
				err,
				checkDeletionSafetyError,
				rpExportPath, bpExportPath)
		}

		// Remove existing exported paths
		if err := os.RemoveAll(bpExportPath); err != nil {
			return burrito.WrapErrorf(err, osRemoveError, bpExportPath)
		}
		if err := os.RemoveAll(rpExportPath); err != nil {
			return burrito.WrapErrorf(err, osRemoveError, rpExportPath)
		}

		// Create symlinks
		if err := createDirLink(filepath.Join(tmpPath, "BP"), bpExportPath); err != nil {
			return burrito.WrapErrorf(err, createDirLinkError, filepath.Join(tmpPath, "BP"), bpExportPath)
		}
		if err := createDirLink(filepath.Join(tmpPath, "RP"), rpExportPath); err != nil {
			return burrito.WrapErrorf(err, createDirLinkError, filepath.Join(tmpPath, "RP"), rpExportPath)
		}
	}

	// Copy the contents of the 'regolith' folder to '[dotRegolithPath]/tmp'
	Logger.Debugf("Copying project files to \"%s\"", tmpPath)
	// Avoid repetitive code of preparing ResourceFolder, BehaviorFolder
	// and DataPath with a closure
	setupTmpDirectory := func(
		path, shortName, descriptiveName string,
	) error {
		p := filepath.Join(tmpPath, shortName)
		if path != "" {
			stats, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					Logger.Warnf(
						"%s %q does not exist", descriptiveName, path)
					err = os.MkdirAll(p, 0755)
					if err != nil {
						return burrito.WrapErrorf(err, osMkdirError, p)
					}
				}
			} else if stats.IsDir() {
				if useSizeTimeCheck || useSymlinkExport {
					err = SyncDirectories(path, p, false)
					if err != nil {
						return burrito.WrapError(err, "Failed to export behavior pack.")
					}
				} else {
					err = copy.Copy(
						path,
						p,
						copy.Options{PreserveTimes: false, Sync: false})
					if err != nil {
						return burrito.WrapErrorf(err, osCopyError, path, p)
					}
				}
			} else { // The folder path leads to a file
				return burrito.WrappedErrorf(isDirNotADirError, path)
			}
		} else {
			err := os.MkdirAll(p, 0755)
			if err != nil {
				return burrito.WrapErrorf(err, osMkdirError, p)
			}
		}
		return nil
	}

	// Setup RP, BP and data folders concurrently
	wg := sync.WaitGroup{}
	errCh := make(chan error, 3)

	wg.Go(func() {
		if err := setupTmpDirectory(config.ResourceFolder, "RP", "resource folder"); err != nil {
			errCh <- burrito.WrapErrorf(err, "Failed to setup RP folder in the temporary directory.")
		}
	})

	wg.Go(func() {
		if err := setupTmpDirectory(config.BehaviorFolder, "BP", "behavior folder"); err != nil {
			errCh <- burrito.WrapErrorf(err, "Failed to setup BP folder in the temporary directory.")
		}
	})

	wg.Go(func() {
		if err := setupTmpDirectory(config.DataPath, "data", "data folder"); err != nil {
			errCh <- burrito.WrapErrorf(err, "Failed to setup data folder in the temporary directory.")
		}
	})

	wg.Wait()
	close(errCh)
	for e := range errCh {
		if e != nil {
			return e
		}
	}

	// Update the edited files list if new symlinks were created. The new
	// content is safe to edit.
	if shouldCreateSymlinks {
		editedFiles := NewEditedFiles()
		err = editedFiles.UpdateFromPaths(rpExportPath, bpExportPath)
		if err != nil {
			return burrito.WrapError(
				err,
				"Failed to create a list of files safe to edit")
		}
		err = editedFiles.Dump(dotRegolithPath)
		if err != nil {
			return burrito.WrapError(err, updatedFilesDumpError)
		}
	}

	Logger.Debug("Setup done in ", time.Since(start))
	return nil
}

func CheckProfileImpl(
	profile Profile, profileName string, config Config,
	parentContext *RunContext, dotRegolithPath string,
) error {
	// Check whether every filter, uses a supported filter type
	for _, f := range profile.Filters {
		err := f.Check(RunContext{
			Config:          &config,
			Parent:          parentContext,
			Profile:         profileName,
			DotRegolithPath: dotRegolithPath,
			Settings:        f.GetSettings(),
		})
		if err != nil {
			return burrito.WrapErrorf(err, filterRunnerCheckError, f.GetId())
		}
	}
	return nil
}

// RunProfile loads the profile from config.json and runs it based on the
// context. If context is in the watch mode, it can repeat the process multiple
// times in case of interruptions (changes in the source files).
func RunProfile(context RunContext) error {
start:
	// Prepare tmp files
	err := SetupTmpFiles(context)
	if err != nil {
		return burrito.WrapErrorf(err, setupTmpFilesError, context.DotRegolithPath)
	}
	if context.IsInterrupted() {
		goto start
	}
	// Run the profile
	interrupted, err := RunProfileImpl(context)
	if err != nil {
		return burrito.PassError(err)
	}
	if interrupted {
		goto start
	}
	// Export files
	Logger.Info("Moving files to target directory.")
	start := time.Now()
	if context.IsInWatchMode() {
		context.fileWatchingStage <- "pause"
	}
	err = ExportProject(context)
	if context.IsInWatchMode() {
		// We need to restart the watcher before error handling. See:
		// https://github.com/Bedrock-OSS/regolith/pull/297#issuecomment-2411981894
		context.fileWatchingStage <- "restart"
	}
	if err != nil {
		return burrito.WrapError(err, exportProjectError)
	}
	if context.IsInterrupted("data") {
		goto start
	}
	Logger.Debug("Done in ", time.Since(start))
	return nil
}

// RunProfileImpl runs the profile from the given context and returns true
// if the execution was interrupted.
func RunProfileImpl(context RunContext) (bool, error) {
	profile, err := context.GetProfile()
	if err != nil {
		return false, burrito.WrapErrorf(err, runContextGetProfileError)
	}
	// Run the filters!
	for filter := range profile.Filters {
		filter := profile.Filters[filter]
		// Disabled filters are skipped
		disabled, err := filter.IsDisabled(context)
		if err != nil {
			return false, burrito.WrapErrorf(err, "Failed to check if filter is disabled")
		}
		if disabled {
			Logger.Infof("Filter \"%s\" is disabled, skipping.", filter.GetId())
			continue
		}
		// Skip printing if the filter ID is empty (most likely a nested profile)
		if filter.GetId() != "" {
			Logger.Infof("Running filter %s", filter.GetId())
		}
		// Run the filter in watch mode
		start := time.Now()
		interrupted, err := filter.Run(context)
		Logger.Debugf("Executed in %s", time.Since(start))
		if err != nil {
			return false, burrito.WrapErrorf(err, filterRunnerRunError, filter.GetId())
		}
		if interrupted {
			return true, nil
		}
	}
	return false, nil
}

// subfilterCollection returns a collection of filters from a
// "filter.json" file of a remote filter.
func (f *RemoteFilter) subfilterCollection(dotRegolithPath string) (*FilterCollection, error) {
	path := filepath.Join(f.GetDownloadPath(dotRegolithPath), "filter.json")
	result := &FilterCollection{Filters: []FilterRunner{}}
	filterCollection, err := loadFilterConfig(path)
	if err != nil {
		return nil, burrito.WrapErrorf(err, readFilterJsonError, path)
	}
	// Filters
	filtersObj, ok := filterCollection["filters"]
	if !ok {
		return nil, extraFilterJsonErrorInfo(
			path, burrito.WrappedErrorf(jsonPathMissingError, "filters"))
	}
	filters, ok := filtersObj.([]any)
	if !ok {
		return nil, extraFilterJsonErrorInfo(
			path, burrito.WrappedErrorf(jsonPathTypeError, "filters", "array"))
	}
	for i, filter := range filters {
		filter, ok := filter.(map[string]any)
		jsonPath := fmt.Sprintf("filters->%d", i) // Used for error messages
		if !ok {
			return nil, extraFilterJsonErrorInfo(
				path, burrito.WrappedErrorf(jsonPathTypeError, jsonPath, "object"))
		}
		// Using the same JSON data to create both the filter
		// definition (installer) and the filter (runner)
		filterId := fmt.Sprintf("%v:subfilter%v", f.Id, i)
		filterInstaller, err := FilterInstallerFromObject(filterId, filter)
		if err != nil {
			return nil, extraFilterJsonErrorInfo(
				path, burrito.WrapErrorf(err, jsonPathParseError, jsonPath))
		}
		filterRunner, err := filterInstaller.CreateFilterRunner(filter, filterId)
		if err != nil {
			// TODO - better filterName?
			filterName := fmt.Sprintf("%v filter from %s.", nth(i), path)
			return nil, burrito.WrapErrorf(
				err, createFilterRunnerError, filterName)
		}
		if _, ok := filterRunner.(*RemoteFilter); ok {
			// TODO - we could possibly implement recursive filters here
			return nil, burrito.WrappedErrorf(
				"Regolith detected a reference to a remote filter inside "+
					"another remote filter.\n"+
					"This feature is not supported.\n"+
					"Filter name: %s"+
					"Filter configuration file: %s\n"+
					"JSON path to remote filter reference: filters->%d",
				f.Id, path, i)
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

// Profile is a collection of filters and an export target
// When editing, adjust ProfileFromObject function as well
type Profile struct {
	FilterCollection
	ExportTarget ExportTarget `json:"export,omitzero"`
}

func ProfileFromObject(
	obj map[string]any, filterDefinitions map[string]FilterInstaller,
) (Profile, error) {
	result := Profile{}
	// Filters
	if _, ok := obj["filters"]; !ok {
		return result, burrito.WrappedErrorf(jsonPathMissingError, "filters")
	}
	filters, ok := obj["filters"].([]any)
	if !ok {
		return result, burrito.WrappedErrorf(jsonPathTypeError, "filters", "array")
	}
	for i, filter := range filters {
		filter, ok := filter.(map[string]any)
		if !ok {
			return result, burrito.WrappedErrorf(
				jsonPathTypeError, fmt.Sprintf("filters->%d", i), "object")
		}
		filterRunner, err := FilterRunnerFromObjectAndDefinitions(
			filter, filterDefinitions, false)
		if err != nil {
			return result, burrito.WrapErrorf(
				err, jsonPathParseError, fmt.Sprintf("filters->%d", i))
		}
		result.Filters = append(result.Filters, filterRunner)
	}
	// ExportTarget
	if _, ok := obj["export"]; !ok {
		return result, burrito.WrappedErrorf(jsonPathMissingError, "export")
	}
	export, ok := obj["export"].(map[string]any)
	if !ok {
		return result, burrito.WrappedErrorf(jsonPathTypeError, "export", "object")
	}
	exportTarget, err := ExportTargetFromObject(export)
	if err != nil {
		return result, burrito.WrapErrorf(err, jsonPathParseError, "export")
	}
	result.ExportTarget = exportTarget
	return result, nil
}

// ForeachFilter iterates over the filters of the profile and applies the
// given function to each filter. If unpackNestedProfiles is true, it will
// unpack the nested profiles and apply the function to their filters as well.
func (p *Profile) ForeachFilter(
	context RunContext,
	fn func(FilterRunner) error,
	unpackNestedProfiles bool,
) error {
	for i := range p.Filters {
		filter := p.Filters[i]
		profileFilter, ok := filter.(*ProfileFilter)
		if unpackNestedProfiles && ok {
			subProfile, ok := context.Config.Profiles[profileFilter.Profile]
			if !ok {
				return burrito.WrappedErrorf(
					"Failed to find profile of the profile-filter.\n"+
						"Parent profile: %s\n"+
						"Profile filter index: %d\n"+
						"Referenced profile: %s\n",
					context.Profile, i, profileFilter.Profile,
				)
			}
			err := subProfile.ForeachFilter(context, fn, unpackNestedProfiles)
			if err != nil {
				return burrito.WrappedErrorf(
					"Failed to iterate over filters of the profile-filter.\n"+
						"Parent profile: %s\n"+
						"Profile filter index: %d\n"+
						"Referenced profile: %s\n",
					context.Profile, i, profileFilter.Profile,
				)
			}
		} else {
			err := fn(filter)
			if err != nil {
				return burrito.WrapErrorf(
					err,
					"Failed to iterate apply function to the filter.\n"+
						"Profile: %s\n"+
						"Filter index: %d\n",
					context.Profile, i,
				)
			}
		}
	}
	return nil
}
