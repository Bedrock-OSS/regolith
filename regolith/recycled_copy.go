package regolith

import (
	"container/list"
	"encoding/hex"
	"encoding/json"
	"hash"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const dirHashPairsPath = ".regolith/cache/dir_hash_pairs.json"
const copyFileBufferSize = 1_000_000 // 1 MB

// PathHashPair is a single entry in the list that represents the state of the
// file path. It contains the path and the hash of the file/directory.
type PathHashPair struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

// RecycledMoveOrCopy moves or copies files from source to the target directory
// so that after the operation the state of the target directory as the state
// of the source directory before the operation. It tries to leverage the
// similarity of the source and target to minimize the number of files that
// need to be copied. The function doesn't care to preser the state of the
// source directory.
//
// sourceState and targetState are lists of PathHashPairs that provide
// pre-calculated hashes of the files in sourcePath or targetPath.
// The function assumes that the hashes are complete, correct and that they're
// sorted alphabetically by the path string. The paths in sourceState and
// targetState are relative to sourcePath and targetPath respectively (if
// sourceState and targetState are the same then both paths have the same
// content).
//
// canMove specifies whether the function is allowed to move files.
func RecycledMoveOrCopy(
	sourcePath, targetPath string,
	sourceState, targetState *list.List,
	canMove bool,
) error {
	if sourceState == nil || targetState == nil {
		return WrappedError(
			"ReplicateDirectory called with nil sourceState or targetState.")
	}
	s := sourceState.Front()
	t := targetState.Front()
	for s != nil || t != nil {
		if t == nil || s.Value.(PathHashPair).Path < t.Value.(PathHashPair).Path {
			// Target is ahead of source - the file doesn't exist in the
			// target. Copy file from source to the target.
			fullSPath := filepath.Join(sourcePath, s.Value.(PathHashPair).Path)
			fullTPath := filepath.Join(targetPath, s.Value.(PathHashPair).Path)
			moved, err := shallowMoveOrCopy(fullSPath, fullTPath, canMove)
			if err != nil {
				return WrapErrorf(
					err, "Failed to copy \"%s\" to \"%s\".", fullSPath,
					fullTPath)
			}
			// Add s.Value to the target hashes before or after t to preserve
			// the order of the list.
			t = addPathToState(targetState, t, s.Value.(PathHashPair))
			// Remove s from sourceState if necessary and advance 's'
			if moved {
				s, err = removePathFromState(sourceState, s)
				if err != nil {
					return WrapErrorf(
						err, "Failed to remove \"%s\" from sourceState.",
						fullSPath)
				}
				// If the directory is empty after moving the file, it should
				// be added to the source state
				fullSPathDir := filepath.Dir(fullSPath)
				if fullSPathDir != sourcePath {
					files, err := ioutil.ReadDir(fullSPathDir)
					if err != nil {
						return WrapErrorf(
							err,
							"Failed to check if directory is empty \"%s\".",
							fullSPathDir)
					}
					if len(files) == 0 {
						fullSPathDir, _ = filepath.Rel( // Trim root
							sourcePath, fullSPathDir)
						dirEntry := PathHashPair{fullSPathDir, ""}
						if s != nil {
							sourceState.InsertBefore(dirEntry, s)
						} else {
							sourceState.PushBack(dirEntry)
						}
					}
				}
			} else {
				s = s.Next()
			}
		} else if s == nil || s.Value.(PathHashPair).Path > t.Value.(PathHashPair).Path {
			// Source is ahead of the target - the file from target path
			// doesn't exist in the source so we need to delete it.
			fullTPath := filepath.Join(targetPath, t.Value.(PathHashPair).Path)
			// Remove the file
			err := os.RemoveAll(fullTPath)
			if err != nil {
				return WrapErrorf(
					err, "Failed to remove \"%s\".", fullTPath)
			}
			// Remove the element from targetState and advance 't'
			t, err = removePathFromState(targetState, t)
			if err != nil {
				return WrapErrorf(
					err, "Failed to remove \"%s\" from targetState.",
					fullTPath)
			}
		} else {
			// The paths are equal, so compare the hashes and if necessary copy
			// the file from source to the target.
			fullSPath := filepath.Join(sourcePath, s.Value.(PathHashPair).Path)
			fullTPath := filepath.Join(targetPath, s.Value.(PathHashPair).Path)
			sHash := s.Value.(PathHashPair).Hash
			tHash := t.Value.(PathHashPair).Hash
			if sHash == tHash { // Nothing to do, advance 's' and 't'
				s = s.Next()
				t = t.Next()
			} else {
				// Copy the file from source to the target overwriting the
				// the target file.
				moved, err := shallowMoveOrCopy(fullSPath, fullTPath, canMove)
				if err != nil {
					return WrapErrorf(
						err, "Failed to copy \"%s\" to \"%s\".", fullSPath,
						fullTPath)
				}
				// Just overwrite the properties of the target element
				t.Value = s.Value
				// Remove from source if necesary and advance 's' and 't'
				if moved {
					s, err = removePathFromState(sourceState, s)
					if err != nil {
						return WrapErrorf(
							err, "Failed to remove \"%s\" from sourceState.",
							fullSPath)
					}
				} else {
					s = s.Next()
				}
				t = t.Next()
			}
		}
	}
	return nil
}

// LoadPathState loads the state of the file path for the RecycledMoveOrCopy.
// It tries to load it from the cacheFilePath first and if
// it failes, it generates the state based on the actual files. The hashes are
// calculated using the hash interface.
//
// The function calls 'hash.Reset()' before calculating the hash. So the hash
// object doesn't have to be initialized before call.
func LoadPathState(
	cacheFilePath, path string, hash hash.Hash,
) (*list.List, error) {
	// Try to load from cached file
	if cacheFilePath != "" {
		file, err := ioutil.ReadFile(cacheFilePath)
		if err == nil {
			var fullFile map[string][]PathHashPair
			err = json.Unmarshal(file, &fullFile)
			if err != nil {
				Logger.Warnf(
					"Failed to parse file with cached file hashes: %s", err)
			} else {
				slice, ok := fullFile[path]
				if ok {
					result := patHashPairSliceToState(slice)
					return result, nil
				}
			}
		}
	}
	// Load the data from the path
	result, err := getStateFromDirPath(path, hash)
	if err != nil {
		return nil, PassError(err)
	}
	return result, nil
}

// SavePathState appends new entry to the cache file of the RecycledMoveOrCopy.
func SavePathState(cacheFilePath, path string, pairs *list.List) error {
	file, err := ioutil.ReadFile(cacheFilePath)
	var fullFile map[string][]PathHashPair
	if err == nil {
		err = json.Unmarshal(file, &fullFile)
	}
	// Read or marshal error, create empty map
	if err != nil {
		fullFile = make(map[string][]PathHashPair)
	}
	entry, err := stateToPathHashPairSlice(pairs)
	if err != nil {
		return WrapError(
			err, "Failed to convert state to slice for JSON convertsion.")
	}
	fullFile[path] = entry
	file, err = json.Marshal(fullFile)
	if err != nil {
		return WrapErrorf(
			err, "Failed to marshal a file with catched file hashes.")
	}
	err = ioutil.WriteFile(cacheFilePath, file, 0644)
	if err != nil {
		return WrapErrorf(
			err, "Failed to write a file with catched file hashes.")
	}
	return nil
}

// DeepCopyAndGetState copies the files from source to the target path and
// calculates the state of the target path (a list of the PathHashPairs sorted
// by paths). The hash is used to calculate the hashes for the PathHashPairs
// of the state. The target path should be empty.
func DeepCopyAndGetState(
	source, target string, hash hash.Hash,
) (*list.List, error) {
	state := list.New()
	err := filepath.WalkDir(
		source, func(path string, d fs.DirEntry, err error) error {
			if path == source {
				return nil // skip the root directory
			}
			relPath, _ := filepath.Rel(source, path) // shouldn't error
			currTarget := filepath.Join(target, relPath)
			if isDir, err := isDirectory(path); err != nil {
				return WrapErrorf(
					err, "Failed to determine if \"%s\" is a directory.",
					path)
			} else if isDir { // Check if the directory is non-empty
				files, err := ioutil.ReadDir(path)
				if err != nil {
					return WrapErrorf(
						err,
						"Failed to check if directory is empty \"%s\".", path)
				}
				if len(files) != 0 {
					// No need to save info about non-empty directories because
					// their existance is implied by the existance of their
					// children.
					return nil
				}
			}
			hashStr, err := shallowCopyAndGetHash(path, currTarget, hash)
			if err != nil {
				return WrapErrorf(
					err, "Failed to copy \"%s\" to \"%s\"",
					path, currTarget)
			}
			state.PushBack(PathHashPair{Path: relPath, Hash: hashStr})
			return nil
		})
	if err != nil {
		return nil, PassError(err)
	}
	return state, nil
}

// PRIVATE FUNCTIONS

// shallowMoveOrCopy takes source and target paths as arguments and tries to
// move or copy the file from source to target. It returns true if the file was
// moved and false if it was copied.
// If source is a directory the function will create an empty directory at
// target (the copy is shallow so the contents of the source directory don't
// matter).
func shallowMoveOrCopy(source, target string, canMove bool) (bool, error) {
	isDir, err := isDirectory(source)
	if err != nil {
		return false, WrapErrorf(
			err, "Failed to determine if \"%s\" is a directory.",
			source)
	}
	// If the target exists, remove it
	if err == nil {
		err = os.RemoveAll(target)
		if err != nil {
			return false, WrapErrorf(
				err, "Failed to remove \"%s\".", target)
		}
	}
	// If source is a directory then just create a directory in the target
	// no need to copy or move anything.
	if isDir {
		_, err := os.Stat(target)
		// If unable to stat the target but it exists, return an error
		if err != nil && !os.IsNotExist(err) {
			return false, WrapErrorf(
				err, "Failed to stat \"%s\".", target)
		}
		// Create the target directory
		err = os.MkdirAll(target, 0755)
		if err != nil {
			return false, WrapErrorf(
				err, "Failed to create \"%s\".", target)
		}
		return false, nil
	}
	// If moving is allowed then try to move the file
	if canMove {
		err := os.MkdirAll(filepath.Dir(target), 0755)
		if err == nil {
			err = os.Rename(source, target)
			if err == nil {
				return true, nil
			}
		}
	}
	// Move failed or not allowed, copy the file
	err = copyFile(source, target)
	if err != nil {
		return false, WrapErrorf(
			err, "Failed to copy \"%s\" to \"%s\".", source, target)
	}
	return false, nil
}

// removePathFromState takes a list of PathHashPairs (the state) and an element
// to remove from the list, and removes the element from the list. If the
// PathHashPair of the element is a directory, it also removes all of the
// elements, which are in the same directory, from the list.
// The function returns the next element (after the removed ones) and an error.
// If the 'element' is nil, the function instantly returns nil and nil.
func removePathFromState(
	state *list.List, element *list.Element,
) (*list.Element, error) {
	if element == nil {
		return nil, nil
	}
	removeElement := func() { // Shortcut for removing the 'element'
		nextElement := element.Next()
		state.Remove(element)
		element = nextElement
	}
	// Remove the first element
	rootPath := element.Value.(PathHashPair).Path
	removeElement()
	// Check if the first element is a directory
	rootIsDir := rootPath == ""
	if !rootIsDir {
		return element, nil
	}
	// If the first element is a directory, remove the children
	for element != nil {
		elementPath := element.Value.(PathHashPair).Path
		isRel, err := isRelative(elementPath, rootPath)
		if err != nil {
			return nil, WrapErrorf(
				err, "Failed to check if \"%s\" is relative to \"%s\".",
				elementPath, rootPath)
		}
		if !isRel {
			break
		}
		removeElement()
	}
	return element, nil
}

// addPathToState takes a list of PathHashPairs (the state) and inserts the
// entry (PathHashPair) before or after the 'element' into the list. The place
// of insertion is determined by sorting of the PathHashPairs by their paths.
// The function assumes that the 'element' is roughly in the right place and
// adding the entry before or after it won't break the sorting. If the element
// is nil, the entry is added to the end of the list.
// The function returns the next element after the element and the added
// element.
func addPathToState(
	state *list.List, element *list.Element, entry PathHashPair,
) *list.Element {
	// If element is last, then temporarily use the last element of the
	// state to specify where to insert the new element.
	isTLast := element == nil

	if state.Len() != 0 {
		if isTLast {
			element = state.Back()
		}
		// Thanks to sorting we can insert directly after or before t
		if element.Value.(PathHashPair).Path > entry.Path {
			state.InsertBefore(entry, element)
		} else { // "<" it can't be "==" because of the sorting
			state.InsertAfter(entry, element)
			element = element.Next()
		}
	} else {
		state.PushBack(entry)
	}
	// No matter what, t is still the last element
	if isTLast {
		element = nil
	}
	return element
}

// getStateFromDirPath returns a state for the file path (a list of
// PathHashPairs of  the files in the path). The list is sorted alphabetically
// by path.
func getStateFromDirPath(dirPath string, hash hash.Hash) (*list.List, error) {
	if stats, err := os.Stat(dirPath); err != nil {
		return nil, WrapErrorf(err, "Failed to stat \"%s\".", dirPath)
	} else if !stats.IsDir() {
		return nil, WrapErrorf(
			err, "\"%s\" is not a directory.", dirPath)
	}
	result := list.New()
	err := filepath.WalkDir(
		dirPath, func(path string, d fs.DirEntry, err error) error {
			if path == dirPath {
				return nil // skip the root directory
			}
			relPath, _ := filepath.Rel(dirPath, path) // shouldn't error
			if err != nil {
				return WrapErrorf(err, "Failed to walk \"%s\".", path)
			}
			if isDir, err := isDirectory(path); err != nil {
				return WrapErrorf(
					err, "Failed to determine if \"%s\" is a directory.",
					path)
			} else if isDir { // Check if the directory is non-empty
				files, err := ioutil.ReadDir(path)
				if err != nil {
					return WrapErrorf(
						err,
						"Failed to check if directory is empty \"%s\".", path)
				}
				if len(files) != 0 {
					// No need to save info about non-empty directories because
					// their existance is implied by the existance of their
					// children.
					return nil
				}
			}
			hashStr, err := getPathHash(path, hash)
			if err != nil {
				return WrapErrorf(err, "Failed to get hash for \"%s\".", path)
			}
			result.PushBack(PathHashPair{relPath, hashStr})
			return nil
		})
	if err != nil {
		return nil, WrapErrorf(err, "Failed to walk \"%s\".", dirPath)
	}
	return result, nil
}

// getPathHash returns a hash of the file at path. If the file doesn't
// exist, it returns an empty string and an error. If the file is a directory,
// it returns an empty string and nil. If the file is a regular file, it
// returns the hash and nil.
func getPathHash(path string, hash hash.Hash) (string, error) {
	if stat, err := os.Stat(path); err != nil {
		return "", WrapErrorf(err, "Failed to stat \"%s\".", path)
	} else if stat.IsDir() {
		return "", nil // Use empty string as a hash for directories.
	}
	file, err := os.Open(path)
	if err != nil {
		return "", WrapErrorf(err, "Failed to open \"%s\".", path)
	}
	defer file.Close()
	hash.Reset()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", WrapErrorf(err, "Failed to get sha1 hash fo \"%s\".", path)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// shallowCopyAndGetHash copies 'source' to 'target' and returns the hash of
// the copied file. If the source is a directory, it creates an empty directory
// in the target and returns an empty string (the copy is shallow so the
// content of the source doesn't matter in this case).
func shallowCopyAndGetHash(
	source, target string, hash hash.Hash,
) (string, error) {
	// Check if source is a directory, if it is the hash will be empty stirng
	// create matching directory in target.
	stat, err := os.Stat(source)
	if err != nil {
		return "", WrapErrorf(err, "Failed to stat \"%s\".", source)
	}
	if stat.IsDir() {
		err = os.MkdirAll(target, 0755)
		return "", nil
	}

	// Source is a file
	err = os.MkdirAll(filepath.Dir(target), 0755)
	if err != nil {
		return "", WrapErrorf(
			err, "Failed to create \"%s\".", target)
	}
	buf := make([]byte, copyFileBufferSize)
	sourceF, err := os.Open(source)
	if err != nil {
		return "", WrapErrorf(
			err, "Failed to open \"%s\" for reading.", source)
	}
	defer sourceF.Close()
	targetF, err := os.Create(target)
	if err != nil {
		return "", WrapErrorf(
			err, "Failed to open \"%s\" for writing.", target)
	}
	defer targetF.Close()
	hash.Reset()
	for {
		n, err := sourceF.Read(buf)
		if err != nil && err != io.EOF {
			return "", WrapErrorf(err, "Failed to read from \"%s\".", source)
		}
		if n == 0 {
			break
		}
		hash.Write(buf[:n])
		if _, err := targetF.Write(buf[:n]); err != nil {
			return "", WrapErrorf(err, "Failed to write to \"%s\".", target)
		}
	}
	targetF.Sync()
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// isDirectory is a function that returns true if the given path is a
// directory.
func isDirectory(path string) (bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return false, WrapErrorf(err, "Failed to stat \"%s\".", path)
	}
	return stat.IsDir(), nil
}

// isRelative is a function that returns true if the given path is relative to
// dir.
func isRelative(path, dir string) (bool, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return false, WrapErrorf(
			err, "Failed to get absolute path of \"%s\".", path)
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return false, WrapErrorf(
			err, "Failed to get absolute path of \"%s\".", dir)
	}
	return strings.HasPrefix(path, dir), nil
}

// copyFile copies a file from source to target. If it's necessary it creates
// the target directory.
func copyFile(source, target string) error {
	err := os.MkdirAll(filepath.Dir(target), 0755)
	if err != nil {
		return WrapErrorf(
			err, "Failed to create \"%s\".", target)
	}
	buf := make([]byte, copyFileBufferSize)
	sourceF, err := os.Open(source)
	if err != nil {
		return WrapErrorf(
			err, "Failed to open \"%s\" for reading.", source)
	}
	defer sourceF.Close()
	targetF, err := os.Create(target)
	if err != nil {
		return WrapErrorf(
			err, "Failed to open \"%s\" for writing.", target)
	}
	defer targetF.Close()
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

// patHashPairSliceToState converts a slice of PathHashPairs to a list.List.
func patHashPairSliceToState(s []PathHashPair) *list.List {
	l := list.New()
	for _, v := range s {
		l.PushBack(v)
	}
	return l
}

// stateToPathHashPairSlice convertes a list.List (with PathHashPair elements)
// to a slice of PathHashPairs.
func stateToPathHashPairSlice(l *list.List) ([]PathHashPair, error) {
	s := make([]PathHashPair, 0)
	for e := l.Front(); e != nil; e = e.Next() {
		item, ok := e.Value.(PathHashPair)
		if !ok {
			return nil, WrappedError(
				"Failed to convert list element to PathHashPair.")
		}
		s = append(s, item)
	}
	return s, nil
}
