package regolith

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
)

const EditedFilesPath = ".regolith/cache/edited_files.json"

// EditedFiles is used to load edited_files.json from cache in order
// to check if the files are safe to delete.
type EditedFiles struct {
	Rp []string `json:"rp"`
	Bp []string `json:"bp"`
}

// Dump dumps EditedFiles to EditedFilesPath in JSON format.
func (f *EditedFiles) Dump() error {
	result, err := json.MarshalIndent(f, "", "\t")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(EditedFilesPath, result, 0666)
	if err != nil {
		return err
	}
	return nil
}

// CheckDeletionSafety checks whether it's safe to delete files from rpPath and
// bpPath based on the lists of removeable files from EditedFiles object.
func (f *EditedFiles) CheckDeletionSafety(rpPath string, bpPath string) error {
	err := checkDeletionSafety(rpPath, f.Rp)
	if err != nil {
		return err
	}
	return checkDeletionSafety(bpPath, f.Bp)
}

// LoadEditedFiles data from edited_files.json or returns an empty object
// if file doesn't exist.
func LoadEditedFiles() EditedFiles {
	data, err := ioutil.ReadFile(EditedFilesPath)
	var result EditedFiles
	if err != nil {
		return EditedFiles{}
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return EditedFiles{}
	}
	return result
}

// NewEditedFiles creates new EditedFiles object with lists of the files from
// rpPath and bpPath.
func NewEditedFiles(rpPath string, bpPath string) (EditedFiles, error) {
	var result EditedFiles
	rpFiles, err := listFiles(rpPath)
	if err != nil {
		return result, err
	}
	bpFiles, err := listFiles(bpPath)
	if err != nil {
		return result, err
	}
	result.Rp = rpFiles
	result.Bp = bpFiles
	return result, nil
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
				return e
			}
			if !d.IsDir() {
				result = append(result, s)
			}
			return nil
		})
	if err != nil {
		return make([]string, 0), err
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
		return err // other error
	} else if !stats.IsDir() {
		return errors.New("output path is a file")
	}
	err = filepath.WalkDir(path,
		func(s string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() { // Directories aren't checked
				return nil
			}
			for {
				if i >= len(removableFiles) {
					return fmt.Errorf("file path %q is not on the list of the files recently modified by Regolith", s)
				}
				currPath := removableFiles[i]
				i++
				if s == currPath { // found path on the list
					break
				} else if s < currPath { // this path won't be on the list
					return fmt.Errorf("file path %q is not on the list of the files recently modified by Regolith", s)
				}
			}
			return nil
		})
	return err
}
