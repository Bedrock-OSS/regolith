package regolith

import (
	"io/ioutil"
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

// RecycledExportProject copies files from the tmp paths (tmp/BP and tmp/RP)
// into the project's export target. The paths are generated with
// GetExportPaths. The function uses cached data about the state of the project
// files to reduce the number of file system operations.
func RecycledExportProject(profile Profile, name string, dataPath string) error {
	exportTarget := profile.ExportTarget
	bpPath, rpPath, err := GetExportPaths(exportTarget, name)
	if err != nil {
		return WrapError(
			err, "Failed to get generate export paths.")
	}

	// Loading edited_files.json or creating empty object
	editedFiles := LoadEditedFiles()
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

	Logger.Infof("Exporting behavior pack to \"%s\".", bpPath)
	// err = MoveOrCopy(".regolith/tmp/BP", bpPath, exportTarget.ReadOnly, true)
	err = FullRecycledMoveOrCopy(
		".regolith/tmp/BP", bpPath,
		RecycledMoveOrCopySettings{
			canMove:                 true,
			saveSourceHashes:        true,
			saveTargetHashes:        true,
			makeTargetReadOnly:      exportTarget.ReadOnly,
			copyTargetAclFromParent: true,
			reloadSourceHashes:      true,
		})
	if err != nil {
		return WrapError(err, "Failed to export behavior pack.")
	}
	Logger.Infof("Exporting project to \"%s\".", rpPath)
	// err = MoveOrCopy(".regolith/tmp/RP", rpPath, exportTarget.ReadOnly, true)
	err = FullRecycledMoveOrCopy(
		".regolith/tmp/RP", rpPath,
		RecycledMoveOrCopySettings{
			canMove:                 true,
			saveSourceHashes:        true,
			saveTargetHashes:        true,
			makeTargetReadOnly:      exportTarget.ReadOnly,
			copyTargetAclFromParent: true,
			reloadSourceHashes:      true,
		})
	if err != nil {
		return WrapError(err, "Failed to export resource pack.")
	}
	// err = MoveOrCopy(".regolith/tmp/data", dataPath, false, false)
	err = FullRecycledMoveOrCopy(
		".regolith/tmp/data", dataPath,
		RecycledMoveOrCopySettings{
			canMove:                 true,
			saveSourceHashes:        true,
			saveTargetHashes:        false,
			makeTargetReadOnly:      false,
			copyTargetAclFromParent: false,
			reloadSourceHashes:      true,
		})
	if err != nil {
		return WrapError(
			err, "Failed to move the filter data back to the project's "+
				"data folder.")
	}

	// Update or create edited_files.json
	err = editedFiles.UpdateFromPaths(rpPath, bpPath)
	if err != nil {
		return WrapError(
			err,
			"Failed to create a list of files edited by this 'regolith run'")
	}
	err = editedFiles.Dump()
	if err != nil {
		return WrapError(
			err, "Failed to update the list of the files edited by Regolith."+
				"This may cause the next run to fail.")
	}
	return nil
}

// ExportProject copies files from the tmp paths (tmp/BP and tmp/RP) into
// the project's export target. The paths are generated with GetExportPaths.
func ExportProject(profile Profile, name string, dataPath string) error {
	exportTarget := profile.ExportTarget
	bpPath, rpPath, err := GetExportPaths(exportTarget, name)
	if err != nil {
		return WrapError(
			err, "Failed to get generate export paths.")
	}

	// Loading edited_files.json or creating empty object
	editedFiles := LoadEditedFiles()
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
	files, err := ioutil.ReadDir(dataPath)
	if err != nil {
		return WrapErrorf(
			err, "Failed to read the files from the data path %q",
			dataPath)
	}
	for _, file := range files {
		filePath := filepath.Join(dataPath, file.Name())
		err = os.RemoveAll(filePath)
		if err != nil {
			return WrapErrorf(
				err, "Failed to clear filter path from data path %q.",
				filePath)
		}
	}

	Logger.Infof("Exporting behavior pack to \"%s\".", bpPath)
	err = MoveOrCopy(".regolith/tmp/BP", bpPath, exportTarget.ReadOnly, true)
	if err != nil {
		return WrapError(err, "Failed to export behavior pack.")
	}
	Logger.Infof("Exporting project to \"%s\".", rpPath)
	err = MoveOrCopy(".regolith/tmp/RP", rpPath, exportTarget.ReadOnly, true)
	if err != nil {
		return WrapError(err, "Failed to export resource pack.")
	}
	err = MoveOrCopy(".regolith/tmp/data", dataPath, false, false)
	if err != nil {
		return WrapError(
			err, "Failed to move the filter data back to the project's "+
				"data folder.")
	}

	// Update or create edited_files.json
	err = editedFiles.UpdateFromPaths(rpPath, bpPath)
	if err != nil {
		return WrapError(
			err,
			"Failed to create a list of files edited by this 'regolith run'")
	}
	err = editedFiles.Dump()
	if err != nil {
		return WrapError(
			err, "Failed to update the list of the files edited by Regolith."+
				"This may cause the next run to fail.")
	}
	return nil
}
