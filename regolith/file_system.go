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
	"time"

	"github.com/Bedrock-OSS/go-burrito/burrito"

	"github.com/otiai10/copy"
)

const copyFileBufferSize = 1_000_000 // 1 MB

// revertibleFsOperations is a struct that performs file system operations,
// keeps track of them, and can undo them if something goes wrong.
type revertibleFsOperations struct {
	// undoOperations is a history of performed operations, ready to be
	// reverted
	undoOperations []func() error

	// The path used for storing the backup files
	backupPath string

	// The counter used for naming the backup files
	backupFileCounter int
}

// NewRevertibleFsOperations creates a new FsOperationBatch struct.
func NewRevertibleFsOperations(backupPath string) (*revertibleFsOperations, error) {
	// Resolve the path to backups in case of changing the working directory
	// during runtime
	fullBackupPath, err := filepath.Abs(backupPath)
	if err != nil {
		return nil, burrito.WrapErrorf(err, filepathAbsError, backupPath)
	}
	// Create empty directory for the backup files in the backup path
	err = createBackupPath(fullBackupPath)
	if err != nil {
		return nil, burrito.PassError(err)
	}

	return &revertibleFsOperations{
		undoOperations: []func() error{},
		backupPath:     fullBackupPath,
	}, nil
}

// Close deletes temporary files of FsOperationBatch. At this point the
// FsOperationBatch should not be used anymore.
func (r *revertibleFsOperations) Close() error {
	// Clean the backup directory
	err := os.RemoveAll(r.backupPath)
	if err != nil {
		return burrito.WrapErrorf(
			err,
			"Failed to clean the backup directory.\n"+
				"Path: %s\n"+
				"This is a directory that Regolith uses to store backup files"+
				" in case of failure while performing file system operations.\n"+
				"Regolith uses them to restore the state of the file system "+
				"when an operation like copy or delete fails.\n"+
				"If your project is missing files, you can check "+
				"this directory.\n"+
				"Please clean this directory manually before running "+
				"Regolith again.",
			r.backupPath)
	}
	return nil
}

// Undo restores the state of the file system from before the operations of
// the FsOperationBatch.
func (r *revertibleFsOperations) Undo() error {
	var undo func() error
	for len(r.undoOperations) > 0 {
		i := len(r.undoOperations) - 1 // Last item index
		undo, r.undoOperations = r.undoOperations[i], r.undoOperations[:i]
		err := undo()
		if err != nil {
			return burrito.PassError(err)
		}
	}
	return nil
}

// Delete removes a file or directory.
// For deleting entire directories, check out the DeleteDir.
func (r *revertibleFsOperations) Delete(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return burrito.WrapErrorf(err, osStatErrorAny, path)
	}
	tmpPath := r.getTempFilePath(path)
	err := ForceMoveFile(path, tmpPath)
	if err != nil {
		return burrito.WrapErrorf(
			err,
			"Failed to move the file to the backup location.\n"+
				"Path: %s\n"+
				"Backup path: %s",
			path, tmpPath)
	}
	r.undoOperations = append(r.undoOperations, func() error {
		err := ForceMoveFile(tmpPath, path)
		if err != nil {
			return burrito.WrapErrorf(err, "Failed to forcefully move file."+
				"\nSource: %s\nTarget: %s", tmpPath, path)
		}
		return nil
	})
	return nil
}

// DeleteDir deletes a directory.
// This method is better for deleting directories than Delete method because
// it moves the files of the directory one by one to the backup directory,
// and it's able to undo the operation even if an error occurs in the middle
// of its execution.
func (r *revertibleFsOperations) DeleteDir(path string) error {
	// TODO - maybe Delete should be able to delete both directories and files and DeleteDir should be private
	stat, err := os.Stat(path)
	if err == nil && !stat.IsDir() {
		err = r.Delete(path)
		if err != nil {
			return burrito.WrapErrorf(err, revertibleFsOperationsDeleteError, path)
		}
		return nil
	}
	deleteFunc := func(currPath string, info fs.FileInfo, err error) error {
		err = r.Delete(currPath)
		if err != nil {
			return burrito.WrapErrorf(err, revertibleFsOperationsDeleteError, currPath)
		}
		return nil
	}
	// Loop source, move files from source to target and create directories
	err = PostorderWalkDir(path, deleteFunc)
	if err != nil {
		return burrito.PassError(err)
	}
	stat, err = os.Stat(path)
	if err != nil {
		return burrito.WrapErrorf(err, osStatErrorAny, path)
	}
	err = deleteFunc(path, stat, nil)
	if err != nil {
		return burrito.PassError(err)
	}
	return nil
}

// Move moves a file or a directory from source to target.
// For moving or copying entire directories, check out the MoveOrCopyDir.
func (r *revertibleFsOperations) Move(source, target string) error {
	err := moveOrCopyAssertions(source, target)
	if err != nil {
		return burrito.PassError(err)
	}
	err = r.move(source, target)
	if err != nil {
		return burrito.PassError(err)
	}
	return nil
}

// Copies a file from source to target.
// For moving or copying entire directories, check out the MoveOrCopyDir.
func (r *revertibleFsOperations) Copy(source, target string) error {
	err := moveOrCopyAssertions(source, target)
	if err != nil {
		return burrito.PassError(err)
	}
	err = r.copy(source, target)
	if err != nil {
		// PasseError copy function shouldn't say that copy failed, the
		// error messages like that are handled outside of the function
		return burrito.PassError(err)
	}
	return nil
}

// MoveOrCopy tries to move source file to the target, if it fails, it copies
// it. If the copy function is performed, the source file remains in its
// original location unless the clearTargetOnMoveFail flag is set to true.
// For moving or copying entire directories, check out the MoveOrCopyDir.
func (r *revertibleFsOperations) MoveOrCopy(
	source, target string, clearTargetOnMoveFail bool,
) error {
	err := moveOrCopyAssertions(source, target)
	if err != nil {
		return burrito.PassError(err)
	}
	// Try to move first
	err = r.move(source, target)

	// If failed, try to copy
	if err != nil {
		err = r.copy(source, target)
		if err != nil {
			// PasseError copy function shouldn't say that copy failed, the
			// error messages like that are handled outside of the function
			return burrito.PassError(err)
		}
		if clearTargetOnMoveFail {
			err = r.Delete(source)
			if err != nil {
				return burrito.WrapError(
					err,
					"Failed to delete the file after copying it to "+
						"the target location.")
			}
		}
	}
	return nil
}

// MkdirAll creates a directory and all of its parents just like the
// os.MkdirAll, but also pushes the delete operations to the undo stack for
// newly created directories. The undo operations of the function only
// delete the directories that it created. If the path already exists, nothing
// goes to the stack. The undo operation deletes entire directory and doesn't
// check if additional content was added to it.
func (r *revertibleFsOperations) MkdirAll(path string) error {
	fullPath, err := filepath.Abs(path)
	if err != nil {
		return burrito.WrapErrorf(err, filepathAbsError, path)
	}

	// Get the root directory of newly created paths for undo operation
	undoPath, found, err := GetFirstNonexistentSubpath(fullPath)
	if err != nil {
		return burrito.WrapErrorf(
			err,
			"Failed to define an undo operation for creating nested "+
				"directories.\n"+
				"Unable to find out which parts of the path are new.\n"+
				"Path: %s", path)
	}

	if found {
		err = os.MkdirAll(fullPath, 0755)
		if err != nil {
			return burrito.PassError(err)
		}
		r.undoOperations = append(r.undoOperations, func() error {
			err := os.RemoveAll(undoPath)
			if err != nil {
				return burrito.PassError(err)
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
// directory one by one, and it's able to undo its actions even if an error
// occurs in the middle of moving.
func (r *revertibleFsOperations) MoveOrCopyDir(source, target string) error {
	// Check if target is empty or doesn't exist
	fullTargetPath, err := filepath.Abs(target)
	if err != nil {
		return burrito.WrapErrorf(err, filepathAbsError, target)
	}
	// Make sure that the directory is empty or doesn't exist
	stat, err := os.Stat(fullTargetPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return burrito.WrapErrorf(err, assertEmptyOrNewDirError, target)
		} // else we can continue
	} else {
		// Target path exists, it must be an empty directory
		if !stat.IsDir() {
			return burrito.WrappedErrorf(assertEmptyOrNewDirError, fullTargetPath)
		}
		empty, err := IsDirEmpty(fullTargetPath)
		if err != nil {
			return burrito.WrapErrorf(err, assertEmptyOrNewDirError, fullTargetPath)
		}
		if !empty {
			return burrito.WrappedErrorf(assertEmptyOrNewDirError, fullTargetPath)
		} // else we can continue
	}
	// Make sure that the source path exists
	if _, err := os.Stat(source); err != nil {
		if os.IsNotExist(err) {
			return burrito.WrapErrorf(err, osStatErrorIsNotExist, source)
		}
		return burrito.WrapErrorf(err, osStatErrorAny, source)
	}

	// Loop source, move files from source to target and create directories
	err = PostorderWalkDir(
		source, func(currSourcePath string, info os.FileInfo, err error) error {
			if err != nil {
				return burrito.WrapErrorf(err, osWalkError, source)
			}
			sourceRelativePath, err := filepath.Rel(source, currSourcePath)
			if err != nil {
				return burrito.WrapErrorf(
					err, filepathRelError, source, currSourcePath)
			}
			currTargetPath := filepath.Join(target, sourceRelativePath)
			if info.IsDir() {
				err = r.MkdirAll(currTargetPath)
				if err != nil {
					return burrito.WrapErrorf(err, osMkdirError, currTargetPath)
				}
				// It's safe because this won't remove non-empty path
				err = os.Remove(currSourcePath)
				if err != nil {
					return burrito.WrapErrorf(err, osRemoveError, currSourcePath)
				}
				return nil
			}
			// We use clearTargetOnFail flag here because we rely on the file
			// in the target location being removed. If it's not removed, the
			// function will fail trying to delete the parent directory of that
			// file later in the loop.
			err = r.MoveOrCopy(currSourcePath, currTargetPath, true)
			if err != nil {
				return burrito.WrapErrorf(
					err, moveOrCopyError, currSourcePath, currTargetPath)
			}
			return nil
		})
	if err != nil {
		return burrito.PassError(err)
	}
	return nil
}

// moveOrCopyAssertions does a common check for move, copy and move or
// copy operation. It asserts that source path is valid and that the
// target doesn't exist.
func moveOrCopyAssertions(source, target string) error {
	if _, err := os.Stat(source); err != nil {
		if os.IsNotExist(err) {
			return burrito.WrapErrorf(err, osStatErrorIsNotExist, source)
		}
		return burrito.WrapErrorf(err, osStatErrorAny, source)
	}
	stat, err := os.Stat(target)
	if stat != nil {
		return burrito.WrappedErrorf(osStatExistsError, target)
	} else if err != nil {
		if !os.IsNotExist(err) {
			return burrito.WrapErrorf(
				err, osStatErrorAny, target)
		}
		// Skip IsNotExist errors because it's ok if target doesn't exist
	}
	return nil
}

// move handles the Move method
func (r *revertibleFsOperations) move(source, target string) error {
	// Make parent directory of target
	err := os.MkdirAll(filepath.Dir(target), 0755)
	if err != nil {
		return burrito.WrapErrorf(
			err, osMkdirError, target)
	}
	err = os.Rename(source, target)
	if err != nil {
		return burrito.WrapErrorf(
			err, osRenameError, source, target)
	}
	r.undoOperations = append(r.undoOperations, func() error {
		return os.Rename(target, source)
	})
	return nil
}

// copy handles the Copy method
func (r *revertibleFsOperations) copy(source, target string) error {
	err := CopyFile(source, target)
	if err != nil {
		// PasseError copy function shouldn't say that copy failed, the
		// error messages like that are handled outside of the function
		return burrito.PassError(err)
	}
	r.undoOperations = append(r.undoOperations, func() error {
		return os.Remove(target)
	})
	return nil
}

// getTempFilePath returns a temporary path in the backup directory to store
// files deleted by the FsOperationBatch before the operations are fully
// applied (before calling Close()).
func (r *revertibleFsOperations) getTempFilePath(base string) string {
	_, file := filepath.Split(base)
	result := filepath.Join(
		r.backupPath, strconv.Itoa(r.backupFileCounter)+"_"+file)
	r.backupFileCounter++
	return result
}

// createBackupPath creates an empty directory at the given path or returns an
// error. The function fails if the path already exists but isn't empty or
// when creating the directory fails.
func createBackupPath(path string) error {
	if stat, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(path, 0755)
			if err != nil {
				return burrito.WrapErrorf(err, osMkdirError, path)
			}
		} else {
			return burrito.WrapErrorf(err, osStatErrorAny, path)
		}
	} else if !stat.IsDir() {
		return burrito.WrapErrorf(
			err,
			"Unable to use path for backups because it's not a directory.\n"+
				"Path: %s",
			path)
	} else {
		isEmpty, err := IsDirEmpty(path)
		if err != nil {
			return burrito.WrapErrorf(err, isDirEmptyError, path)
		}
		if !isEmpty {
			return burrito.WrapError(
				err,
				"Unable to use path for backups because the directory is"+
					" not empty.")
		}
	}
	return nil
}

// GetFirstNonexistentSubpath takes a path and returns its ancestor.
// The returned path that doesn't exist but has an existing parent.
// The function returns 3 values - the path, a boolean indicating if
// the path was found successfully and an error. If the input path already
// exists, it returns ("", false, nil).
func GetFirstNonexistentSubpath(path string) (string, bool, error) {
	path = filepath.Clean(path)
	fullPath, err := filepath.Abs(path)
	if err != nil {
		burrito.WrapErrorf(err, filepathAbsError, path)
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
			return "", false, burrito.WrappedErrorf(
				"Found a subpath that is not a directory.\n"+
					"Subpath: %s\n"+
					"Full path: %s\n"+
					"Unable to continue searching for further subpaths "+
					"because files can't have subpaths.",
				currPath, path)
		}
	}
	return "", false, nil
}

// IsDirEmpty checks whether the path points at empty directory. If the path
// is not a directory or info about the path can't be obtained it returns
// false. If the path is a directory, and it is empty, it returns true.
func IsDirEmpty(path string) (bool, error) {
	if stat, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, burrito.WrappedErrorf(osStatErrorIsNotExist, path)
		}
		return false, burrito.WrapErrorf(err, osStatErrorAny, path)
	} else if !stat.IsDir() {
		return false, burrito.WrappedErrorf(isDirNotADirError, path)
	}
	f, err := os.Open(path)
	if err != nil {
		return false, burrito.WrapErrorf(err, osOpenError, path)
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	} else if err != nil {
		return false, burrito.WrapErrorf(
			err,
			"Failed to access subdirectories list.\n"+
				"Path: %s", path)
	}
	// err is nil -> not empty
	return false, nil
}

// GetMatchingDirContents returns a list of files in the directory that match the
// ones specified in the files parameter. If the path is not a directory or
// info about the path can't be obtained it returns an empty list and an error.
func GetMatchingDirContents(path string, files []string) ([]string, error) {
	result := make([]string, 0)
	if stat, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return result, burrito.WrappedErrorf(osStatErrorIsNotExist, path)
		}
		return result, burrito.WrapErrorf(err, osStatErrorAny, path)
	} else if !stat.IsDir() {
		return result, burrito.WrappedErrorf(isDirNotADirError, path)
	}
	f, err := os.Open(path)
	if err != nil {
		return result, burrito.WrapErrorf(err, osOpenError, path)
	}
	defer f.Close()
	names, err := f.Readdirnames(0)
	if err == io.EOF {
		return result, nil
	} else if err != nil {
		return result, burrito.WrapErrorf(
			err,
			"Failed to access subdirectories list.\n"+
				"Path: %s", path)
	}
	for _, name := range names {
		// Need to use lowercase because Windows is case-insensitive
		if stringInSlice(strings.ToLower(name), files) {
			result = append(result, name)
		}
	}
	// err is nil -> not empty
	return result, nil
}

// AreFilesEqual compares files from two paths A and B and returns true if
// they're equal.
func AreFilesEqual(a, b string) (bool, error) {
	const bufferSize = 4000 // 4kB
	aStat, err := os.Stat(a)
	if err != nil {
		return false, burrito.WrapErrorf(err, osStatErrorAny, a)
	}
	bStat, err := os.Stat(b)
	if err != nil {
		return false, burrito.WrapErrorf(err, osStatErrorAny, b)
	}
	if aStat.Size() != bStat.Size() {
		return false, nil
	}
	aFile, err := os.Open(a)
	if err != nil {
		return false, burrito.WrapErrorf(err, osOpenError, a)
	}
	defer aFile.Close()
	bFile, err := os.Open(b)
	if err != nil {
		return false, burrito.WrapErrorf(err, osOpenError, b)
	}
	defer bFile.Close()
	aBuff := make([]byte, bufferSize)
	bBuff := make([]byte, bufferSize)
	for {
		aRead, err := aFile.Read(aBuff)
		if err != nil && err != io.EOF {
			return false, burrito.WrapErrorf(err, fileReadError, a)
		}
		bRead, err := bFile.Read(bBuff)
		if err != nil && err != io.EOF {
			return false, burrito.WrapErrorf(err, fileReadError, b)
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
		return burrito.WrapErrorf(
			err, osMkdirError, target)
	}
	buf := make([]byte, copyFileBufferSize)
	// Open source for reading
	sourceF, err := os.Open(source)
	if err != nil {
		return burrito.WrapErrorf(
			err, osOpenError, source)
	}
	defer sourceF.Close()
	// Open target for writing
	targetF, err := os.Create(target)
	if err != nil {
		return burrito.WrapErrorf(
			err, osCreateError, target)
	}
	defer targetF.Close()
	// Copy the file
	for {
		n, err := sourceF.Read(buf)
		if err != nil && err != io.EOF {
			return burrito.WrapErrorf(err, fileReadError, source)
		}
		if n == 0 {
			break
		}

		if _, err := targetF.Write(buf[:n]); err != nil {
			return burrito.WrapErrorf(err, fileWriteError, target)
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
		return burrito.WrapErrorf(err, osStatErrorAny, source)
	} else if stat.IsDir() {
		err = os.MkdirAll(target, 0755)
		if err != nil {
			return burrito.WrapErrorf(err, osMkdirError, target)
		}
		os.Remove(source) // Only works for empty directories
		if err != nil {
			return burrito.WrapErrorf(err, osRemoveError, source)
		}
	} else { // Regular file
		if err := CopyFile(source, target); err != nil {
			return burrito.WrapErrorf(err, osCopyError, source, target)
		}
	}
	if err := os.RemoveAll(source); err != nil {
		return burrito.WrapErrorf(err, "Failed to remove file copied.")
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

// move moves files from source to destination. If both source and destination
// are directories, and the destination is empty, it will move the files from
// source to destination directly (without deleting the destination first).
// Moving the subdirectories to destination one by one instead of deleting it
// and renaming entire directory. This is important because, the deletion of
// the destination would break observation of the destination directory.
// This function is used by MoveOrCopy.
func move(source, destination string) error {
	// Check if source and destination are directories
	sourceInfo, err1 := os.Stat(source)
	destinationInfo, err2 := os.Stat(destination)

	// TODO - this part of code could be moved to another function. It's too much.
	if err1 == nil && err2 == nil && sourceInfo.IsDir() && destinationInfo.IsDir() {
		// Target must be empty
		if empty, err := IsDirEmpty(destination); err != nil {
			return burrito.WrapErrorf(err, isDirEmptyError, destination)
		} else if !empty {
			return burrito.WrapErrorf(err, isDirEmptyNotEmptyError, destination)
		}
		// Move all files in source to destination
		files, err := os.ReadDir(source)
		if err != nil {
			return burrito.WrapErrorf(err, osReadDirError, source)
		}
		movedFiles := make([][2]string, 0, 100)
		movingFailed := false
		var errMoving error
		for _, file := range files {
			src := filepath.Join(source, file.Name())
			dst := filepath.Join(destination, file.Name())
			errMoving = os.Rename(src, dst)
			if errMoving != nil {
				errMoving = burrito.WrapErrorf(
					errMoving, osRenameError, src, dst)
				Logger.Warnf(
					"Failed to move content of directory.\n"+
						"\tSource: %s\n"+
						"\tTarget: %s\n\n"+
						"\tOperation failed while moving a file:\n"+
						"\tSource: %s\n"+
						"\tTarget: %s\n\n"+
						"\tTrying to recover from error...",
					source, destination, src, dst)
				movingFailed = true
				break
			}
			movedFiles = append(movedFiles, [2]string{src, dst})
		}
		// If moving failed, rollback the moves
		if movingFailed {
			for _, movePair := range movedFiles {
				err = os.Rename(movePair[1], movePair[0])
				if err != nil {
					// This is a critical error that leaves the file system in
					// an invalid state. It shouldn't happen because it's from
					// moving files, that we had access to just a moment ago.
					Logger.Fatalf(
						"Regolith failed to recover from error which occured "+
							"while moving files from directory.\n"+
							"\tSource: %s\n"+
							"\tTarget: %s\n\n"+
							"\tRecovery failed while moving file.\n"+
							"\tSource: %s\n"+
							"\tTarget: %s\n"+
							"\tError: %s\n\n"+
							"\tThis is a critical error that leaves your "+
							"files in unorganized manner.\n\n"+
							"\tYou can try to recover the files manually "+
							"from:\n"+
							"\tPath: %s\n",
						source, destination, movePair[1], movePair[0], err,
						source)
				}
			}
			return burrito.WrapErrorf(
				errMoving,
				"Successfully recovered the original state of the directory "+
					"before crash.\nPath: %s", source)
		} else {
			return nil
		}
	}
	// Either source or destination is not a directory,
	// use normal os.Rename
	err := os.Rename(source, destination)
	if err != nil {
		return burrito.WrapErrorf(err, osRenameError, source, destination)
	}
	return nil
}

// MoveOrCopy tries to move the source to destination first and in case
// of failure it copies the files instead.
func MoveOrCopy(
	source string, destination string, makeReadOnly bool, copyParentAcl bool,
) error {
	// Make destination parent if not exists
	destinationParent := filepath.Dir(destination)
	if _, err := os.Stat(destinationParent); os.IsNotExist(err) {
		err = os.MkdirAll(destinationParent, 0755)
		if err != nil {
			return burrito.WrapErrorf(
				err, osMkdirError, destinationParent)
		}
	}
	// Move the source to the destination
	if err := move(source, destination); err != nil {
		Logger.Debugf(
			"Failed to move files.\n\tSource: %s\n\tTarget: %s\n"+
				"Trying to copy files instead...",
			filepath.Clean(source), filepath.Clean(destination))
		copyOptions := copy.Options{PreserveTimes: false, Sync: false}
		err := copy.Copy(source, destination, copyOptions)
		if err != nil {
			return burrito.WrapErrorf(err, osCopyError, source, destination)
		}
	} else if copyParentAcl { // No errors with moving files but needs ACL copy
		// TODO - this entire code block should be moved into the. copyFileSecurityInfo
		// printing this Info message below on Linux makes no sense.
		parent := filepath.Dir(destination)
		Logger.Infof(
			"Copying ACL from parent directory.\n\tSource: %s\n\tTarget: %s",
			parent, destination)
		if _, err := os.Stat(parent); os.IsNotExist(err) {
			return burrito.WrapErrorf(err, osStatErrorIsNotExist, parent)
		}
		err = copyFileSecurityInfo(parent, destination)
		if err != nil {
			return burrito.WrapErrorf(
				err, copyFileSecurityInfoError, source, destination)
		}
	}
	// Make files read only if this option is selected
	if makeReadOnly {
		Logger.Infof("Changing the access for output path to "+
			"read-only.\n\tPath: %s", destination)
		err := filepath.WalkDir(destination,
			func(s string, d fs.DirEntry, e error) error {

				if e != nil {
					// Error message isn't important as it's not passed further
					// in the code
					return e
				}
				if !d.IsDir() {
					os.Chmod(s, 0444)
				}
				return nil
			})
		if err != nil {
			Logger.Warnf(
				"Failed to change access of the output path to read-only.\n"+
					"\tPath: %s",
				destination)
		}
	}
	return nil
}

// SyncDirectories copies the source to destination while checking size and modification time.
func SyncDirectories(
	source string, destination string, makeReadOnly bool, copyParentAcl bool,
) error {
	// Make destination parent if not exists
	destinationParent := filepath.Dir(destination)
	if err := os.MkdirAll(destinationParent, 0755); err != nil {
		return burrito.WrapErrorf(err, osMkdirError, destinationParent)
	}
	err := filepath.Walk(source, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(source, srcPath)
		if err != nil {
			return burrito.WrapErrorf(err, filepathRelError, source, srcPath)
		}
		destPath := filepath.Join(destination, relPath)

		destInfo, err := os.Stat(destPath)
		if err != nil && !os.IsNotExist(err) {
			return burrito.WrapErrorf(err, osStatErrorAny, destPath)
		}
		if (err != nil && os.IsNotExist(err)) || info.ModTime() != destInfo.ModTime() || info.Size() != destInfo.Size() {
			if info.IsDir() {
				return os.MkdirAll(destPath, info.Mode())
			}
			Logger.Debugf("SYNC: Copying file %s to %s", srcPath, destPath)
			return copyFile(srcPath, destPath, info)
		} else {
			Logger.Debugf("SYNC: Skipping file %s", srcPath)
		}
		return nil
	})
	if err != nil {
		return burrito.WrapErrorf(err, osCopyError, source, destination)
	}

	// Remove files/folders in destination that are not in source
	toRemoveList := make([]string, 0)
	err = filepath.Walk(destination, func(destPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(destination, destPath)
		if err != nil {
			return burrito.WrapErrorf(err, filepathRelError, destination, destPath)
		}
		srcPath := filepath.Join(source, relPath)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			// TODO: Not sure if this is the best way to do this
			// The toRemoveList might get pretty big
			if !SliceAny[string](toRemoveList, func(s string) bool {
				return strings.HasPrefix(destPath, s)
			}) {
				Logger.Debugf("SYNC: Removing file %s", destPath)
				// Add to list of files to remove, because otherwise walk function might fail
				// when trying to walk a directory that doesn't exist anymore
				toRemoveList = append(toRemoveList, destPath)
			}
		}
		return nil
	})

	if err != nil {
		return burrito.PassError(err)
	}

	for _, path := range toRemoveList {
		err = os.RemoveAll(path)
		if err != nil {
			return burrito.WrapErrorf(err, osRemoveError, path)
		}
	}

	//TODO: copy ACL. To be honest, I have no clue why it was there in the first place

	// Make files read only if this option is selected
	if makeReadOnly {
		Logger.Infof("Changing the access for output path to "+
			"read-only.\n\tPath: %s", destination)
		err := filepath.WalkDir(destination,
			func(s string, d fs.DirEntry, e error) error {

				if e != nil {
					// Error message isn't important as it's not passed further
					// in the code
					return e
				}
				if !d.IsDir() {
					os.Chmod(s, 0444)
				}
				return nil
			})
		if err != nil {
			Logger.Warnf(
				"Failed to change access of the output path to read-only.\n"+
					"\tPath: %s",
				destination)
		}
	}
	return nil
}

func copyFile(src, dest string, info os.FileInfo) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return burrito.WrapErrorf(err, fileReadError, src)
	}
	if err = os.WriteFile(dest, data, info.Mode()); err != nil {
		return burrito.WrapErrorf(err, fileWriteError, dest)
	}
	err = os.Chtimes(dest, time.Now(), info.ModTime())
	if err != nil {
		return burrito.WrapErrorf(err, osChtimesError, dest)
	}
	return nil
}
