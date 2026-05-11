package regolith

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"github.com/otiai10/copy"
)

// runShellCommands executes multiple shell commands in a single shell session,
// allowing environment variables to persist across commands and be injected into the parent process.
func runShellCommands(commands []string) error {
	if len(commands) == 0 {
		return nil
	}

	if runtime.GOOS == "windows" {
		return runShellCommandsWindows(commands)
	}
	return runShellCommandsUnix(commands)
}

// runShellCommandsWindows executes commands in PowerShell and captures environment changes
func runShellCommandsWindows(commands []string) error {
	// Build a script that:
	// 1. Executes all user commands
	// 2. Outputs environment variables in a parseable format
	script := ""
	for _, cmd := range commands {
		Logger.Debugf("Executing shell command: %s", cmd)
		script += cmd + "; "
	}
	// Output all environment variables after commands execute
	script += "[Environment]::GetEnvironmentVariables('Process').GetEnumerator() | ForEach-Object { Write-Output \"__REGOLITH_ENV__$($_.Key)=$($_.Value)\" }"

	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	// Capture stdout to parse environment variables
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	// Display output (excluding our env markers)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "__REGOLITH_ENV__") {
			// Parse and set environment variable
			envLine := strings.TrimPrefix(line, "__REGOLITH_ENV__")
			if idx := strings.Index(envLine, "="); idx > 0 {
				key := envLine[:idx]
				value := strings.TrimSpace(envLine[idx+1:])
				os.Setenv(key, value)
			}
		} else if line != "" {
			// Print regular output
			fmt.Println(line)
		}
	}

	return nil
}

// runShellCommandsUnix executes commands in sh and captures environment changes
func runShellCommandsUnix(commands []string) error {
	// Build a script that:
	// 1. Executes all user commands
	// 2. Outputs environment variables in a parseable format
	script := "set -e\n" // Exit on error
	for _, cmd := range commands {
		Logger.Debugf("Executing shell command: %s", cmd)
		script += cmd + "\n"
	}
	// Output all environment variables after commands execute
	script += "env | while IFS='=' read -r key value; do echo \"__REGOLITH_ENV__$key=$value\"; done"

	cmd := exec.Command("sh", "-c", script)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	// Capture stdout to parse environment variables
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	// Display output (excluding our env markers)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "__REGOLITH_ENV__") {
			// Parse and set environment variable
			envLine := strings.TrimPrefix(line, "__REGOLITH_ENV__")
			if idx := strings.Index(envLine, "="); idx > 0 {
				key := envLine[:idx]
				value := strings.TrimSpace(envLine[idx+1:])
				os.Setenv(key, value)
			}
		} else if line != "" {
			// Print regular output
			fmt.Println(line)
		}
	}

	return nil
}

// SetupTmpFiles set up the workspace for the filters.
func SetupTmpFiles(context RunContext) error {
	config := *context.Config
	dotRegolithPath := context.DotRegolithPath
	start := time.Now()
	useSizeTimeCheck := IsExperimentEnabled(SizeTimeCheck)
	useSymlinkExport := IsExperimentEnabled(SymlinkExport)
	absTmpPath, err := GetAbsoluteWorkingDirectory(dotRegolithPath)
	if err != nil {
		return burrito.WrapError(err, getAbsoluteWorkingDirectoryError)
	}
	bpTmpPath := filepath.Join(absTmpPath, "BP")
	rpTmpPath := filepath.Join(absTmpPath, "RP")

	// Check if should create symlinks, if yes load bp and rp paths
	var bpExportPath, rpExportPath string
	shouldCreateSymlinks := false
	if useSymlinkExport {
		profile, err := context.GetProfile()
		if err != nil {
			return burrito.WrapErrorf(err, runContextGetProfileError)
		}
		activeTargets := profile.activeExportTargets()
		if len(activeTargets) != 1 {
			if len(activeTargets) > 1 {
				Logger.Debugf("SymlinkExport experiment is enabled but the profile has multiple active export targets. Using regular export.")
			}
			useSymlinkExport = false
		} else {
			primaryTarget := activeTargets[0]
			bpExportPath, rpExportPath, err = GetExportPaths(primaryTarget, context)
			if err != nil {
				return burrito.WrapError(err, getExportPathsError)
			}
			bpLink := isSymlinkTo(bpTmpPath, bpExportPath)
			rpLink := isSymlinkTo(rpTmpPath, rpExportPath)
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
	if isRegularRun {
		Logger.Debugf("Cleaning \"%s\"", absTmpPath)
		err := os.RemoveAll(absTmpPath)
		if err != nil {
			return burrito.WrapErrorf(err, osRemoveError, absTmpPath)
		}
	} else if shouldCreateSymlinks {
		for _, tmpPath := range []string{bpTmpPath, rpTmpPath} {
			if err := removeJunctionSafe(tmpPath); err != nil {
				return burrito.PassError(err)
			}
		}
	}

	// Prepare temp path root
	err = os.MkdirAll(absTmpPath, 0755)
	if err != nil {
		return burrito.WrapErrorf(err, osMkdirError, absTmpPath)
	}

	// Create symlinks
	if shouldCreateSymlinks {
		if !context.UnsafeMode {
			editedFiles := LoadEditedFiles(dotRegolithPath)
			err := editedFiles.CheckDeletionSafety(rpExportPath, bpExportPath)
			if err != nil {
				return burrito.WrapErrorf(
					err,
					checkDeletionSafetyError,
					rpExportPath, bpExportPath)
			}
		}
		if err := os.MkdirAll(bpExportPath, 0755); err != nil {
			return burrito.WrapErrorf(err, osMkdirError, bpExportPath)
		}
		if err := os.MkdirAll(rpExportPath, 0755); err != nil {
			return burrito.WrapErrorf(err, osMkdirError, rpExportPath)
		}

		if err := createDirLink(filepath.Join(absTmpPath, "BP"), bpExportPath); err != nil {
			return burrito.WrapErrorf(err, createDirLinkError, filepath.Join(absTmpPath, "BP"), bpExportPath)
		}
		if err := createDirLink(filepath.Join(absTmpPath, "RP"), rpExportPath); err != nil {
			return burrito.WrapErrorf(err, createDirLinkError, filepath.Join(absTmpPath, "RP"), rpExportPath)
		}
	}

	// Copy the contents of the 'regolith' folder to '[dotRegolithPath]/tmp'
	Logger.Debugf("Copying project files to \"%s\"", absTmpPath)
	// Avoid repetitive code of preparing ResourceFolder, BehaviorFolder
	// and DataPath with a closure
	setupTmpDirectory := func(
		path, shortName, descriptiveName string,
	) error {
		p := filepath.Join(absTmpPath, shortName)
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
			return burrito.WrapError(err, updatedFilesUpdateError)
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
	// Execute preShell commands if present
	profile, err := context.GetProfile()
	if err != nil {
		return burrito.WrapErrorf(err, runContextGetProfileError)
	}
	preShellCmds := profile.PreShell.GetCommandsForCurrentOS()
	if len(preShellCmds) > 0 {
		Logger.Info("Running preShell commands...")
		err := runShellCommands(preShellCmds)
		if err != nil {
			return burrito.WrapErrorf(err, "PreShell commands failed")
		}
	}

	// Prepare tmp files
	err = SetupTmpFiles(context)
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

	// Execute postShell commands if present
	postShellCmds := profile.PostShell.GetCommandsForCurrentOS()
	if len(postShellCmds) > 0 {
		Logger.Info("Running postShell commands...")
		err := runShellCommands(postShellCmds)
		if err != nil {
			return burrito.WrapErrorf(err, "PostShell commands failed")
		}
	}

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

		err = filter.AddExtraArguments(context.ExtraArguments)
		if err != nil {
			return false, burrito.WrapErrorf(err, filterRunnerRunError, filter.GetId())
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

// ShellCommands represents shell commands that can be either:
// - A simple array of strings (executed on all OS)
// - An object with OS-specific arrays (windows, linux, darwin)
type ShellCommands struct {
	All     []string `json:"-"`
	Windows []string `json:"windows,omitempty"`
	Linux   []string `json:"linux,omitempty"`
	Darwin  []string `json:"darwin,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for ShellCommands.
// If only All field has values, marshals as a JSON array.
// Otherwise, marshals as a JSON object with platform-specific keys.
func (sc ShellCommands) MarshalJSON() ([]byte, error) {
	// If only All has commands, marshal as array
	if len(sc.All) > 0 && len(sc.Windows) == 0 && len(sc.Linux) == 0 && len(sc.Darwin) == 0 {
		return json.Marshal(sc.All)
	}

	// Otherwise, marshal as object with platform keys
	obj := make(map[string][]string)
	if len(sc.Windows) > 0 {
		obj["windows"] = sc.Windows
	}
	if len(sc.Linux) > 0 {
		obj["linux"] = sc.Linux
	}
	if len(sc.Darwin) > 0 {
		obj["darwin"] = sc.Darwin
	}
	return json.Marshal(obj)
}

// GetCommandsForCurrentOS returns the commands to execute for the current OS
func (sc *ShellCommands) GetCommandsForCurrentOS() []string {
	if len(sc.All) > 0 {
		return sc.All
	}

	switch runtime.GOOS {
	case "windows":
		return sc.Windows
	case "linux":
		return sc.Linux
	case "darwin":
		return sc.Darwin
	default:
		return nil
	}
}

// Profile is a collection of filters and export targets.
// When editing, adjust ProfileFromObject function as well.
type Profile struct {
	FilterCollection
	ExportTarget ExportTargets `json:"export,omitzero"`
	PreShell     ShellCommands `json:"preShell,omitzero"`
	PostShell    ShellCommands `json:"postShell,omitzero"`
}

func (p Profile) exportTargets() ExportTargets {
	return p.ExportTarget
}

func (p Profile) activeExportTargets() ExportTargets {
	targets := p.exportTargets()
	activeTargets := make(ExportTargets, 0, len(targets))
	for _, target := range targets {
		if target.Target != "none" {
			activeTargets = append(activeTargets, target)
		}
	}
	return activeTargets
}

func shellCommandsFromObject(obj map[string]any, key string) (ShellCommands, error) {
	var result ShellCommands
	if shellObj, ok := obj[key]; ok {
		if shellArray, ok := shellObj.([]any); ok {
			// Simple array format - applies to all OS
			for i, cmd := range shellArray {
				cmdStr, ok := cmd.(string)
				if !ok {
					return result, burrito.WrappedErrorf(
						jsonPathTypeError, fmt.Sprintf("%s->%d", key, i), "string")
				}
				result.All = append(result.All, cmdStr)
			}
		} else if shellMap, ok := shellObj.(map[string]any); ok {
			// OS-specific format
			if windowsCmds, exists := shellMap["windows"]; exists {
				windowsArray, ok := windowsCmds.([]any)
				if !ok {
					return result, burrito.WrappedErrorf(
						jsonPathTypeError, fmt.Sprintf("%s->windows", key), "array")
				}
				for i, cmd := range windowsArray {
					cmdStr, ok := cmd.(string)
					if !ok {
						return result, burrito.WrappedErrorf(
							jsonPathTypeError, fmt.Sprintf("%s->windows->%d", key, i), "string")
					}
					result.Windows = append(result.Windows, cmdStr)
				}
			}
			if linuxCmds, exists := shellMap["linux"]; exists {
				linuxArray, ok := linuxCmds.([]any)
				if !ok {
					return result, burrito.WrappedErrorf(
						jsonPathTypeError, fmt.Sprintf("%s->linux", key), "array")
				}
				for i, cmd := range linuxArray {
					cmdStr, ok := cmd.(string)
					if !ok {
						return result, burrito.WrappedErrorf(
							jsonPathTypeError, fmt.Sprintf("%s->linux->%d", key, i), "string")
					}
					result.Linux = append(result.Linux, cmdStr)
				}
			}
			if darwinCmds, exists := shellMap["darwin"]; exists {
				darwinArray, ok := darwinCmds.([]any)
				if !ok {
					return result, burrito.WrappedErrorf(
						jsonPathTypeError, fmt.Sprintf("%s->darwin", key), "array")
				}
				for i, cmd := range darwinArray {
					cmdStr, ok := cmd.(string)
					if !ok {
						return result, burrito.WrappedErrorf(
							jsonPathTypeError, fmt.Sprintf("%s->darwin->%d", key, i), "string")
					}
					result.Darwin = append(result.Darwin, cmdStr)
				}
			}
		} else {
			return result, burrito.WrappedErrorf(
				jsonPathTypeError, key, "array or object")
		}
	}
	return result, nil
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
	exportValue, ok := obj["export"]
	if !ok {
		return result, burrito.WrappedErrorf(jsonPathMissingError, "export")
	}
	exportTargets, err := ExportTargetsFromObject(exportValue)
	if err != nil {
		return result, burrito.PassError(err)
	}
	result.ExportTarget = exportTargets
	// PreShell and PostShell
	preShell, err := shellCommandsFromObject(obj, "preShell")
	if err != nil {
		return result, burrito.PassError(err)
	}
	result.PreShell = preShell

	postShell, err := shellCommandsFromObject(obj, "postShell")
	if err != nil {
		return result, burrito.PassError(err)
	}
	result.PostShell = postShell

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
