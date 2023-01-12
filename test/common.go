package test

import (
	"crypto/md5"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The ".ignoreme" files inside the test directories are files used to simulate
// empty directories in git repository.
const (
	// freshProjectPath is the regolith project created with `regolith init`
	freshProjectPath = "testdata/fresh_project"

	// minimalProjectPath is the simplest possible valid project, no filters
	// but with addition of *manifest.json* for BP and RP, and with empty file
	// in data path.
	minimalProjectPath = "testdata/minimal_project"

	// multitarget_project is a copy of minimal_project but with modified
	// config.json, to add multiple profiles with different export targets.
	multitargetProjectPath = "testdata/multitarget_project"

	// double_remote_project is a project that uses a remote filter from
	// https://github.com/Bedrock-OSS/regolith-test-filters. The filter has a
	// reference to another remote filter on the same reposiotry.
	doubleRemoteProjectPath = "testdata/double_remote_project"

	// double_remote_project_installed is expected result of contents of
	// double_remote_project after installation.
	doubleRemoteProjectInstalledPath = "testdata/double_remote_project_installed"

	// run_missing_rp_project is a project that for testing "regolith run"
	// which with missing "packs/RP". The profile doesn't have any filters.
	runMissingRpProjectPath = "testdata/run_missing_rp_project"

	localRequirementsPath                = "testdata/local_requirements"
	versionedRemoteFilterProject         = "testdata/versioned_remote_filter_project"
	versionedRemoteFilterProjectAfterRun = "testdata/versioned_remote_filter_project_after_run"
	exeFilterPath                        = "testdata/exe_filter"

	// profileFilterPath is a directory that contains files for testing
	// ProfileFilter. It contains a project and an expected result. The
	// projects have both valid and invalid profiles.
	profileFilterPath = "testdata/profile_filter"

	// regolithUpdatePath is a directory that contains files for testing
	// "regolith install-all" command. It has multiple projects, each with
	// different config.json for installing different versions of the same
	// filter.
	regolithUpdatePath = "testdata/regolith_update"

	// applyFilterPath is a directory that contains the files for testing
	// 'regolith apply-filter' command. It contains two projects, one before running
	// 'regolith apply-filter' command and one after. The command should run the
	// 'test_filter' with 'Regolith' argument. The filter adds a single file
	// with 'Hello Regolith!' greeting.
	applyFilterPath = "testdata/apply_filter"

	// conditionalFilterPath contains two subdirectories 'project' and
	// 'expected_build_result'. The project is a Regolith project with a simple
	// Python filter and with configuration that runs it based on a 'when'
	// condition. The 'expected_build_result' contains the expected result of
	// the execution.
	conditionalFilterPath = "testdata/conditional_filter"

	dataModifyRemoteFilter = "testdata/data_modify_remote_filter"
)

// firstErr returns the first error in a list of errors. If the list is empty
// or all errors are nil, nil is returned.
func firstErr(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

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
			if data.Name() == ".ignoreme" || data.Name() == "lockfile.txt" { // Ignored file
				return nil
			}
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if data.IsDir() {
				result[relPath] = ""
			} else {
				content, err := os.ReadFile(path)
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
