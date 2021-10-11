package regolith

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/otiai10/copy"
)

// GetExportPaths returns file paths for exporting behavior pack and
// resource pack based on exportTarget (a structure with data related to
// export settings) and the name of the project.
func GetExportPaths(exportTarget ExportTarget, name string) (bpPath string, rpPath string, err error) {
	if exportTarget.Target == "development" {
		comMojang, err := FindMojangDir()
		if err != nil {
			return "", "", wrapError("Failed to find com.mojang directory", err)
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
				return "", "", errors.New("Using both \"worldName\" and \"worldPath\" is not allowed.")
			}
			bpPath = filepath.Join(exportTarget.WorldPath, "behavior_packs", name+"_bp")
			rpPath = filepath.Join(exportTarget.WorldPath, "resource_packs", name+"_rp")
		} else if exportTarget.WorldName != "" {
			dir, err := FindMojangDir()
			if err != nil {
				return "", "", wrapError("Failed to find com.mojang directory", err)
			}
			worlds, err := ListWorlds(dir)
			if err != nil {
				return "", "", wrapError("Failed to list worlds", err)
			}
			for _, world := range worlds {
				if world.Name == exportTarget.WorldName {
					bpPath = filepath.Join(world.Path, "behavior_packs", name+"_bp")
					rpPath = filepath.Join(world.Path, "resource_packs", name+"_rp")
				}
			}
		} else {
			err = errors.New("The \"world\" export target requires either a \"worldName\" or \"worldPath\" property")
		}
	} else if exportTarget.Target == "local" {
		bpPath = "build/BP/"
		rpPath = "build/RP/"
	} else {
		err = errors.New(fmt.Sprintf("Export '%s' target not valid", exportTarget.Target))
	}
	return
}

// ExportProject copies files from the tmp paths (tmp/BP and tmp/RP) into
// the project's export target. The paths are generated with GetExportPaths.

func ExportProject(profile Profile, name string) error {
	exportTarget := profile.ExportTarget
	bpPath, rpPath, err := GetExportPaths(exportTarget, name)
	if err != nil {
		return err
	}

	// Loading edited_files.json or creating empty object
	editedFiles := LoadEditedFiles()
	err = editedFiles.CheckDeletionSafety(rpPath, bpPath)
	if err != nil {
		return errors.New(
			"Exporting project was aborted because it could remove some " +
				"files you want to keep: " + err.Error() + ". If you are trying to run " +
				"Regolith for the first time on this project make sure that the " +
				"export paths are empty. Otherwise, you can check \"" +
				EditedFilesPath + "\" file to see if it contains the full list of " +
				"the files that can be removed.")
	}

	// Clearing output locations
	// Spooky, I hope file protection works, and it won't do any damage
	err = os.RemoveAll(bpPath)
	if err != nil {
		return wrapError("Failed to clear behavior pack build output", err)
	}
	err = os.RemoveAll(rpPath)
	if err != nil {
		return wrapError("Failed to clear resource pack build output", err)
	}
	err = os.RemoveAll(profile.DataPath)
	if err != nil {
		return wrapError("Failed to clear filter data path", err)
	}

	Logger.Info("Exporting project to ", bpPath)
	err = CopyOrMove(".regolith/tmp/BP", bpPath)
	if err != nil {
		return err
	}

	Logger.Info("Exporting project to ", rpPath)
	err = CopyOrMove(".regolith/tmp/RP", rpPath)
	if err != nil {
		return err
	}

	err = CopyOrMove(".regolith/tmp/data", profile.DataPath)
	if err != nil {
		return err
	}

	// Create new edited_files.json
	editedFiles, err = NewEditedFiles(rpPath, bpPath)
	if err != nil {
		return err
	}
	err = editedFiles.Dump()
	return err
}

func CopyOrMove(source string, destination string) error {
	err := os.Rename(source, destination)
	if err != nil { // Rename might fail if output path is on a different drive
		Logger.Infof("Couldn't move files to %s. Trying to copy files instead...", destination)
		err = copy.Copy(source, destination, copy.Options{PreserveTimes: false, Sync: false})
		if err != nil {
			return wrapError(fmt.Sprintf("Couldn't copy data files to %s, aborting.", destination), err)
		}
	}
	return nil
}
