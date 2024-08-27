package regolith

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Bedrock-OSS/go-burrito/burrito"
)

// GetExportPaths returns file paths for exporting behavior pack and
// resource pack based on exportTarget (a structure with data related to
// export settings) and the name of the project.
func GetExportPaths(
	exportTarget ExportTarget, ctx RunContext,
) (bpPath string, rpPath string, err error) {
	bpName, rpName, err := GetExportNames(exportTarget, ctx)
	if exportTarget.Target == "development" {
		comMojang, err := FindMojangDir()
		if err != nil {
			return "", "", burrito.WrapError(
				err, "Failed to find \"com.mojang\" directory.")
		}
		return GetDevelopmentExportPaths(bpName, rpName, comMojang)
	} else if exportTarget.Target == "preview" {
		comMojang, err := FindPreviewDir()
		if err != nil {
			return "", "", burrito.WrapError(
				err, "Failed to find preview \"com.mojang\" directory.")
		}
		return GetDevelopmentExportPaths(bpName, rpName, comMojang)
	} else if exportTarget.Target == "exact" {
		return GetExactExportPaths(exportTarget)
	} else if exportTarget.Target == "world" {
		return GetWorldExportPaths(exportTarget, bpName, rpName)
	} else if exportTarget.Target == "local" {
		bpPath = "build/" + bpName + "/"
		rpPath = "build/" + rpName + "/"
	} else if exportTarget.Target == "none" {
		bpPath = ""
		rpPath = ""
	} else {
		err = burrito.WrappedErrorf(
			"Export target %q is not valid", exportTarget.Target)
	}
	return
}

func GetDevelopmentExportPaths(bpName, rpName, comMojang string) (bpPath string, rpPath string, err error) {
	if err != nil {
		return "", "", burrito.WrapError(
			err, "Failed to find \"com.mojang\" directory.")
	}

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

func GetWorldExportPaths(exportTarget ExportTarget, bpName, rpName string) (bpPath string, rpPath string, err error) {
	if exportTarget.WorldPath != "" {
		if exportTarget.WorldName != "" {
			return "", "", burrito.WrappedError(
				"Using both \"worldName\" and \"worldPath\" is not" +
					" allowed.")
		}
		wPath, err := ResolvePath(exportTarget.WorldPath)
		if err != nil {
			return "", "", burrito.WrapError(
				err, "Failed to resolve world path.")
		}
		bpPath = filepath.Join(
			wPath, "behavior_packs", bpName)
		rpPath = filepath.Join(
			wPath, "resource_packs", rpName)
	} else if exportTarget.WorldName != "" {
		dir, err := FindMojangDir()
		if err != nil {
			return "", "", burrito.WrapError(
				err, "Failed to find \"com.mojang\" directory.")
		}
		worlds, err := ListWorlds(dir)
		if err != nil {
			return "", "", burrito.WrapError(err, "Failed to list worlds.")
		}
		for _, world := range worlds {
			if world.Name == exportTarget.WorldName {
				bpPath = filepath.Join(
					world.Path, "behavior_packs", bpName)
				rpPath = filepath.Join(
					world.Path, "resource_packs", rpName)
			}
		}
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
	dataPath := ctx.Config.DataPath
	dotRegolithPath := ctx.DotRegolithPath
	// Get the export target paths
	exportTarget := profile.ExportTarget
	bpPath, rpPath, err := GetExportPaths(exportTarget, ctx)
	if err != nil {
		return burrito.WrapError(
			err, "Failed to get generate export paths.")
	}

	MeasureStart("Export - LoadEditedFiles")
	// Loading edited_files.json or creating empty object
	editedFiles := LoadEditedFiles(dotRegolithPath)
	err = editedFiles.CheckDeletionSafety(rpPath, bpPath)
	if err != nil {
		return burrito.WrapErrorf(
			err,
			"Safety mechanism stopped Regolith to protect unexpected files "+
				"from your export targets.\n"+
				"Did you edit the exported files manually?\n"+
				"Please clear your export paths and try again.\n"+
				"Resource pack export path: %s\n"+
				"Behavior pack export path: %s",
			rpPath, bpPath)
	}

	MeasureStart("Export - Clean")
	// When comparing the size and modification time of the files, we need to
	// keep the files in target paths.
	if !IsExperimentEnabled(SizeTimeCheck) {
		// Clearing output locations
		// Spooky, I hope file protection works, and it won't do any damage
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
	MeasureEnd()
	// List the names of the filters that opt-in to the data export process
	var exportedFilterNames []string
	for filter := range profile.Filters {
		filter := profile.Filters[filter]
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
	MeasureStart("Export - RevertibleOps")
	// Create revertible operations object
	backupPath := filepath.Join(dotRegolithPath, ".dataBackup")
	revertibleOps, err := NewRevertibleFsOperations(backupPath)
	if err != nil {
		return burrito.WrapErrorf(err, newRevertibleFsOperationsError, backupPath)
	}
	// Export data
	MeasureStart("Export - ExportData")
	for _, exportedFilterName := range exportedFilterNames {
		// Clear export target
		targetPath := filepath.Join(dataPath, exportedFilterName)
		if _, err := os.Stat(targetPath); err == nil {
			err = revertibleOps.DeleteDir(targetPath)
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
	MeasureStart("Export - MoveOrCopy")
	if IsExperimentEnabled(SizeTimeCheck) {
		// Export BP
		Logger.Infof("Exporting behavior pack to \"%s\".", bpPath)
		err = SyncDirectories(filepath.Join(dotRegolithPath, "tmp/BP"), bpPath, exportTarget.ReadOnly)
		if err != nil {
			return burrito.WrapError(err, "Failed to export behavior pack.")
		}
		// Export RP
		Logger.Infof("Exporting project to \"%s\".", filepath.Clean(rpPath))
		err = SyncDirectories(filepath.Join(dotRegolithPath, "tmp/RP"), rpPath, exportTarget.ReadOnly)
		if err != nil {
			return burrito.WrapError(err, "Failed to export resource pack.")
		}
	} else {
		// Export BP
		Logger.Infof("Exporting behavior pack to \"%s\".", bpPath)
		err = MoveOrCopy(filepath.Join(dotRegolithPath, "tmp/BP"), bpPath, exportTarget.ReadOnly, true)
		if err != nil {
			return burrito.WrapError(err, "Failed to export behavior pack.")
		}
		// Export RP
		Logger.Infof("Exporting project to \"%s\".", filepath.Clean(rpPath))
		err = MoveOrCopy(filepath.Join(dotRegolithPath, "tmp/RP"), rpPath, exportTarget.ReadOnly, true)
		if err != nil {
			return burrito.WrapError(err, "Failed to export resource pack.")
		}
	}
	MeasureStart("Export - UpdateFromPaths")
	// Update or create edited_files.json
	err = editedFiles.UpdateFromPaths(rpPath, bpPath)
	if err != nil {
		return burrito.WrapError(
			err,
			"Failed to create a list of files edited by this 'regolith run'")
	}
	err = editedFiles.Dump(dotRegolithPath)
	if err != nil {
		return burrito.WrapError(
			err, "Failed to update the list of the files edited by Regolith."+
				"This may cause the next run to fail.")
	}
	if err := revertibleOps.Close(); err != nil {
		return burrito.PassError(err)
	}
	MeasureEnd()
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
			err = revertibleOps.DeleteDir(deleteDir)
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
