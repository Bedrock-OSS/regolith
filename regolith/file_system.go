package regolith

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const copyFileBufferSize = 1_000_000 // 1 MB
const directoryNotEmptyOrNewError = "\"%s\" must be a new path or an empty directory."
const filepathAbsFailError = "Failed to get full path to \"%s\""

// RevertableFsOperations is a struct that performs file system operations,
// keeps track of them, and can undo them if something goes wrong.
type RevertableFsOperations struct {
	// undoOperations is a history of performed operations, ready to be
	// reverted
	undoOperations []func() error

	// The path used for storing the backup files
	backupPath string

	// The counter used for naming the backup files
	backupFileCounter int
}

// NewRevertableFsOperaitons creates a new FsOperationBatch struct.
func NewRevertableFsOperaitons(backupPath string) (*RevertableFsOperations, error) {
	// Resolve the path to backups in case of changing the working directory
	// during runtime
	fullBackupPath, err := filepath.Abs(backupPath)
	if err != nil {
		return nil, WrapErrorf(err, filepathAbsFailError, backupPath)
	}
	// Create empty directory for the backup files in the backup path
	err = createBackupPath(fullBackupPath)
	if err != nil {
		return nil, PassError(err)
	}

	return &RevertableFsOperations{
		undoOperations: []func() error{},
		backupPath:     fullBackupPath,
	}, nil
}

// Close deletes temporary files of FsOperationBatch. At this point the
// FsOperationBatch should not be used anymore.
func (r *RevertableFsOperations) Close() error {
	// Clean the backup directory
	err := os.RemoveAll(r.backupPath)
	if err != nil {
		return WrapErrorf(
			err, "Failed to clean the backup directory \"%s\".\n"+
				"If there are any file left in that directory, please "+
				"clean them manually.", r.backupPath)
	}

	return nil
}

// Undo restores the state of the file system from before the operations of
// the FsOperationBatch.
func (r *RevertableFsOperations) Undo() error {
	var undo func() error
	for len(r.undoOperations) > 0 {
		i := len(r.undoOperations) - 1 // Last item index
		undo, r.undoOperations = r.undoOperations[i], r.undoOperations[:i]
		err := undo()
		if err != nil {
			return WrapError(
				err, "Failed to undo file system operations.")
		}
	}
	return nil
}

// Delete removes a file or directory.
// For deleting entire directories, check out the DeleteDir.
func (r *RevertableFsOperations) Delete(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return WrapErrorf(err, "Failed to check if file exists: %s", path)
	}
	tmpPath := r.getTempFilePath(path)
	err := ForceMoveFile(path, tmpPath)
	if err != nil {
		return WrapErrorf(err, "Failed to delete directory: %s", path)
	}
	r.undoOperations = append(r.undoOperations, func() error {
		err := ForceMoveFile(tmpPath, path)
		if err != nil {
			return PassError(err)
		}
		return nil
	})
	return nil
}

// DeleteDir deletes a directory.
// This method is better for deleting directories than Delete method because
// it moves the files of the directory one by one to the backup directory,
// and it's able to undo the operation even if an error occures in the middle
// of its execution.
func (r *RevertableFsOperations) DeleteDir(path string) error {
	stat, err := os.Stat(path)
	if err == nil && !stat.IsDir() {
		err = r.Delete(path)
		if err != nil {
			return PassError(err)
		}
		return nil
	}
	deleteFunc := func(currPath string, info fs.FileInfo, err error) error {
		if r == nil {
			panic("NO CO KURWA")
		}
		err = r.Delete(currPath)
		if err != nil {
			return PassError(err)
		}
		return nil
	}
	// Loop source, move files from source to target and create directories
	err = PostorderWalkDir(path, deleteFunc)
	if err != nil {
		return PassError(err)
	}
	stat, err = os.Stat(path)
	if err != nil {
		return PassError(err)
	}
	err = deleteFunc(path, stat, nil)
	if err != nil {
		return PassError(err)
	}
	return nil
}

// Move moves a file or a directory from source to target.
// For moving or copying entire directories, check out the MoveoOrCopyDir.
func (r *RevertableFsOperations) Move(source, target string) error {
	err := moveOrCopyAssertions(source, target)
	if err != nil {
		return PassError(err)
	}
	err = r.move(source, target)
	if err != nil {
		return PassError(err)
	}
	return nil
}

// Copies a file from source to target.
// For moving or copying entire directories, check out the MoveoOrCopyDir.
func (r *RevertableFsOperations) Copy(source, target string) error {
	err := moveOrCopyAssertions(source, target)
	if err != nil {
		return PassError(err)
	}
	err = r.copy(source, target)
	if err != nil {
		return PassError(err)
	}
	return nil
}

// MoveOrCopy tries to move source file to the target, if it fails, it copies
// it. If the copy function is performed, the source file remains in its
// original location.
// For moving or copying entire directories, check out the MoveoOrCopyDir.
func (r *RevertableFsOperations) MoveOrCopy(source, target string) error {
	err := moveOrCopyAssertions(source, target)
	if err != nil {
		return PassError(err)
	}
	// Try to move first
	err = r.move(source, target)

	// If failed, try to copy
	if err != nil {
		err = r.copy(source, target)
		if err != nil {
			return PassError(err)
		}
		return nil
	}
	return nil
}

// MkdirAll creates a directory and all of its parents just like the
// os.MkdirAll, but also pushes the delete operations to the undo stack for
// newly created directories. The undo operations of the function only
// delete the directories that it created. If the path already exists, nothing
// goes to the stack. The undo operation deletes entire directory and doesn't
// check if additional content was added to it.
func (r *RevertableFsOperations) MkdirAll(path string) error {
	fullPath, err := filepath.Abs(path)
	if err != nil {
		return WrapErrorf(err, filepathAbsFailError, path)
	}

	// Get the root directory of newly created paths for undo operation
	undoPath, found, err := GetFirstUnexistingSubpath(fullPath)
	if err != nil {
		return WrapErrorf(
			err, "Failed to check if path \"%s\" is valid for creating a "+
				"directory.", path)
	}

	if found {
		err = os.MkdirAll(fullPath, 0755)
		if err != nil {
			return PassError(err)
		}
		r.undoOperations = append(r.undoOperations, func() error {
			err := os.RemoveAll(undoPath)
			if err != nil {
				return PassError(err)
			}
			return nil
		})
	}
	return nil
}

// MoveOrCopyDir safely moves a directory form source to target.
// The target path must not exist or be empty directory.
// This function is better for moving or copying directories than
// Move, Copy or MoveOrCopy functions because it moves the files of the
// directory one by one and it's able to undo its actions even if an error
// occures in the middle of moving.
func (r *RevertableFsOperations) MoveoOrCopyDir(source, target string) error {
	// Check if target is empty or doesn't exist
	fullTargetPath, err := filepath.Abs(target)
	if err != nil {
		return WrapErrorf(err, directoryNotEmptyOrNewError, target)
	}
	// Check if target is an existing empty directory, or non-existing
	// directory
	stat, err := os.Stat(fullTargetPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return WrapErrorf(err, directoryNotEmptyOrNewError, target)
		} // else we can continue
	} else {
		// Target path exists, it must be an empty directory
		if !stat.IsDir() {
			return WrappedErrorf(directoryNotEmptyOrNewError, fullTargetPath)
		}
		empty, err := IsDirEmpty(fullTargetPath)
		if err != nil {
			return WrapErrorf(err, directoryNotEmptyOrNewError, fullTargetPath)
		}
		if !empty {
			return WrappedErrorf(directoryNotEmptyOrNewError, fullTargetPath)
		} // else we can continue
	}

	// Loop source, move files from source to target and create directories
	err = PostorderWalkDir(
		source, func(currSourcePath string, info os.FileInfo, err error) error {
			sourceRelativePath, err := filepath.Rel(source, currSourcePath)
			if err != nil {
				return PassError(err)
			}
			currTargetPath := filepath.Join(target, sourceRelativePath)
			if info.IsDir() {
				err = r.MkdirAll(currTargetPath)
				if err != nil {
					return PassError(err)
				}
				// It's safe because this won't remove non-empty path
				err = os.Remove(currSourcePath)
				if err != nil {
					return PassError(err)
				}
				return nil
			}
			err = r.MoveOrCopy(currSourcePath, currTargetPath)
			if err != nil {
				return PassError(err)
			}
			return nil
		})
	if err != nil {
		return PassError(err)
	}
	return nil
}

// moveOrCopyAssertions does a common check for move, copy and move or
// copy operation. It asserts that source path is valid and that the
// target doesn't exist.
func moveOrCopyAssertions(source, target string) error {
	if _, err := os.Stat(source); err != nil {
		if os.IsNotExist(err) {
			return WrapErrorf(err, "Source file doesn't exist: %s", source)
		}
		return WrapErrorf(err, "Failed to check if file exists: %s", source)
	}
	stat, err := os.Stat(target)
	if stat != nil {
		return WrappedErrorf("Target path already exists %s", target)
	} else if err != nil {
		if !os.IsNotExist(err) {
			return WrapErrorf(
				err, "Failed to check if file exists: %s", target)
		}
		// Skip IsNotExist errors because it's ok if target doesn't exist
	}
	return nil
}

// move handles the Move method
func (r *RevertableFsOperations) move(source, target string) error {
	// Make parent directory of target
	err := os.MkdirAll(filepath.Dir(target), 0755)
	if err != nil {
		return WrapErrorf(
			err, "Failed to create \"%s\".", target)
	}
	err = os.Rename(source, target)
	if err != nil {
		return WrapErrorf(
			err, "Failed to move file %s to %s", source, target)
	}
	r.undoOperations = append(r.undoOperations, func() error {
		return os.Rename(target, source)
	})
	return nil
}

// copy handles the Copy method
func (r *RevertableFsOperations) copy(source, target string) error {
	err := CopyFile(source, target)
	if err != nil {
		return WrapErrorf(
			err, "Failed to copy file \"%s\" to \"%s\"", source, target)
	}
	r.undoOperations = append(r.undoOperations, func() error {
		return os.Remove(target)
	})
	return nil
}

// getTempFilePath returns a temporary path in the bacup directory to store
// files deleted by the FsOperationBatch before the operations are fully
// applied (before calling Close()).
func (r *RevertableFsOperations) getTempFilePath(base string) string {
	_, file := filepath.Split(base)
	return filepath.Join(
		r.backupPath, strconv.Itoa(r.backupFileCounter)+"_"+file)
}

// createBackupPath creates an empty directory at the given path or returns an
// error. The function fails if the path already exists but isn't empty or
// when creating the directory fails.
func createBackupPath(path string) error {
	if stat, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(path, 0755)
			if err != nil {
				return WrapErrorf(
					err, "Failed to create backup directory: %s", path)
			}
		} else {
			return WrapErrorf(
				err, "Failed to check if backup directory exists: %s",
				path)
		}
	} else if !stat.IsDir() {
		return WrapErrorf(
			err, "Unable to create backups in \"%s\" because it's not a directory.",
			path)
	} else {
		isEmpty, err := IsDirEmpty(path)
		if err != nil {
			return WrapErrorf(err, "Failed to check if backup directory is "+
				"empty: %s", path)
		}
		if !isEmpty {
			return WrapErrorf(err, "Unable to create backups in \"%s\" "+
				"because the directory is not empty.", path)
		}
	}
	return nil
}

// GetFirstUnexistingSubpath takes a path and returns its ancestor.
// The returned path that doesn't exist but has an existing parent.
// The function returns 3 values - the path, a boolean indicating if
// the path was found successfully and an error. If the input path already
// exists, it returns ("", false, nil).
func GetFirstUnexistingSubpath(path string) (string, bool, error) {
	path = filepath.Clean(path)
	fullPath, err := filepath.Abs(path)
	if err != nil {
		WrapErrorf(err, filepathAbsFailError, path)
	}
	pathParts := strings.Split(fullPath, string(os.PathSeparator))
	currPath := pathParts[0] // There is always at least 1 item
	// Keep adding path parts until we find a non-existing path
	for i := 1; i < len(pathParts); i++ {
		// Join the parts together. Don't filepath.Join(). It fails when
		// joining with drive letter on Windows.
		currPath = strings.Join(
			[]string{currPath, pathParts[i]}, string(os.PathSeparator))
		if stat, err := os.Stat(currPath); err != nil {
			if os.IsNotExist(err) {
				return currPath, true, nil
			}
		} else if !stat.IsDir() {
			return "", false, WrappedErrorf(
				"Subpath %s of %s is not a directory",
				currPath, path)
		}
	}
	return "", false, nil
}

// IsDirEmpty checks whether the path points at empty directory. If the path
// is not a directory or info about the path can't be obtaioned it returns
// false. If the path is a directory and it is empty, it returns true.
func IsDirEmpty(path string) (bool, error) {
	if stat, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, WrappedErrorf("Path %q does not exist.", path)
		}
		return false, WrapErrorf(err, "Failed to stat %q.", path)
	} else if !stat.IsDir() {
		return false, WrappedErrorf("Path %q is not a directory.", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return false, WrapErrorf(err, "Failed to open %q.", path)
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	} else if err != nil {
		return false, PassError(err)
	}
	// err is nil -> not empty
	return false, nil
}

// AreFilesEqual compares files from two paths A and B and returns true if
// they're equal.
func AreFilesEqual(a, b string) (bool, error) {
	const bufferSize = 4000 // 4kB
	aStat, err := os.Stat(a)
	if err != nil {
		return false, WrapErrorf(err, "Failed to stat file: %s", a)
	}
	bStat, err := os.Stat(b)
	if err != nil {
		return false, WrapErrorf(err, "Failed to stat file: %s", b)
	}
	if aStat.Size() != bStat.Size() {
		return false, nil
	}
	aFile, err := os.Open(a)
	if err != nil {
		return false, WrapErrorf(err, "Failed to open file: %s", a)
	}
	defer aFile.Close()
	bFile, err := os.Open(b)
	if err != nil {
		return false, WrapErrorf(err, "Failed to open file: %s", b)
	}
	defer bFile.Close()
	aBuff := make([]byte, bufferSize)
	bBuff := make([]byte, bufferSize)
	for {
		aRead, err := aFile.Read(aBuff)
		if err != nil && err != io.EOF {
			return false, WrapErrorf(err, "Failed to read file: %s", a)
		}
		bRead, err := bFile.Read(bBuff)
		if err != nil && err != io.EOF {
			return false, WrapErrorf(err, "Failed to read file: %s", b)
		}
		if !bytes.Equal(aBuff[:aRead], bBuff[:bRead]) {
			return false, nil
		}
		if err == io.EOF {
			break
		}
	}
	return true, nil
}

// CopyFile copies a file from source to target. If it's necessary it creates
// the target directory.
func CopyFile(source, target string) error {
	// Make parent directory of target
	err := os.MkdirAll(filepath.Dir(target), 0755)
	if err != nil {
		return WrapErrorf(
			err, "Failed to create \"%s\".", target)
	}
	buf := make([]byte, copyFileBufferSize)
	// Open source for reading
	sourceF, err := os.Open(source)
	if err != nil {
		return WrapErrorf(
			err, "Failed to open \"%s\" for reading.", source)
	}
	defer sourceF.Close()
	// Open target for writing
	targetF, err := os.Create(target)
	if err != nil {
		return WrapErrorf(
			err, "Failed to open \"%s\" for writing.", target)
	}
	defer targetF.Close()
	// Copy the file
	for {
		n, err := sourceF.Read(buf)
		if err != nil && err != io.EOF {
			return WrapErrorf(err, "Failed to read from \"%s\".", source)
		}
		if n == 0 {
			break
		}

		if _, err := targetF.Write(buf[:n]); err != nil {
			return WrapErrorf(err, "Failed to write to \"%s\".", target)
		}
	}
	targetF.Sync()
	return nil
}

// ForceMoveFile is a function that forces to move file in file system.
// If os.Move fails, it creates a copy of the file to the target location and
// then deletes the original file.
func ForceMoveFile(source, target string) error {
	// Try regular move first
	err := os.Rename(source, target)
	if err == nil {
		return nil
	}
	// Failed to rename try to copy
	stat, err := os.Stat(source)
	if err != nil {
		return WrapErrorf(err, "Failed to stat file: %s", source)
	} else if stat.IsDir() {
		err = os.MkdirAll(target, 0755)
		if err != nil {
			return WrapErrorf(err, "Failed to create directory: %s", target)
		}
		os.Remove(source) // Only works for empty directories
		if err != nil {
			return WrapErrorf(err, "Failed to remove directory: %s", source)
		}
	}
	if err := CopyFile(source, target); err != nil {
		return WrapErrorf(err, "Failed to copy file \"%s\" to \"%s\"",
			source, target)
	}
	if err := os.RemoveAll(source); err != nil {
		return WrapErrorf(
			err,
			"Failed to remove file \"%s\" after it was copied to \"%s\"",
			source, target)
	}
	return nil
}

// PostorderWalkDir walks a directory like filepath.WalkDir but the order is
// defined by the postorder traversal algorithm (leafs first, than their root).
// Since the function calls the walkFunc for the leafs first, it's impossible
// to ignore directories using "filepath.SkipDir" as an error like in the
// regular filepath.WalkDir.
func PostorderWalkDir(root string, fn filepath.WalkFunc) error {
	info, err := os.Lstat(root)
	if err != nil {
		err = fn(root, nil, err) // Special case, pass through fn
	} else {
		if info.IsDir() {
			err = postorderWalkDir(root, info, fn)
		} else {
			err = fn(root, info, err)
		}
	}
	return err
}

// postorderWalkDir is used by PostorderWalkDir for recursion.
func postorderWalkDir(path string, info os.FileInfo, fn filepath.WalkFunc) error {
	f, err := os.Open(path)
	if err != nil {
		return fn(path, info, err)
	}
	defer f.Close()
	subdirs, _ := f.Readdirnames(-1)
	sort.Strings(subdirs)
	for _, subdir := range subdirs {
		subpath := filepath.Join(path, subdir)
		stat, err := os.Lstat(subpath)
		if err != nil {
			err = fn(subpath, stat, err)
		} else {
			err = postorderWalkDir(subpath, stat, fn)
		}
		if err != nil {
			return err
		}
		err = fn(subpath, stat, err)
		if err != nil {
			return err
		}
	}
	return nil
}
