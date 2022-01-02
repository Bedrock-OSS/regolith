package regolith

import (
	"errors"
	"fmt"
	"io/fs"
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
			err.Error() +
				". Exporting to the target directory would delete all of its files. " +
				"If you are trying to run Regolith for the first time on this" +
				" project make sure that the export paths are empty. " +
				"Otherwise, you can check \"" +
				EditedFilesPath + "\" to see if it contains the full " +
				"list of the files that can be removed.")
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
	err = MoveOrCopy(".regolith/tmp/BP", bpPath, exportTarget.ReadOnly, true)
	if err != nil {
		return err
	}

	Logger.Info("Exporting project to ", rpPath)
	err = MoveOrCopy(".regolith/tmp/RP", rpPath, exportTarget.ReadOnly, true)
	if err != nil {
		return err
	}

	err = MoveOrCopy(".regolith/tmp/data", profile.DataPath, false, false)
	if err != nil {
		return err
	}

	// Update or create edited_files.json
	err = editedFiles.UpdateFromPaths(rpPath, bpPath)
	if err != nil {
		return err
	}
	err = editedFiles.Dump()
	return err
}

// MoveOrCopy tries to move the the source to destination first and in case
// of failore it copies the files instead.
func MoveOrCopy(
	source string, destination string, makeReadOnly bool, copyParentAcl bool,
) error {
	if err := os.Rename(source, destination); err != nil {
		Logger.Infof("Couldn't move files to %s. Trying to copy files instead...", destination)
		copyOptions := copy.Options{PreserveTimes: false, Sync: false}
		err := copy.Copy(source, destination, copyOptions)
		if err != nil {
			return wrapError(fmt.Sprintf("Couldn't copy data files to %s, aborting.", destination), err)
		}
	} else if copyParentAcl { // No errors with moving files but needs ACL copy
		parent := filepath.Dir(destination)
		if _, err := os.Stat(parent); os.IsNotExist(err) {
			return err
		}
		err = copyFileSecurityInfo(parent, destination)
		if err != nil {
			return wrapError(
				fmt.Sprintf(
					"Counldn't set ACL to the target file %s, aborting.",
					destination),
				err,
			)
		}
	}
	// Make files read only if this option is selected
	if makeReadOnly {
		err := filepath.WalkDir(destination,
			func(s string, d fs.DirEntry, e error) error {
				if e != nil {
					return e
				}
				if !d.IsDir() {
					os.Chmod(s, 0444)
				}
				return nil
			})
		if err != nil {
			Logger.Warnf("Unable to change file permissions of %q into read-only", destination)
		}
	}
	return nil
}
