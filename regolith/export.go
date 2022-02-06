package regolith

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/otiai10/copy"
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
			return "", "", WrapError(err, "failed to find com.mojang directory")
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
				return "", "", WrapError(
					nil, "using both \"worldName\" and \"worldPath\" is not"+
						" allowed")
			}
			bpPath = filepath.Join(
				exportTarget.WorldPath, "behavior_packs", name+"_bp")
			rpPath = filepath.Join(
				exportTarget.WorldPath, "resource_packs", name+"_rp")
		} else if exportTarget.WorldName != "" {
			dir, err := FindMojangDir()
			if err != nil {
				return "", "", WrapError(
					err, "failed to find \"com.mojang\" directory")
			}
			worlds, err := ListWorlds(dir)
			if err != nil {
				return "", "", WrapError(err, "Failed to list worlds")
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
			err = WrapError(
				nil, "the \"world\" export target requires either a "+
					"\"worldName\" or \"worldPath\" property")
		}
	} else if exportTarget.Target == "local" {
		bpPath = "build/BP/"
		rpPath = "build/RP/"
	} else {
		err = WrapErrorf(
			nil, "export target %q is not valid", exportTarget.Target)
	}
	return
}

// ExportProject copies files from the tmp paths (tmp/BP and tmp/RP) into
// the project's export target. The paths are generated with GetExportPaths.
func ExportProject(profile Profile, name string, dataPath string) error {
	exportTarget := profile.ExportTarget
	bpPath, rpPath, err := GetExportPaths(exportTarget, name)
	if err != nil {
		return WrapError(err, "failed to get export paths")
	}

	// Loading edited_files.json or creating empty object
	editedFiles := LoadEditedFiles()
	err = editedFiles.CheckDeletionSafety(rpPath, bpPath)
	if err != nil {
		return WrapError(
			err,
			"The safety check failed, this means that the list of the files\n"+
				"from export path from previous run don't match to the\n"+
				"current state of the folder. Please remove the files from the\n"+
				"export path and try again.")
	}

	// Clearing output locations
	// Spooky, I hope file protection works, and it won't do any damage
	err = os.RemoveAll(bpPath)
	if err != nil {
		return WrapErrorf(
			err, "failed to clear behavior pack build output path: %q", bpPath)
	}
	err = os.RemoveAll(rpPath)
	if err != nil {
		return WrapErrorf(
			err, "failed to clear resource pack build output path: %q", rpPath)
	}
	// TODO - this code is dangerous. You can put any dataPath into the config
	// file and regolith will delete it
	err = os.RemoveAll(dataPath)
	if err != nil {
		return WrapErrorf(
			err, "failed to clear filter data path: %q", dataPath)
	}
	Logger.Info("Exporting project to ", bpPath)
	err = MoveOrCopy(".regolith/tmp/BP", bpPath, exportTarget.ReadOnly, true)
	if err != nil {
		return WrapError(err, "failed to export behavior pack")
	}
	Logger.Info("Exporting project to ", rpPath)
	err = MoveOrCopy(".regolith/tmp/RP", rpPath, exportTarget.ReadOnly, true)
	if err != nil {
		return WrapError(err, "failed to export resource pack")
	}

	err = MoveOrCopy(".regolith/tmp/data", dataPath, false, false)
	if err != nil {
		return WrapError(
			err,
			"failed to move the filter data back to the project's data folder")
	}
	// Update or create edited_files.json
	err = editedFiles.UpdateFromPaths(rpPath, bpPath)
	if err != nil {
		return WrapError(
			err,
			"failed to create a list of files edited by this 'regolith run'")
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
			return WrapErrorf(
				err, "couldn't copy data files to %s, aborting.", destination)
		}
	} else if copyParentAcl { // No errors with moving files but needs ACL copy
		parent := filepath.Dir(destination)
		if _, err := os.Stat(parent); os.IsNotExist(err) {
			return WrapError(
				err,
				"couldn't copy ACLs - parent directory (used as a source of "+
					"ACL data) doesn't exist")
		}
		err = copyFileSecurityInfo(parent, destination)
		if err != nil {
			return WrapErrorf(
				err,
				"counldn't copy ACLs to the target file %s",
				destination,
			)
		}
	}
	// Make files read only if this option is selected
	if makeReadOnly {
		err := filepath.WalkDir(destination,
			func(s string, d fs.DirEntry, e error) error {
				if e != nil {
					return WrapErrorf(
						e, "failed to walk directory %q", destination)
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
