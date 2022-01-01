package test

import (
	"crypto/md5"
	"encoding/hex"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

const freshProjectPath = "testdata/fresh_project"
const minimalProjectPath = "testdata/minimal_project"
const multitargetProjectPath = "testdata/multitarget_project"
const doubleRemoteProjectPath = "testdata/double_remote_project"
const doubleRemoteProjectInstalledPath = "testdata/double_remote_project_installed"
const runMissingRpProjectPath = "testdata/run_missing_rp_project"

// listPaths returns a dictionary with paths of the files from 'path' directory
// relative to 'root' directory used as keys, and with md5 hashes paths as
// values. The directory paths use empty strings instead of MD5. The function
// ignores files called .ignoreme (they simulate empty directories
// in git repository).
func listPaths(path string, root string) (map[string]string, error) {
	result := map[string]string{}
	err := filepath.WalkDir(path,
		func(path string, data fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if data.Name() == ".ignoreme" { // Ignored file
				return nil
			}
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if data.IsDir() {
				result[relPath] = ""
			} else {
				content, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}
				hash := md5.New()
				// Get the hash value, ignore carriage return
				hash.Write([]byte(strings.Replace(string(content), "\r", "", -1)))
				hashInBytes := hash.Sum(nil)
				result[relPath] = hex.EncodeToString(hashInBytes)
			}
			return nil
		})
	if err != nil {
		return map[string]string{}, err
	}
	return result, nil
}

// comparePathMaps compares maps created by listPaths function and runs
// t.Fatal in case of finding a difference.
func comparePathMaps(
	expectedPaths map[string]string, createdPaths map[string]string,
	t *testing.T,
) {
	checked := struct{}{}
	checklist := map[string]struct{}{}
	// Check if all expectedPaths are created
	for k, expectedHash := range expectedPaths {
		checklist[k] = checked
		createdHash, exists := createdPaths[k]
		if !exists {
			t.Fatal("Missing expected path:", k)
		} else if createdHash != expectedHash {
			if expectedHash == "" {
				t.Fatalf("%q should be a file but is a directory instead", k)
			} else if createdHash == "" {
				t.Fatalf("%q should be a directory but is a file instead", k)
			}
			// Print the file, that doesn't match
			//bytes, _ := ioutil.ReadFile(k)
			//t.Log(string(bytes))
			t.Fatalf("%q file is different that expected", k)
		}
	}
	// Check if all createdPaths are expected
	for k, createdHash := range createdPaths {
		if _, checked := checklist[k]; checked {
			continue // This is checked already (skip)
		}
		expectedHash, exists := expectedPaths[k]
		if !exists {
			t.Fatal("Additional unexpected path was created:", k)
		} else if createdHash != expectedHash {
			if expectedHash == "" {
				t.Fatalf("%q should be a file but is a directory instead", k)
			} else if createdHash == "" {
				t.Fatalf("%q should be a directory but is a file instead", k)
			}
			t.Fatalf("%q file is different that expected", k)
		}
	}
}
