package regolith

import (
	"container/list"
	"encoding/hex"
	"encoding/json"
	"hash"
	"hash/crc32"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const defaultHashPairsPath = ".regolith/cache/dir_hash_pairs.json"
const copyFileBufferSize = 1_000_000 // 1 MB

// PathHashPair is a single entry in the list that represents the state of the
// file path. It contains the path and the hash of the file/directory.
type PathHashPair struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

// RecycledMoveOrCopySettings is a structure that defines the settings of the
// FullRecycledMoveOrCopy function.
type RecycledMoveOrCopySettings struct {
	sourceState             *list.List // Preloaded file hashes of source path
	targetState             *list.List // Preloaded target hashes of target path
	hashPairsPath           string     // Path to the file that contains cached hashes
	canMove                 bool       // Whether the files can be moved out of source
	reloadSourceHashes      bool       // Whether the source hashes should be reloaded from file system instead of using cache
	reloadTargetHashes      bool       // Whether the target hashes should be reloaded from file system instead of using cache
	saveSourceHashes        bool       // Whether the source hashes should be saved in the cache
	saveTargetHashes        bool       // Whether the target hashes should be saved in the cache
	hash                    hash.Hash  // Hash object for getting file hash values
	makeTargetReadOnly      bool       // Whether the target files should be made read-only
	copyTargetAclFromParent bool       // Whether the target should copy the security info from it's parent
}

func (s *RecycledMoveOrCopySettings) loadDefaults() {
	if s.hash == nil {
		s.hash = crc32.NewIEEE()
	}
	if s.hashPairsPath == "" {
		s.hashPairsPath = defaultHashPairsPath
	}
}

// FullRecycledMoveOrCopy performs RecycledMoveOrCopy with additionall settings
// it also takes care of backing up the collected file hashes.
func FullRecycledMoveOrCopy(
	sourcePath, targetPath string, settings RecycledMoveOrCopySettings,
) error {
	var err error
	settings.loadDefaults()
	// Create source and target paths if they don't exist
	err = os.MkdirAll(sourcePath, 0755)
	if err != nil {
		return WrapErrorf(err, "Failed to create path \"%s\"", sourcePath)
	}
	err = os.MkdirAll(targetPath, 0755)
	if err != nil {
		return WrapErrorf(err, "Failed to create path \"%s\"", targetPath)
	}
	// Try to load the states from the file if they are not loaded yet
	if settings.sourceState == nil {
		// Try to load from cache if allowed
		if !settings.reloadSourceHashes {
			settings.sourceState, _ = LoadStateFromCache(
				settings.hashPairsPath, sourcePath)
		}
		// If still nil, generate from path
		if settings.sourceState == nil {
			settings.sourceState, err = GetStateFromPath(sourcePath, settings.hash)
			if err != nil {
				return WrapErrorf(
					err, "Failed to load the state of the path %s",
					sourcePath)
			}
		}
	}
	if settings.targetState == nil {
		// Try to load from cache if allowed
		if !settings.reloadTargetHashes {
			settings.targetState, _ = LoadStateFromCache(
				settings.hashPairsPath, targetPath)
		}
		// If still nil, generate from path
		if settings.targetState == nil {
			settings.targetState, err = GetStateFromPath(targetPath, settings.hash)
			if err != nil {
				return WrapErrorf(
					err, "Failed to load the state of the path %s",
					targetPath)
			}
		}
	}

	err = RecycledMoveOrCopy(
		sourcePath, targetPath, settings.sourceState,
		settings.targetState, settings.canMove)
	if err != nil {
		return PassError(err)
	}

	// Save the new hashes if necessary
	if settings.saveSourceHashes {
		err = SavePathState(
			settings.hashPairsPath, sourcePath, settings.sourceState)
		if err != nil {
			return WrapError(err, "Failed to save the state of the files.")
		}
	}
	if settings.saveTargetHashes {
		err = SavePathState(
			settings.hashPairsPath, targetPath, settings.targetState)
		if err != nil {
			return WrapError(err, "Failed to save the state of the files.")
		}
	}
	if settings.copyTargetAclFromParent {
		parent := filepath.Dir(targetPath)
		err = copyFileSecurityInfo(parent, targetPath)
		if err != nil {
			Logger.Warnf(
				"Failed to copy the security info from %s to %s.",
				parent, targetPath)
		}
	}
	if settings.makeTargetReadOnly {
		err := filepath.WalkDir(targetPath,
			func(s string, d fs.DirEntry, e error) error {
				if e != nil {
					return WrapErrorf(
						e, "Failed to walk directory \"%s\".", targetPath)
				}
				if !d.IsDir() {
					os.Chmod(s, 0444)
				}
				return nil
			})
		if err != nil {
			Logger.Warnf(
				"Unable to change file permissions of \"%s\" into read-only",
				targetPath)
		}
	}
	return nil
}

// RecycledMoveOrCopy moves or copies files from source to the target directory
// so that after the operation the state of the target directory as the state
// of the source directory before the operation. It tries to leverage the
// similarity of the source and target to minimize the number of files that
// need to be copied. If canMove flag is set to true, the function doesn't
// care to preserve the state of the source directory.
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
//
// The algorithm can be described with pseudocode:
//    Apply to each element sequentially:
//        T=END | S < T: mv S T; s++
//        S=END | S > T: del T; t++
//        S=T: s++; t++
//
//    Where:
//        - T - element of the source list
//        - S - element of the source list
//        - s++ - point S at the next element of the source list
//        - t++ - point T at the next element of the target list
//        - del - delete the element from list
//        - mv - move the element from source to target
//        - END - end of the list
// It also handles the situations where "mv" fails to move and copies file
// instead (but this is not described in the pseudocode)
func RecycledMoveOrCopy(
	sourcePath, targetPath string,
	sourceState, targetState *list.List,
	canMove bool,
) error {
	if sourceState == nil || targetState == nil {
		return WrappedError(
			"RecycledMoveOrCopy called with nil sourceState or targetState.")
	}
	// Counters for debug messages
	movedFiles := 0
	copiedFiles := 0
	deletedFiles := 0
	skippedFiles := 0

	s := sourceState.Front()
	t := targetState.Front()
	for s != nil || t != nil {
		if t == nil || (s != nil && -1 == compareFilePaths(s.Value.(PathHashPair).Path, t.Value.(PathHashPair).Path)) { // S < T
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
			addPathToState(targetState, t, s.Value.(PathHashPair))
			// Remove s from sourceState if necessary and advance 's'
			if moved {
				// If the file left an empty directory behind, remove it and all of
				// its parent empty directories if there are any.
				if err := removeEmptyDirectoryChain(sourcePath, filepath.Dir(fullSPath)); err != nil {
					return WrapErrorf(
						err, "Failed to remove empty directory and its empty parent directories.")
				}
				movedFiles++
				s, err = removePathFromState(sourceState, s)
				if err != nil {
					return WrapErrorf(
						err, "Failed to remove \"%s\" from sourceState.",
						fullSPath)
				}
			} else { // copied
				copiedFiles++
				s = s.Next()
			}
		} else if s == nil || (t != nil && 1 == compareFilePaths(s.Value.(PathHashPair).Path, t.Value.(PathHashPair).Path)) { // S > T
			// Source is ahead of the target - the file from target path
			// doesn't exist in the source so we need to delete it.
			fullTPath := filepath.Join(targetPath, t.Value.(PathHashPair).Path)
			// Remove the file
			err := os.RemoveAll(fullTPath)
			if err != nil {
				return WrapErrorf(
					err, "Failed to remove \"%s\".", fullTPath)
			}
			// If the file left an empty directory behind, remove it and all of
			// its parent empty directories if there are any.
			if err := removeEmptyDirectoryChain(targetPath, filepath.Dir(fullTPath)); err != nil {
				return WrapErrorf(
					err, "Failed to remove empty directory and its empty parent directories.")
			}
			deletedFiles++
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
				skippedFiles++
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
				if moved {
					// Remove from source if necesary and advance 's' and 't'
					movedFiles++
					s, err = removePathFromState(sourceState, s)
					if err != nil {
						return WrapErrorf(
							err, "Failed to remove \"%s\" from sourceState.",
							fullSPath)
					}
					// If the file left an empty directory behind, remove it and all of
					// its parent empty directories if there are any.
					if err := removeEmptyDirectoryChain(sourcePath, filepath.Dir(fullSPath)); err != nil {
						return WrapErrorf(
							err, "Failed to remove empty directory and its empty parent directories.")
					}
				} else {
					copiedFiles++
					s = s.Next()
				}
				t = t.Next()
			}
		}
	}
	Logger.Debugf(
		"Target: %s; Moved %d; Copied %d; Deleted %d; Skipped (already in target) %d;",
		targetPath, movedFiles, copiedFiles, deletedFiles, skippedFiles)
	return nil
}

// ClearCachedStates clears the defaultHashPairsPath. It doesn't matter if the
// if it's a file or a folder. If the path is cleared successfully or doesn't
// exist returns nil, otherwise returns an error.
func ClearCachedStates() error {
	Logger.Debug("Clearing the cached path states.")
	_, err := os.Stat(defaultHashPairsPath)
	if err == nil {
		isDir, err := isDirectory(defaultHashPairsPath)
		if err == nil {
			if isDir {
				err = os.RemoveAll(defaultHashPairsPath)
			} else {
				err = os.Remove(defaultHashPairsPath)
			}
		}
	} else if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return WrapErrorf(
			err, "Failed to clear path \"%s\"", defaultHashPairsPath)
	}
	return nil
}

// LoadStateFromCache loads the state of the file path for the RecycledMoveOrCopy.
// It tries to load it from the cacheFilePath first and if
// it failes, it generates the state based on the actual files. The hashes are
// calculated using the hash interface.
func LoadStateFromCache(cacheFilePath, path string) (*list.List, error) {
	// Try to load from cached file
	file, err := ioutil.ReadFile(cacheFilePath)
	var fullFile map[string][]PathHashPair
	err = json.Unmarshal(file, &fullFile)
	if err != nil {
		return nil, WrapErrorf(
			err, "Failed to parse file with cached file hashes: %s",
			cacheFilePath)
	}
	slice, ok := fullFile[path]
	if !ok {
		return nil, WrapErrorf(
			err, "Failed to find path \"%s\" in cache file.", path)
	}
	result := patHashPairSliceToState(slice)
	return result, nil
}

// GetStateFromPath returns a state for the file path (a list of
// PathHashPairs of  the files in the path). The list is sorted alphabetically
// by path.
func GetStateFromPath(dirPath string, hash hash.Hash) (*list.List, error) {
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
			relPath, err := filepath.Rel(dirPath, path) // shouldn't error
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
	// create parent of cacheFilePath
	if err := os.MkdirAll(filepath.Dir(cacheFilePath), 0755); err != nil {
		return WrapErrorf(err, "Failed to create parent directory of \"%s\".",
			cacheFilePath)
	}
	// Create the file
	err = ioutil.WriteFile(cacheFilePath, file, 0644)
	if err != nil {
		return WrapErrorf(
			err, "Failed to write a file with catched file hashes.")
	}
	return nil
}

// SaveStateInDefaultCache saves a state of a path in the default cache file
// using the default hash function. If targetPath doesn't exist, it creates
// it before getting the state.
func SaveStateInDefaultCache(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return WrapErrorf(err, "Failed to create directory \"%s\".", path)
	}
	state, err := GetStateFromPath(path, crc32.NewIEEE())
	if err != nil {
		return WrapErrorf(err, "Failed to get state for \"%s\".", path)
	}
	return SavePathState(defaultHashPairsPath, path, state)
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
	err = os.RemoveAll(target)
	if err != nil {
		return false, WrapErrorf(
			err, "Failed to remove \"%s\".", target)
	}
	// If source is a directory then just create a directory in the target
	// no need to copy or move anything.
	if isDir {
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
func removePathFromState(
	state *list.List, element *list.Element,
) (*list.Element, error) {
	removeElement := func() { // Shortcut for removing the 'element'
		nextElement := element.Next()
		state.Remove(element)
		element = nextElement
	}
	// Remove the first element
	rootPath := element.Value.(PathHashPair).Path
	rootIsDir := element.Value.(PathHashPair).Hash == ""
	removeElement()
	// Check if the first element is a directory
	if !rootIsDir {
		return element, nil
	}
	// If the first element is a directory, remove the children
	for element != nil { // element is nil when whe reach the end of the list
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
// entry (PathHashPair) before or after the 'element' into the list. If the
// element is nil, it inserts the entry at the end of the list.
func addPathToState(
	state *list.List, element *list.Element, entry PathHashPair) {
	// element is nil when the list if empty or it's the last elemetn
	if element == nil {
		state.PushBack(entry)
	} else {
		state.InsertBefore(entry, element)
	}
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
		return "", WrapErrorf(err, "Failed to get hash for \"%s\".", path)
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

// removeEmptyDirectoryChain removes takes two paths, a root and a path
// relative the root. If the path is an empty directory, it gets removed,
// if the parent of that path is also an empty directory relative to the
// root, it also gets removed and so on.
func removeEmptyDirectoryChain(root string, path string) error {
	for {

		if isRel, err := isRelative(path, root); err != nil {
			return WrapErrorf(err, "Failed to check if \"%s\" is relative to \"%s\".", path, root)
		} else if isRel && path != root {
			if isEmpty, err := isDirEmpty(path); err != nil {
				return WrapErrorf(err, "Failed to check if \"%s\" is empty.", path)
			} else if isEmpty {
				os.Remove(path)
				// Running this code for the first time is so fucking scary.
				// I hope It won't wipe out my drive.
				path = filepath.Dir(path)
			} else {
				break // path is not empty
			}
		} else {
			break // path is not relative to root
		}
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
