package regolith

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"golang.org/x/mod/semver"
)

// GetExportPaths returns file paths for exporting behavior pack and
// resource pack based on exportTarget (a structure with data related to
// export settings) and the name of the project.
func GetExportPaths(
	exportTarget ExportTarget, ctx RunContext,
) (bpPath string, rpPath string, err error) {
	bpName, rpName, err := GetExportNames(exportTarget, ctx)
	if err != nil {
		return "", "", burrito.WrapError(
			err, "Failed to get the export names.")
	}
	vFormatVersion := "v" + ctx.Config.FormatVersion

	if semver.Compare(vFormatVersion, "v1.4.0") < 0 {
		bpPath, rpPath, err = getExportPathsV1_2_0(
			exportTarget, bpName, rpName)
	} else if semver.Compare(vFormatVersion, "v1.4.0") == 0 {
		bpPath, rpPath, err = getExportPathsV1_4_0(
			exportTarget, bpName, rpName)
	} else {
		err = burrito.WrappedErrorf(
			incompatibleFormatVersionError,
			ctx.Config.FormatVersion, latestCompatibleVersion)
	}
	return
}

func FindMojangDir(build string, pathType ComMojangPathType) (string, error) {
	switch build {
	case "standard":
		return FindStandardMojangDir(pathType)
	case "preview":
		return FindPreviewDir(pathType)
	case "education":
		return FindEducationDir()
		// WARNING: If for some reason we will expand this in the future to
		// match a new format version, we need to split this into versioned
		// functions.
	default:
		return "", burrito.WrappedErrorf(
			invalidExportPathError,
			// current value; valid values
			build, "standard, preview, education")
	}
}

// getExportPathsV1_2_0 handles GetExportPaths for Regolith format versions
// below 1.4.0.
func getExportPathsV1_2_0(
	exportTarget ExportTarget, bpName string, rpName string,
) (bpPath string, rpPath string, err error) {
	switch exportTarget.Target {
	case "development":
		comMojang, err := FindStandardMojangDir(PacksPath)
		if err != nil {
			return "", "", burrito.WrapError(
				err, findMojangDirError)
		}
		return GetDevelopmentExportPaths(bpName, rpName, comMojang)
	case "preview":
		comMojang, err := FindPreviewDir(PacksPath)
		if err != nil {
			return "", "", burrito.WrapError(
				err, findPreviewDirError)
		}
		return GetDevelopmentExportPaths(bpName, rpName, comMojang)
	case "exact":
		return GetExactExportPaths(exportTarget)
	case "world":
		return GetWorldExportPaths(
			exportTarget.WorldPath,
			exportTarget.WorldName,
			"standard",
			bpName, rpName)
	case "local":
		bpPath = "build/" + bpName + "/"
		rpPath = "build/" + rpName + "/"
	case "none":
		bpPath = ""
		rpPath = ""
	default:
		err = burrito.WrappedErrorf(
			"Export target %q is not valid", exportTarget.Target)
	}
	return
}

// getExportPathsV1_4_0 handles GetExportPaths for Regolith format version
// 1.4.0.
func getExportPathsV1_4_0(
	exportTarget ExportTarget, bpName string, rpName string,
) (bpPath string, rpPath string, err error) {
	switch exportTarget.Target {
	case "development":
		comMojang, err := FindMojangDir(exportTarget.Build, PacksPath)
		if err != nil {
			return "", "", burrito.PassError(err)
		}
		return GetDevelopmentExportPaths(bpName, rpName, comMojang)
	case "world":
		return GetWorldExportPaths(
			exportTarget.WorldPath,
			exportTarget.WorldName,
			exportTarget.Build,
			bpName, rpName)
	case "exact":
		return GetExactExportPaths(exportTarget)
	case "local":
		bpPath = "build/" + bpName + "/"
		rpPath = "build/" + rpName + "/"
	case "none":
		bpPath = ""
		rpPath = ""
	default:
		err = burrito.WrappedErrorf(
			"Export target %q is not valid", exportTarget.Target)
	}
	return
}

func GetDevelopmentExportPaths(bpName, rpName, comMojang string) (bpPath string, rpPath string, err error) {
	bpPath = comMojang + "/development_behavior_packs/" + bpName
	rpPath = comMojang + "/development_resource_packs/" + rpName
	return
}

func GetExactExportPaths(exportTarget ExportTarget) (bpPath string, rpPath string, err error) {
	bpPath, err = ResolvePath(exportTarget.BpPath)
	if err != nil {
		return "", "", burrito.WrapError(
			err, "Failed to resolve behavior pack path.")
	}
	rpPath, err = ResolvePath(exportTarget.RpPath)
	if err != nil {
		return "", "", burrito.WrapError(
			err, "Failed to resolve resource pack path.")
	}
	return
}

func GetWorldExportPaths(
	worldPath, worldName, build, bpName, rpName string,
) (bpPath string, rpPath string, err error) {
	if worldPath != "" {
		if worldName != "" {
			return "", "", burrito.WrappedError(
				"Using both \"worldName\" and \"worldPath\" is not" +
					" allowed.")
		}
		wPath, err := ResolvePath(worldPath)
		if err != nil {
			return "", "", burrito.WrapError(
				err, "Failed to resolve world path.")
		}
		bpPath = filepath.Join(
			wPath, "behavior_packs", bpName)
		rpPath = filepath.Join(
			wPath, "resource_packs", rpName)
	} else if worldName != "" {
		dir, err := FindMojangDir(build, WorldPath)
		if err != nil {
			return "", "", burrito.WrapError(
				err, "Failed to find \"com.mojang\" directory.")
		}
		worlds, err := ListWorlds(dir)
		if err != nil {
			return "", "", burrito.WrapError(err, "Failed to list worlds.")
		}
		for _, world := range worlds {
			if world.Name != worldName {
				continue
			}
			bpPath = filepath.Join(
				world.Path, "behavior_packs", bpName)
			rpPath = filepath.Join(
				world.Path, "resource_packs", rpName)
			return bpPath, rpPath, nil
		}
		return "", "", burrito.WrappedErrorf(
			"Failed to find the world.\n"+
				"World name: %s", worldName)
	} else {
		err = burrito.WrappedError(
			"The \"world\" export target requires either a " +
				"\"worldName\" or \"worldPath\" property")
	}
	return
}

// GetExportNames returns the names for the behavior pack and resource pack
// based on the evaluated values of the "bpName" and "rpName" from the
// exportTarget object.
func GetExportNames(exportTarget ExportTarget, ctx RunContext) (bpName string, rpName string, err error) {
	if exportTarget.BpName != "" {
		bpName, err = EvalString(exportTarget.BpName, ctx)
		if err != nil {
			return "", "", burrito.WrapError(
				err, "Failed to evaluate behavior pack name.")
		}
	} else {
		bpName = ctx.Config.Name + "_bp"
	}
	if exportTarget.RpName != "" {
		rpName, err = EvalString(exportTarget.RpName, ctx)
		if err != nil {
			return "", "", burrito.WrapError(
				err, "Failed to evaluate resource pack name.")
		}
	} else {
		rpName = ctx.Config.Name + "_rp"
	}
	return
}

// ExportProject copies files from the tmp paths (tmp/BP and tmp/RP) into
// the project's export target. The paths are generated with GetExportPaths.
func ExportProject(ctx RunContext) error {
	MeasureStart("Export - GetExportPaths")
	profile, err := ctx.GetProfile()
	if err != nil {
		return burrito.WrapError(err, runContextGetProfileError)
	}
	if profile.ExportTarget.Target == "none" {
		Logger.Debugf("Export target is set to \"none\". Skipping export.")
		return nil
	}
	// Get the necessary paths and variables
	dotRegolithPath := ctx.DotRegolithPath
	exportTarget := profile.ExportTarget
	bpPath, rpPath, err := GetExportPaths(exportTarget, ctx)
	if err != nil {
		return burrito.WrapError(
			err, getExportPathsError)
	}
	// Load edited files
	MeasureStart("Export - CheckDeletionSafety")
	editedFiles := LoadEditedFiles(dotRegolithPath)
	err = editedFiles.CheckDeletionSafety(rpPath, bpPath)
	if err != nil {
		return burrito.WrapErrorf(
			err,
			checkDeletionSafetyError,
			rpPath, bpPath)
	}
	// Export RP and BP if necessary
	if IsExperimentEnabled(SymlinkExport) {
		Logger.Debugf("SymlinkExport experiment is enabled. Skipping RP and BP export.")
	} else {
		err = exportProjectRpAndBp(profile, rpPath, bpPath, ctx)
		if err != nil {
			return burrito.PassError(err)
		}
	}
	// Export data for exportData filters
	MeasureStart("Export - ExportData")
	err = exportProjectData(profile, ctx)
	if err != nil {
		return burrito.PassError(err)
	}
	MeasureStart("Export - EditedFiles.UpdateFromPaths")
	// Update or create edited_files.json
	err = editedFiles.UpdateFromPaths(rpPath, bpPath)
	if err != nil {
		return burrito.WrapError(
			err,
			"Failed to create a list of files edited by this 'regolith run'")
	}
	err = editedFiles.Dump(dotRegolithPath)
	if err != nil {
		return burrito.WrapError(err, updatedFilesDumpError)
	}
	// Remove the exported pack paths if they're empty
	if !IsExperimentEnabled(SymlinkExport) {
		MeasureStart("Export - Remove Empty Export Paths")
		for _, packPath := range []string{rpPath, bpPath} {
			pathEmpty, _ := IsDirEmpty(packPath)
			if pathEmpty {
				if err := os.Remove(packPath); err != nil {
					Logger.Warnf(
						"Failed to remove empty pack directory.\n"+
							"Path: %s\n"+
							"Error: %v", packPath, err)
				}
			}
		}
	}
	MeasureEnd()
	return nil
}

// exportProjectRpAndBp is a helper function for ExportProject. It exports the 'rp'
// and 'bp' folders to the target location. This assumes that the symlinkExport
// is disabled.
func exportProjectRpAndBp(profile Profile, rpPath, bpPath string, ctx RunContext) error {
	dotRegolithPath := ctx.DotRegolithPath
	exportTarget := profile.ExportTarget

	var err error
	// When comparing the size and modification time of the files, we need to
	// keep the files in target paths.
	if !IsExperimentEnabled(SizeTimeCheck) {
		// Clearing output locations
		MeasureStart("Export - Clean")
		err = os.RemoveAll(bpPath)
		if err != nil {
			return burrito.WrapErrorf(
				err, "Failed to clear behavior pack from build path %q.\n"+
					"Are user permissions correct?", bpPath)
		}
		err = os.RemoveAll(rpPath)
		if err != nil {
			return burrito.WrapErrorf(
				err, "Failed to clear resource pack from build path %q.\n"+
					"Are user permissions correct?", rpPath)
		}
	}
	MeasureStart("Export - MoveOrCopy")
	var wg sync.WaitGroup
	packsData := []struct {
		packPath string
		tmpPath  string
		packType string
	}{
		{bpPath, "tmp/BP", "behavior"},
		{rpPath, "tmp/RP", "resource"},
	}
	errChan := make(chan error, len(packsData))
	for _, packData := range packsData {
		packPath, tmpPath, packType := packData.packPath, packData.tmpPath, packData.packType
		wg.Go(func() {
			Logger.Infof("Exporting %s pack to \"%s\".", packType, packPath)
			var e error
			if IsExperimentEnabled(SizeTimeCheck) {
				e = SyncDirectories(filepath.Join(dotRegolithPath, tmpPath), packPath, exportTarget.ReadOnly)
			} else {
				e = MoveOrCopy(filepath.Join(dotRegolithPath, tmpPath), packPath, exportTarget.ReadOnly, true)
			}
			if e != nil {
				errChan <- burrito.WrapErrorf(e, "Failed to export %s pack.", packType)
				return
			}
			errChan <- nil
		})
	}

	wg.Wait()
	close(errChan)
	for e := range errChan {
		if e != nil {
			return e
		}
	}
	return nil
}

// exportProjectData is a helper function for ExportProject. It exports the 'data'
// folder back to the project's source files for the filters that opted-in for
// that with exportProjectData option.
func exportProjectData(profile Profile, ctx RunContext) error {
	dataPath := ctx.Config.DataPath
	dotRegolithPath := ctx.DotRegolithPath
	// List the names of the filters that opt-in to the data export process
	var exportedFilterNames []string
	err := profile.ForeachFilter(ctx, func(filter FilterRunner) error {
		usingDataPath, err := filter.IsUsingDataExport(dotRegolithPath, ctx)
		if err != nil {
			return burrito.WrapErrorf(
				err,
				"Failed to check if filter is using data export.\n"+
					"Path: %s", filter.GetId())
		}
		if usingDataPath {
			// Make sure that the filter name isn't a path that tries to access
			// files outside of the data path.
			filterName := filter.GetId()
			for _, forbidden := range []string{"..", "/", "\\", ":"} {
				if strings.Contains(filterName, forbidden) {
					// Other cases should be handled by mkdirAll
					return burrito.WrappedErrorf(
						"Filter name %q contains %q which is not allowed.",
						filterName, forbidden)
				}
			}
			// Add the filter name to the list of paths to export
			exportedFilterNames = append(exportedFilterNames, filter.GetId())
		}
		return nil
	}, true)
	if err != nil {
		return burrito.WrapError(err, "Failed to walk the list of the filters.")
	}
	// The root of the data path cannot be deleted because the
	// "regolith watch" function would stop watching the file changes
	// (due to Windows API limitation).
	_, err = os.ReadDir(dataPath)
	if err != nil {
		var err1 error = nil
		if os.IsNotExist(err) {
			err1 = os.MkdirAll(dataPath, 0755)
		}
		if err1 != nil {
			return burrito.WrapErrorf(err, osReadDirError, dataPath)
		}
	}
	// Create revertible operations object
	backupPath := filepath.Join(dotRegolithPath, ".dataBackup")
	revertibleOps, err := NewRevertibleFsOperations(backupPath)
	if err != nil {
		return burrito.WrapErrorf(err, newRevertibleFsOperationsError, backupPath)
	}
	// Export data
	for _, exportedFilterName := range exportedFilterNames {
		// Clear export target
		targetPath := filepath.Join(dataPath, exportedFilterName)
		if _, err := os.Stat(targetPath); err == nil {
			err = revertibleOps.Delete(targetPath)
			if err != nil {
				handlerError := revertibleOps.Undo()
				mainError := burrito.WrapErrorf(err, updateSourceFilesError, targetPath)
				if handlerError != nil {
					return burrito.GroupErrors(mainError, burrito.WrapError(handlerError, fsUndoError))
				}
				if handlerError := revertibleOps.Close(); handlerError != nil {
					return burrito.GroupErrors(mainError, handlerError)
				}
				return mainError
			}
		} else if os.IsNotExist(err) {
			err = os.MkdirAll(targetPath, 0755)
			if err != nil {
				return burrito.WrapErrorf(err, osMkdirError, targetPath)
			}
		} else {
			return burrito.WrapErrorf(err, osStatErrorAny, targetPath)
		}
		sourcePath := filepath.Join(dotRegolithPath, "tmp/data", exportedFilterName)
		// If source path doesn't exist, skip
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			continue
		}
		// Copy data
		err = revertibleOps.MoveOrCopyDir(sourcePath, targetPath)
		if err != nil {
			handlerError := revertibleOps.Undo()
			mainError := burrito.WrapErrorf(err, moveOrCopyError, sourcePath, targetPath)
			if handlerError != nil {
				return burrito.GroupErrors(mainError, burrito.WrapError(handlerError, fsUndoError))
			}
			if handlerError := revertibleOps.Close(); handlerError != nil {
				return burrito.GroupErrors(mainError, handlerError)
			}
			return mainError
		}
	}
	if err := revertibleOps.Close(); err != nil {
		return burrito.PassError(err)
	}
	return nil
}

// InplaceExportProject copies the files from the tmp paths (tmp/BP, tmp/RP and
// tmp/data) into the project's source files. It's used by the "regolith apply-filter"
// command. This operation is destructive and cannot be undone.
func InplaceExportProject(
	config *Config, dotRegolithPath string,
) (err error) {
	// Create revertible ops object
	backupPath := filepath.Join(dotRegolithPath, ".dataBackup")
	revertibleOps, err := NewRevertibleFsOperations(backupPath)
	if err != nil {
		return burrito.WrapErrorf(err, newRevertibleFsOperationsError, backupPath)
	}
	// Schedule Undo in case of a revertible ops error and schedule Close()
	defer func() {
		if err != nil { // Handle previous error
			Logger.Warnf("Reverting changes...")
			handlerError := revertibleOps.Undo()
			if handlerError != nil {
				err = burrito.GroupErrors(err, burrito.WrapError(handlerError, fsUndoError))
				return
			}
			handlerError = revertibleOps.Close()
			if handlerError != nil {
				err = burrito.GroupErrors(err, handlerError)
			}
		} else { // No previous error but Close() must be called
			err = revertibleOps.Close()
			if err != nil {
				err = burrito.PassError(err)
			}
		}
	}()
	// Delete RP, BP and data before replacing them with files from tmp
	deleteDirs := []string{
		config.ResourceFolder, config.BehaviorFolder, config.DataPath}
	for _, deleteDir := range deleteDirs {
		if deleteDir != "" {
			err = revertibleOps.Delete(deleteDir)
			if err != nil {
				err = burrito.WrapErrorf(
					err, updateSourceFilesError, deleteDir)
				return err // Overwritten by defer
			}
		}
	}
	// Move files from tmp to RP, BP and data
	moveFiles := [][2]string{
		{filepath.Join(dotRegolithPath, "tmp/RP"), config.ResourceFolder},
		{filepath.Join(dotRegolithPath, "tmp/BP"), config.BehaviorFolder},
		{filepath.Join(dotRegolithPath, "tmp/data"), config.DataPath},
	}
	for _, moveFile := range moveFiles {
		source, target := moveFile[0], moveFile[1]
		if source != "" {
			err = revertibleOps.MoveOrCopyDir(source, target)
			if err != nil {
				err = burrito.WrapErrorf(
					err, moveOrCopyError, source, target)
				return err // Overwritten by defer
			}
		}
	}
	return err // Can be altered by defer
}
