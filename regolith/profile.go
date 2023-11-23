package regolith

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Bedrock-OSS/go-burrito/burrito"

	"github.com/otiai10/copy"
)

type FileSystem interface {
	Close()
	SaveToPath(from, to string, makeReadOnly, copyParentAcl bool) error
	SyncToPath(from, to string, makeReadOnly, copyParentAcl bool) error
}

type NoopFileSystem struct{}

func (f *NoopFileSystem) Close() {

}

func (f *NoopFileSystem) SaveToPath(from, to string, makeReadOnly, copyParentAcl bool) error {
	return nil
}

func (f *NoopFileSystem) SyncToPath(from, to string, makeReadOnly, copyParentAcl bool) error {
	return nil
}

type TmpFileSystem struct {
	dotRegolithPath string
}

func (f *TmpFileSystem) Close() {

}

func (f *TmpFileSystem) SaveToPath(from, to string, makeReadOnly, copyParentAcl bool) error {
	return MoveOrCopy(filepath.Join(f.dotRegolithPath, "tmp", from), to, makeReadOnly, copyParentAcl)
}

func (f *TmpFileSystem) SyncToPath(from, to string, makeReadOnly, copyParentAcl bool) error {
	return SyncDirectories(filepath.Join(f.dotRegolithPath, "tmp", from), to, makeReadOnly, copyParentAcl)
}

// SetupTmpFiles set up the workspace for the filters.
func SetupTmpFiles(config Config, dotRegolithPath string) (FileSystem, error) {
	// Prepare cleanup function
	var fs FileSystem = &NoopFileSystem{}
	// Setup Directories
	MeasureStart("SetupTmpFiles - Clean")
	tmpPath := filepath.Join(dotRegolithPath, "tmp")
	Logger.Debugf("Cleaning \"%s\"", tmpPath)
	err := os.RemoveAll(tmpPath)
	if err != nil {
		return fs, burrito.WrapErrorf(err, osRemoveError, tmpPath)
	}

	err = os.MkdirAll(tmpPath, 0755)
	if err != nil {
		return fs, burrito.WrapErrorf(err, osMkdirError, tmpPath)
	}
	MeasureEnd()

	if IsExperimentEnabled(VFS) {
		Logger.Debugf("Setting up VFS at \"%s\"", tmpPath)
		MeasureStart("SetupTmpFiles - Set up VFS")
		fs, err = NewRegolithFS([]Root{
			{
				Source:    config.ResourceFolder,
				MountName: "RP",
			},
			{
				Source:    config.BehaviorFolder,
				MountName: "BP",
			},
			{
				Source:    config.DataPath,
				MountName: "data",
			},
		}, tmpPath)
		if err != nil {
			return fs, burrito.WrapErrorf(err, "Failed to create VFS at %s.", tmpPath)
		}
		MeasureEnd()
	} else {
		fs = &TmpFileSystem{dotRegolithPath: dotRegolithPath}
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
					err = copy.Copy(
						path,
						p,
						copy.Options{PreserveTimes: IsExperimentEnabled(SizeTimeCheck), Sync: false})
					if err != nil {
						return burrito.WrapErrorf(err, osCopyError, path, p)
					}
				} else { // The folder paths leads to a file
					return burrito.WrappedErrorf(isDirNotADirError, path)
				}
			} else {
				err = os.MkdirAll(p, 0755)
				if err != nil {
					return burrito.WrapErrorf(err, osMkdirError, p)
				}
			}
			return nil
		}

		MeasureStart("SetupTmpFiles - RP")
		err = setupTmpDirectory(config.ResourceFolder, "RP", "resource folder")
		if err != nil {
			return fs, burrito.WrapErrorf(
				err, "Failed to setup RP folder in the temporary directory.")
		}
		MeasureStart("SetupTmpFiles - BP")
		err = setupTmpDirectory(config.BehaviorFolder, "BP", "behavior folder")
		if err != nil {
			return fs, burrito.WrapErrorf(
				err, "Failed to setup BP folder in the temporary directory.")
		}
		MeasureStart("SetupTmpFiles - Data")
		err = setupTmpDirectory(config.DataPath, "data", "data folder")
		if err != nil {
			return fs, burrito.WrapErrorf(
				err, "Failed to setup data folder in the temporary directory.")
		}

		MeasureEnd()
	}
	return fs, nil
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
	profile, err := context.GetProfile()
	if err != nil {
		return burrito.WrapErrorf(err, runContextGetProfileError)
	}
	fs, err := SetupTmpFiles(*context.Config, context.DotRegolithPath)
	// in case of VFS, cleanup must be called even in case of an error
	defer fs.Close()
	if err != nil {
		return burrito.WrapErrorf(err, setupTmpFilesError, context.DotRegolithPath)
	}
	if context.IsInterrupted() {
		// Dirty gotos break my deferred cleanup
		fs.Close()
		goto start
	}
	// Run the profile
	interrupted, err := RunProfileImpl(context)
	if err != nil {
		return burrito.PassError(err)
	}
	if interrupted {
		// Dirty gotos break my deferred cleanup
		fs.Close()
		goto start
	}
	// Export files
	Logger.Info("Moving files to target directory.")
	start := time.Now()
	err = ExportProject(
		profile, context.Config.Name, context.Config.DataPath, context.DotRegolithPath, fs)
	if err != nil {
		return burrito.WrapError(err, exportProjectError)
	}
	if context.IsInterrupted("data") {
		// Dirty gotos break my deferred cleanup
		fs.Close()
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
	file, err := os.ReadFile(path)

	if err != nil {
		return nil, burrito.WrappedErrorf(readFilterJsonError, path)
	}

	var filterCollection map[string]interface{}
	err = json.Unmarshal(file, &filterCollection)
	if err != nil {
		return nil, burrito.WrapErrorf(err, jsonUnmarshalError, path)
	}
	// Filters
	filtersObj, ok := filterCollection["filters"]
	if !ok {
		return nil, extraFilterJsonErrorInfo(
			path, burrito.WrappedErrorf(jsonPathMissingError, "filters"))
	}
	filters, ok := filtersObj.([]interface{})
	if !ok {
		return nil, extraFilterJsonErrorInfo(
			path, burrito.WrappedErrorf(jsonPathTypeError, "filters", "array"))
	}
	for i, filter := range filters {
		filter, ok := filter.(map[string]interface{})
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
		// Remote filters don't have the "filter" key but this would break the
		// code as it's required by local filters. Adding it here to make the
		// code work.
		// TODO - this is a hack, fix it!
		filter["filter"] = filterId
		filterRunner, err := filterInstaller.CreateFilterRunner(filter)
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
		return result, burrito.WrappedErrorf(jsonPathMissingError, "filters")
	}
	filters, ok := obj["filters"].([]interface{})
	if !ok {
		return result, burrito.WrappedErrorf(jsonPathTypeError, "filters", "array")
	}
	for i, filter := range filters {
		filter, ok := filter.(map[string]interface{})
		if !ok {
			return result, burrito.WrappedErrorf(
				jsonPathTypeError, fmt.Sprintf("filters->%d", i), "object")
		}
		filterRunner, err := FilterRunnerFromObjectAndDefinitions(
			filter, filterDefinitions)
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
	export, ok := obj["export"].(map[string]interface{})
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
