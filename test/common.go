package test

import (
	"crypto/md5"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/otiai10/copy"
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

	// developmentExportTargets is a minimal project with 3 profiles -
	// standard, education and preview. Each profile has different export
	// targets.
	developmentExportTargets = "testdata/development_export_targets"

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

	// customPackNamePath contains two subdirectories 'project' and
	// 'expected_build_result'. The project is a Regolith project with custom
	// rpName and bpName properties in the filter. The 'expected_build_result'
	// contains the expected content of the build directory.
	customPackNamePath = "testdata/custom_pack_name"

	dataModifyRemoteFilter = "testdata/data_modify_remote_filter"

	// sizeTimeCheckOptimizationPath contains two subdirectories 'project' and
	// 'project_after_run'. The project is a Regolith project with a simple
	// Python filter that generates some additional files.The
	// 'project_after_run' is the same project but after running Regolith with
	// the size_time_check experiment enabled.
	sizeTimeCheckOptimizationPath = "testdata/size_time_check_optimization"

	// asyncFilterPath contains two subdirectories 'project' and
	// 'expected_build_result'. They are used for testing the asynchronous
	// filters.
	asyncFilterPath = "testdata/async_filter"
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

// getPathHashes returns a dictionary with paths starting from the 'root'
// used as keys, and with their md5 hashes as values. The directory paths use
// empty strings instead of MD5.
// The function ignores ".ignoreme" and "lockfile.txt" files.
// All paths are relative to the 'root' directory.
func getPathHashes(root string) (map[string]string, error) {
	result := map[string]string{}
	err := filepath.WalkDir(
		root,
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
		},
	)
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
	t.Log("Checking if all expected paths are correct...")
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
	t.Log("Checking if there are no unexpected paths...")
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

// comparePaths compares the paths created by the test with the expected paths
// and runs t.Fatal in case of finding a difference.
func comparePaths(expectedPath, createdPath string, t *testing.T) {
	t.Log("Loading the expected results...")
	expectedPaths, err := getPathHashes(expectedPath)
	if err != nil {
		t.Fatalf(
			"Failed to load expected results for test result evaluation.\n"+
				"Expected path: %v\nError: %v",
			expectedPath, err)
	}
	t.Log("Loading the created paths...")
	createdPaths, err := getPathHashes(createdPath)
	if err != nil {
		t.Fatalf("Failed to load created paths for test result evaluation.\n"+
			"Created path: %v\nError: %v",
			createdPath, err)
	}
	t.Log("Comparing created and expected paths...")
	comparePathMaps(expectedPaths, createdPaths, t)
}

// prepareTestDirectory prepares the test directory by removing all of its files
// or creating it if necessary and returns the path to the directory.
// Exits with t.Fatal in case of error.
func prepareTestDirectory(path string, t *testing.T) string {
	const testResultsDir = "test_results"
	// Create the output directory
	result := filepath.Join(testResultsDir, path)
	if err := os.RemoveAll(result); err != nil {
		t.Fatalf(
			"Failed to delete the files form the testing directory."+
				"\nPath: %q\nError: %v",
			testResultsDir, err)
	}
	if err := os.MkdirAll(result, 0755); err != nil {
		t.Fatalf(
			"Failed to prepare the testing directory.\nPath: %q\nError: %v",
			testResultsDir, err)
	}
	// Get absolute path
	result, err := filepath.Abs(result)
	if err != nil {
		t.Fatalf(
			"Failed to resolve the path of the testing directory to an absolute path."+
				"\nPath: %q\nError: %v",
			testResultsDir, err)
	}
	return result
}

// getWdOrFatal returns the current working directory or exits with t.Fatal in
// case of error.
func getWdOrFatal(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("Unable to get current working directory")
	}
	return wd
}

// copyFilesOrFatal copies files from src to dest or exits with t.Fatal in case
// of error.
func copyFilesOrFatal(src, dest string, t *testing.T) {
	os.MkdirAll(dest, 0755)
	err := copy.Copy(
		src, dest, copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		t.Fatalf(
			"Failed to copy files.\nSource: %s\nDestination: %s\nError: %v",
			src, dest, err)
	}
}

// absOrFatal returns the absolute path of the given path or exits with t.Fatal
// in case of error.
func absOrFatal(path string, t *testing.T) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf(
			"Failed to resolve the path to an absolute path.\n"+
				"Path: %q\nError: %v",
			path, err)
	}
	return abs
}

// assertDirExistsOrFatal asserts that the given path is a directory or exits
// with t.Fatal in case of error.
func assertDirExistsOrFatal(dir string, t *testing.T) {
	if stats, err := os.Stat(dir); err != nil {
		t.Fatalf("Unable to get stats of %q", dir)
	} else if !stats.IsDir() {
		t.Fatalf("Created path %q is not a directory", dir)
	}
}
