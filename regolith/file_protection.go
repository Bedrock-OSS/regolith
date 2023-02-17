package regolith

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Bedrock-OSS/go-burrito/burrito"
)

const EditedFilesPath = "cache/edited_files.json"

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
func LoadEditedFiles(dotRegolithPath string) EditedFiles {
	data, err := os.ReadFile(filepath.Join(dotRegolithPath, EditedFilesPath))
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
func (f *EditedFiles) Dump(dotRegolithPath string) error {
	result, err := json.MarshalIndent(f, "", "\t")
	if err != nil { // This should never happen.
		return burrito.WrapError(err, "Failed to marshal edited files list JSON.")
	}
	// Create parent directory of EditedFilesPath
	efp := filepath.Join(dotRegolithPath, EditedFilesPath)
	parentDir := filepath.Dir(efp)
	err = os.MkdirAll(parentDir, 0755)
	if err != nil {
		return burrito.WrapErrorf(err, osMkdirError, parentDir)
	}
	err = os.WriteFile(efp, result, 0644)
	if err != nil {
		return burrito.WrapErrorf(err, fileWriteError, efp)
	}
	return nil
}

// CheckDeletionSafety checks whether it's safe to delete files from rpPath and
// bpPath based on the lists of removable files from EditedFiles object.
func (f *EditedFiles) CheckDeletionSafety(rpPath string, bpPath string) error {
	files, ok := f.Rp[rpPath]
	if !ok {
		files = make([]string, 0)
	}
	err := checkDeletionSafety(rpPath, files)
	if err != nil {
		return burrito.WrapError(
			err, "Deletion safety check for resource pack failed.")
	}
	files, ok = f.Bp[bpPath]
	if !ok {
		files = make([]string, 0)
	}
	err = checkDeletionSafety(bpPath, files)
	if err != nil {
		return burrito.WrapError(
			err, "Deletion safety check for behavior pack failed.")
	}
	return nil
}

// UpdateFromPaths updates the edited files data based on the paths to the
// resource pack and behavior pack.
func (f *EditedFiles) UpdateFromPaths(rpPath string, bpPath string) error {
	rpFiles, err := listFiles(rpPath)
	if err != nil {
		return burrito.WrapError(err, "Failed to list resource pack files.")
	}
	bpFiles, err := listFiles(bpPath)
	if err != nil {
		return burrito.WrapError(err, "Failed to list behavior pack files.")
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

// listFiles returns a slice of strings with paths to all the files
// starting from "path"
func listFiles(path string) ([]string, error) {
	// 150 is just an arbitrary number I chose to avoid constant memory
	// allocation while expanding the slice capacity
	result := make([]string, 0, 150)
	err := filepath.WalkDir(path,
		func(s string, d fs.DirEntry, e error) error {
			if e != nil {
				return burrito.PassError(e)
			}
			if !d.IsDir() {
				relpath, err := filepath.Rel(path, s)
				if err != nil {
					return burrito.WrapErrorf(err, osRelError, path, s)
				}
				result = append(result, relpath)
			}
			return nil
		})
	if err != nil {
		return make([]string, 0), burrito.WrapErrorf(err, osWalkError, path)
	}
	return result, nil
}

// checkDeletionSafety checks whether it's safe to delete files from given path
// based on the list of removable files. The removableFiles list must be
// sorted. The function relies on filepath.WalkDir walking files
// alphabetically. It returns nil value when it's safe to delete the files or
// an error in opposite case.
func checkDeletionSafety(path string, removableFiles []string) error {
	i := 0 // current index on the removableFiles list to check
	stats, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // directory doesn't exist there is nothing to check
		}
		return burrito.WrapErrorf(err, osStatErrorAny, path)
	} else if !stats.IsDir() {
		return burrito.WrappedErrorf(isDirNotADirError, path)
	}
	err = filepath.WalkDir(path,
		func(s string, d fs.DirEntry, e error) error {
			if e != nil {
				return burrito.WrapErrorf(e, osWalkError, path)
			}
			if d.IsDir() { // Directories aren't checked
				return nil
			}
			relpath, err := filepath.Rel(path, s)
			if err != nil {
				return burrito.WrapErrorf(err, osRelError, path, s)
			}
			s = relpath // remove path from the file path
			const notRegolithFileError = "File is not on the list of files" +
				" created by Regolith.\nPath: %s"
			for {
				if i >= len(removableFiles) {
					return burrito.WrappedErrorf(notRegolithFileError, s)
				}
				currPath := removableFiles[i]
				i++
				cmpVal := compareFilePaths(s, currPath)
				if cmpVal == 0 { // found path on the list
					break
				} else if cmpVal < 0 { // this path won't be on the list
					return burrito.WrappedErrorf(notRegolithFileError, s)
				}
			}
			return nil
		})
	if err != nil {
		return burrito.PassError(err)
	}
	return nil
}

// compareFilePaths compares two filepaths to oder them lexicographically.
// This is not the same as comparing the file paths as strings because
// "." < "/" and "." < "\\" but the "text.txt" should be greater than
//  "text/text.txt" ("text.txt" > "text/text.txt"). This is the same order
// that you would get when you use filepath.Walk.
// The function returns -1 when "a" < "b", 0 when "a" == "b" and 1 when
// "a" > "b".
func compareFilePaths(a, b string) int {
	a = strings.Replace(a, "\\", "/", -1)
	b = strings.Replace(b, "\\", "/", -1)
	aSlice := strings.Split(a, string("/"))
	bSlice := strings.Split(b, string("/"))
	for i := 0; i < len(aSlice) && i < len(bSlice); i++ {
		if cmp := strings.Compare(aSlice[i], bSlice[i]); cmp != 0 {
			return cmp
		} // else - they're the same
	}
	if len(aSlice) < len(bSlice) {
		// This shouldn't really happen because you can't use exactly the same
		// name for file and directory.
		return -1
	}
	if len(aSlice) > len(bSlice) {
		// This shouldn't really happen because you can't use exactly the same
		// name for file and directory.
		return 1
	}
	return 0
}
