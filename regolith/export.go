package regolith

import (
	"errors"
	"fmt"
	"github.com/otiai10/copy"
	"os"
	"path/filepath"
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
	} else {
		err = errors.New(fmt.Sprintf("Export '%s' target not valid", exportTarget.Target))
	}
	return
}

// ExportProject copies files from the build paths (build/BP and build/RP) into
// the project's export target and its name. The paths are generated with
// GetExportPaths.
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
				"the files that can be reomved.")
	}
	// Clearing output locations
	// Spooky, I hope file protection works, and it won't do any damage
	err = os.RemoveAll(bpPath)
	if err != nil {
		return wrapError("Failed to clear behavior pack build output", err)
	}
	err = os.MkdirAll(bpPath, 0777)
	if err != nil {
		return wrapError("Failed to create required directories for behavior pack build output", err)
	}
	err = os.RemoveAll(rpPath)
	if err != nil {
		return wrapError("Failed to clear resource pack build output", err)
	}
	err = os.MkdirAll(rpPath, 0777)
	if err != nil {
		return wrapError("Failed to create required directories for resource pack build output", err)
	}

	Logger.Info("Exporting project to ", bpPath)
	Logger.Info("Exporting project to ", rpPath)

	err = copy.Copy("build/BP/", bpPath, copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		return wrapError(fmt.Sprintf("Couldn't copy BP files to %s", bpPath), err)
	}

	err = copy.Copy("build/RP/", rpPath, copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		return wrapError(fmt.Sprintf("Couldn't copy RP files to %s", rpPath), err)
	}

	// Create new edited_files.json
	editedFiles, err = NewEditedFiles(rpPath, bpPath)
	if err != nil {
		return err
	}
	err = editedFiles.Dump()
	return err
}
