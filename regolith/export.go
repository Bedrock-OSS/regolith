package regolith

import (
	"os"
	"path/filepath"
)

// GetExportPaths returns file paths for exporting behavior pack and
// resource pack based on exportTarget (a structure with data related to
// export settings) and the name of the project.
func GetExportPaths(
	exportTarget ExportTarget, name string,
) (bpPath string, rpPath string, err error) {
	if exportTarget.Target == "development" {
		comMojang, err := FindMojangDir()
		if err != nil {
			return "", "", WrapError(
				err, "Failed to find \"com.mojang\" directory.")
		}

		// TODO - I don't like the _rp and _bp sufixes. Can we get rid of that?
		// I for example always name my packs "0".
		bpPath = comMojang + "/development_behavior_packs/" + name + "_bp"
		rpPath = comMojang + "/development_resource_packs/" + name + "_rp"
	} else if exportTarget.Target == "preview" {
		comMojang, err := FindPreviewDir()
		if err != nil {
			return "", "", WrapError(
				err, "Failed to find preview \"com.mojang\" directory.")
		}

		// TODO - I don't like the _rp and _bp sufixes. Can we get rid of that?
		// I for example always name my packs "0".
		bpPath = comMojang + "/development_behavior_packs/" + name + "_bp"
		rpPath = comMojang + "/development_resource_packs/" + name + "_rp"
	} else if exportTarget.Target == "exact" {
		bpPath = exportTarget.BpPath
		rpPath = exportTarget.RpPath
	} else if exportTarget.Target == "world" {
		if exportTarget.WorldPath != "" {
			if exportTarget.WorldName != "" {
				return "", "", WrappedError(
					"Using both \"worldName\" and \"worldPath\" is not" +
						" allowed.")
			}
			bpPath = filepath.Join(
				exportTarget.WorldPath, "behavior_packs", name+"_bp")
			rpPath = filepath.Join(
				exportTarget.WorldPath, "resource_packs", name+"_rp")
		} else if exportTarget.WorldName != "" {
			dir, err := FindMojangDir()
			if err != nil {
				return "", "", WrapError(
					err, "Failed to find \"com.mojang\" directory.")
			}
			worlds, err := ListWorlds(dir)
			if err != nil {
				return "", "", WrapError(err, "Failed to list worlds.")
			}
			for _, world := range worlds {
				if world.Name == exportTarget.WorldName {
					bpPath = filepath.Join(
						world.Path, "behavior_packs", name+"_bp")
					rpPath = filepath.Join(
						world.Path, "resource_packs", name+"_rp")
				}
			}
		} else {
			err = WrappedError(
				"The \"world\" export target requires either a " +
					"\"worldName\" or \"worldPath\" property")
		}
	} else if exportTarget.Target == "local" {
		bpPath = "build/BP/"
		rpPath = "build/RP/"
	} else {
		err = WrappedErrorf(
			"Export target %q is not valid", exportTarget.Target)
	}
	return
}

// ExportProject copies files from the tmp paths (tmp/BP and tmp/RP) into
// the project's export target. The paths are generated with GetExportPaths.
func ExportProject(
	profile Profile, name, dataPath, dotRegolithPath string,
) error {
	exportTarget := profile.ExportTarget
	bpPath, rpPath, err := GetExportPaths(exportTarget, name)
	if err != nil {
		return WrapError(
			err, "Failed to get generate export paths.")
	}

	// Loading edited_files.json or creating empty object
	editedFiles := LoadEditedFiles(dotRegolithPath)
	err = editedFiles.CheckDeletionSafety(rpPath, bpPath)
	if err != nil {
		return WrapErrorf(
			err,
			"Safety mechanism stopped Regolith to protect unexpected files "+
				"from your export targets.\n"+
				"Did you edit the exported files manually?\n"+
				"Please clear your export paths and try again.\n"+
				"Resource pack export path: %s\n"+
				"Behavior pack export path: %s",
			rpPath, bpPath)
	}

	// Clearing output locations
	// Spooky, I hope file protection works, and it won't do any damage
	err = os.RemoveAll(bpPath)
	if err != nil {
		return WrapErrorf(
			err, "Failed to clear behavior pack from build path %q.\n"+
				"Are user permissions correct?", bpPath)
	}
	err = os.RemoveAll(rpPath)
	if err != nil {
		return WrapErrorf(
			err, "Failed to clear resource pack from build path %q.\n"+
				"Are user permissions correct?", rpPath)
	}
	// The root of the data path cannot be deleted because the
	// "regolith watch" function would stop watching the file changes
	// (due to Windows API limitation).
	paths, err := os.ReadDir(dataPath)
	if err != nil {
		var err1 error = nil
		if os.IsNotExist(err) {
			err1 = os.MkdirAll(dataPath, 0755)
		}
		if err1 != nil {
			return WrapErrorf(
				err, "Failed to read the files from the data path %q",
				dataPath)
		}
	}
	backupPath := filepath.Join(dotRegolithPath, ".dataBackup")
	revertibleOps, err := NewrevertibleFsOperations(backupPath)
	if err != nil {
		return WrapErrorf(err, newRevertibleFsOperationsError, backupPath)
	}
	for _, path := range paths {
		path := filepath.Join(dataPath, path.Name())
		err = revertibleOps.DeleteDir(path)
		if err != nil {
			handlerError := revertibleOps.Undo()
			mainError := WrapErrorf(err, updateSourceFilesError, path)
			if handlerError != nil {
				return WrapErrorHandlerError(
					mainError, handlerError, errorConnector, fsUndoError)
			}
			if handlerError := revertibleOps.Close(); handlerError != nil {
				return PassErrorHandlerError(
					mainError, handlerError, errorConnector)
			}
			return mainError
		}
	}

	Logger.Infof("Exporting behavior pack to \"%s\".", bpPath)
	err = MoveOrCopy(filepath.Join(dotRegolithPath, "tmp/BP"), bpPath, exportTarget.ReadOnly, true)
	if err != nil {
		return WrapError(err, "Failed to export behavior pack.")
	}
	Logger.Infof("Exporting project to \"%s\".", filepath.Clean(rpPath))
	err = MoveOrCopy(filepath.Join(dotRegolithPath, "tmp/RP"), rpPath, exportTarget.ReadOnly, true)
	if err != nil {
		return WrapError(err, "Failed to export resource pack.")
	}
	err = revertibleOps.MoveOrCopyDir(
		filepath.Join(dotRegolithPath, "tmp/data"), dataPath)
	if err != nil {
		handlerError := revertibleOps.Undo()
		mainError := WrapError(
			err, "Failed to move the filter data back to the project's "+
				"data folder.")
		if handlerError != nil {
			return WrapErrorHandlerError(
				mainError, handlerError, errorConnector, fsUndoError)
		}
		if handlerError := revertibleOps.Close(); handlerError != nil {
			return PassErrorHandlerError(
				mainError, handlerError, errorConnector)
		}
		return mainError
	}

	// Update or create edited_files.json
	err = editedFiles.UpdateFromPaths(rpPath, bpPath)
	if err != nil {
		return WrapError(
			err,
			"Failed to create a list of files edited by this 'regolith run'")
	}
	err = editedFiles.Dump(dotRegolithPath)
	if err != nil {
		return WrapError(
			err, "Failed to update the list of the files edited by Regolith."+
				"This may cause the next run to fail.")
	}
	if err := revertibleOps.Close(); err != nil {
		return PassError(err)
	}
	return nil
}

// InplaceExportProject copies the files from the tmp paths (tmp/BP, tmp/RP and
// tmp/data) into the project's source files. It's used by the "regolith tool"
// command. This operation is destructive and cannot be undone.
func InplaceExportProject(
	config *Config, dotRegolithPath string,
) error {
	// Create revertible ops object
	backupPath := filepath.Join(dotRegolithPath, ".dataBackup")
	revertibleOps, err := NewrevertibleFsOperations(backupPath)
	if err != nil {
		return WrapErrorf(err, newRevertibleFsOperationsError, backupPath)
	}
	// Schedule Undo in case of an revertible ops error and schedule Close()
	defer func() {
		if err != nil { // Handle previous error
			Logger.Warnf("Reverting changes...")
			handlerError := revertibleOps.Undo()
			if handlerError != nil {
				err = WrapErrorHandlerError(
					err, handlerError, errorConnector,
					fsUndoError)
				return
			}
			handlerError = revertibleOps.Close()
			if handlerError != nil {
				err = PassErrorHandlerError(
					err, handlerError, errorConnector)
			}
		} else { // No previous error but Close() must be called
			err = revertibleOps.Close()
			if err != nil {
				err = PassError(err)
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
				err = WrapErrorf(
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
			err = revertibleOps.MoveOrCopy(source, target, true)
			if err != nil {
				err = WrapErrorf(
					err, moveOrCopyError, source, target)
				return err // Overwritten by defer
			}
		}
	}
	return err // Can be altered by defer
}
