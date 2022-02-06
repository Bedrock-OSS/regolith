package regolith

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
)

const EditedFilesPath = ".regolith/cache/edited_files.json"

// PathList is an alias for []string. It's used to store a list of file paths.
type filesList = []string

// EditedFiles is used to load edited_files.json from cache in order
// to check if the files are safe to delete.
type EditedFiles struct {
	Rp map[string]filesList `json:"rp"`
	Bp map[string]filesList `json:"bp"`
}

// LoadEditedFiles data from edited_files.json or returns an empty object
// if file doesn't exist.
func LoadEditedFiles() EditedFiles {
	data, err := os.ReadFile(EditedFilesPath)
	if err != nil {
		return NewEditedFiles()
	}
	result := NewEditedFiles()
	err = json.Unmarshal(data, &result)
	if err != nil {
		return NewEditedFiles()
	}
	return result
}

// Dump dumps EditedFiles to EditedFilesPath in JSON format.
func (f *EditedFiles) Dump() error {
	result, err := json.MarshalIndent(f, "", "\t")
	if err != nil {
		return WrapError(err, "failed to marshal edited files list JSON")
	}
	// Create parent directory of EditedFilesPath
	parentDir := filepath.Dir(EditedFilesPath)
	err = os.MkdirAll(parentDir, 0666)
	if err != nil {
		return WrapErrorf(
			err, "failed to create %q directory for edited files list",
			parentDir)
	}
	err = os.WriteFile(EditedFilesPath, result, 0666)
	if err != nil {
		return WrapErrorf(
			err, "failed to save edited files list in %q", EditedFilesPath)
	}
	return nil
}

// CheckDeletionSafety checks whether it's safe to delete files from rpPath and
// bpPath based on the lists of removeable files from EditedFiles object.
func (f *EditedFiles) CheckDeletionSafety(rpPath string, bpPath string) error {
	files, ok := f.Rp[rpPath]
	if !ok {
		files = make([]string, 0)
	}
	err := checkDeletionSafety(rpPath, files)
	if err != nil {
		return WrapErrorf(
			err,
			"the resource pack files from %q has been modified and "+
				"cannot be deleted",
			rpPath)
	}
	files, ok = f.Bp[bpPath]
	if !ok {
		files = make([]string, 0)
	}
	err = checkDeletionSafety(bpPath, files)
	if err != nil {
		return WrapErrorf(
			err,
			"the behavior pack files from %q has been modified and"+
				" cannot be deleted",
			bpPath)
	}
	return nil
}

func (f *EditedFiles) UpdateFromPaths(rpPath string, bpPath string) error {
	rpFiles, err := listFiles(rpPath)
	if err != nil {
		return WrapErrorf(
			err, "failed to list resource pack files from %q", rpPath)
	}
	bpFiles, err := listFiles(bpPath)
	if err != nil {
		return WrapErrorf(
			err, "failed to list behavior pack files from %q", bpPath)
	}
	f.Rp[rpPath] = rpFiles
	f.Bp[bpPath] = bpFiles
	return nil
}

// NewEditedFiles creates new EditedFiles object with lists of the files from
// rpPath and bpPath.
func NewEditedFiles() EditedFiles {
	var result EditedFiles
	result.Rp = make(map[string]filesList)
	result.Bp = make(map[string]filesList)
	return result
}

// listFiles returns a slice of strings with paths to all of the files
// starting from "path"
func listFiles(path string) ([]string, error) {
	// 150 is just an arbitrary number I chose to avoid constant memory
	// allocation while expanding the slice capacity
	result := make([]string, 0, 150)
	err := filepath.WalkDir(path,
		func(s string, d fs.DirEntry, e error) error {
			if e != nil {
				return WrapErrorf(e, "failed to walk directory %q", path)
			}
			if !d.IsDir() {
				result = append(result, s[len(path)+1:])
			}
			return nil
		})
	if err != nil {
		return make([]string, 0), WrapErrorf(
			err, "failed to walk directory %q", path)
	}
	return result, nil
}

// checkDeletionSafety checks whether it's safe to delete files from given path
// based on the list of removeable files. The removeableFiles list must be
// sorted. The function relies on filepath.WalkDir walking files
// alphabetically. It returns nil value when its safe to delete the files or
// an error in opposite case.
func checkDeletionSafety(path string, removableFiles []string) error {
	i := 0 // current index on the removableFiles list to check
	stats, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // directory doesn't exist there is nothing to check
		}
		return WrapErrorf(err, "unable to access stats of the %q path", path)
	} else if !stats.IsDir() {
		return WrapErrorf(nil, "the output path %q is not a directory", path)
	}
	err = filepath.WalkDir(path,
		func(s string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() { // Directories aren't checked
				return nil
			}
			s = s[len(path)+1:] // remove path from the file path
			for {
				if i >= len(removableFiles) {
					return WrapErrorf(
						nil, "file path %q is not on the list of the files "+
							"recently modified by Regolith", s)
				}
				currPath := removableFiles[i]
				i++
				if s == currPath { // found path on the list
					break
				} else if s < currPath { // this path won't be on the list
					return WrapErrorf(
						nil, "file path %q is not on the list of the files "+
							"recently modified by Regolith", s)
				}
			}
			return nil
		})
	return err
}
