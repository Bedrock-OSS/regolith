package test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
	"github.com/otiai10/copy"
)

// TestInstallAllUnlockAndRun runs a project with a modified config.json file.
// The config file uses a remote filter with a specific version. This test
// tests 'regolith install-all', 'regolith unlock', and 'regolith run'. The
// results of running the filter are compared to a project with an expected
// result.
func TestInstallAllUnlockAndRun(t *testing.T) {
	// Switching working directories in this test, make sure to go back
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("Unable to get current working directory")
	}
	defer os.Chdir(wd)
	// Load expected output
	expectedPaths, err := listPaths(
		versionedRemoteFilterProjectAfterRun, versionedRemoteFilterProjectAfterRun)
	if err != nil {
		t.Fatal("Unable load the expected paths:", err)
	}
	// Create a temporary directory
	tmpDir, err := ioutil.TempDir("", "regolith-test")
	if err != nil {
		t.Fatal("Unable to create temporary directory:", err)
	}
	t.Log("Created temporary directory:", tmpDir)
	// Before deleting "workingDir" the test must stop using it
	defer os.RemoveAll(tmpDir)
	defer os.Chdir(wd)
	workingDir := filepath.Join(tmpDir, "working-dir")
	os.Mkdir(workingDir, 0755)
	// Copy the test project to the working directory
	err = copy.Copy(
		versionedRemoteFilterProject,
		workingDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	if err != nil {
		t.Fatalf(
			"Failed to copy test files %q into the working directory %q",
			versionedRemoteFilterProject, workingDir,
		)
	}
	// Switch to the working directory
	os.Chdir(workingDir)
	// THE TEST
	// Run InstallDependencies
	err = regolith.InstallAll(false, true)
	if err != nil {
		t.Fatal("'regolith install-all' failed:", err)
	}
	err = regolith.Unlock(true)
	if err != nil {
		t.Fatal("'regolith unlock' failed:", err)
	}
	err = regolith.Run("dev", false, true)
	if err != nil {
		t.Fatal("'regolith run' failed:", err)
	}
	// Load created paths for comparison with expected output
	createdPaths, err := listPaths(".", ".")
	if err != nil {
		t.Fatal("Unable to load the created paths:", err)
	}
	// Compare the installed dependencies with the expected dependencies
	comparePathMaps(expectedPaths, createdPaths, t)
}

// TestInstall tests the 'regolith install' command. It forcefully installs
// a filter with various versions and compares the outputs with the expected
// results.
func TestInstall(t *testing.T) {
	// SETUP
	wd, err1 := os.Getwd()
	defer os.Chdir(wd) // Go back before the test ends
	tmpDir, err2 := ioutil.TempDir("", "regolith-test")
	defer os.RemoveAll(tmpDir)
	defer os.Chdir(wd) // 'tmpDir' can't be used when we delete it
	err3 := copy.Copy( // Copy the test files
		freshProjectPath,
		tmpDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	err4 := os.Chdir(tmpDir)
	if err := firstErr(err1, err2, err3, err4); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	t.Logf("The testing directory is in: %s", tmpDir)

	// THE TEST
	filterName := "github.com/Bedrock-OSS/regolith-test-filters/" +
		"hello-version-python-filter"
	var regolithInstallProjects = map[string]string{
		"1.0.0":  "testdata/regolith_install/1.0.0",
		"1.0.1":  "testdata/regolith_install/1.0.1",
		"latest": "testdata/regolith_install/latest",

		// The expected result of the HEAD barnch might change in the future
		// once the test repository is updated. This means that the test data
		// also needs to be updated.
		"HEAD": "testdata/regolith_install/HEAD",
		"0c129227eb90e2f10a038755e4756fdd47e765e6": "testdata/regolith_install/sha",
		"TEST_TAG_1": "testdata/regolith_install/tag",
	}
	for version, expectedResultPath := range regolithInstallProjects {
		expectedResultPath = filepath.Join(wd, expectedResultPath)
		// Install the filter with given version
		err := regolith.Install(
			[]string{filterName + "==" + version}, true, true)
		if err != nil {
			t.Fatal("'regolith install' failed:", err)
		}
		// Load expected result
		expectedPaths, err := listPaths(expectedResultPath, expectedResultPath)
		if err != nil {
			t.Fatalf(
				"Failed to load expected results for version %q: %v",
				version, err)
		}
		// Load created paths for comparison with expected output
		createdPaths, err := listPaths(".", ".")
		if err != nil {
			t.Fatal("Unable to load the created paths:", err)
		}
		// Compare the installed dependencies with the expected dependencies
		comparePathMaps(expectedPaths, createdPaths, t)
	}
}

func TestUpdateAndUpdateAll(t *testing.T) {
	// SETUP
	wd, err1 := os.Getwd()
	defer os.Chdir(wd) // Go back before the test ends
	tmpDir, err2 := ioutil.TempDir("", "regolith-test")
	defer os.RemoveAll(tmpDir)
	defer os.Chdir(wd) // 'tmpDir' can't be used when we delete it
	err3 := copy.Copy( // Copy the test files
		freshProjectPath,
		tmpDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	err4 := os.Chdir(tmpDir)
	if err := firstErr(err1, err2, err3, err4); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	t.Logf("The testing directory is in: %s", tmpDir)

	// THE TEST
	filterName := "hello-version-python-filter"
	var regolithInstallProjects = map[string]string{
		"1.0.0":  "testdata/regolith_update/1.0.0",
		"1.0.1":  "testdata/regolith_update/1.0.1",
		"latest": "testdata/regolith_update/latest",

		// The expected result of the HEAD barnch might change in the future
		// once the test repository is updated. This means that the test data
		// also needs to be updated.
		"HEAD": "testdata/regolith_update/HEAD",
		"0c129227eb90e2f10a038755e4756fdd47e765e6": "testdata/regolith_update/sha",
		"TEST_TAG_1": "testdata/regolith_update/tag",
	}
	updateFunctions := map[string]func([]string) error{
		"update": func(filters []string) error {
			return regolith.Update(filters, true)
		},
		"update-all": func(_ []string) error {
			// UpdateAll doesn't use the "filters" argument
			return regolith.UpdateAll(true)
		},
	}
	for updateFunctionName, updateFunction := range updateFunctions {
		t.Logf("Testing 'regolith %s'", updateFunctionName)
		for version, expectedResultPath := range regolithInstallProjects {
			t.Logf("Testing version %q", version)
			expectedResultPath = filepath.Join(wd, expectedResultPath)
			// Copy the config file from expectedResultPath to tmpDir
			err := copy.Copy(
				filepath.Join(expectedResultPath, "config.json"),
				filepath.Join(tmpDir, "config.json"),
			)
			if err != nil {
				t.Fatal("Failed to copy config file for the test setup:", err)
			}
			// Run 'regolith update' / 'regolith update-all'
			err = updateFunction([]string{filterName})
			if err != nil {
				t.Fatal("'regolith update' failed:", err)
			}
			// Load expected result
			expectedPaths, err := listPaths(
				expectedResultPath, expectedResultPath)
			if err != nil {
				t.Fatalf(
					"Failed to load expected results for version %q: %v",
					version, err)
			}
			// Load created paths for comparison with expected output
			createdPaths, err := listPaths(".", ".")
			if err != nil {
				t.Fatal("Unable to load the created paths:", err)
			}
			// Compare the installed dependencies with the expected
			// dependencies
			comparePathMaps(expectedPaths, createdPaths, t)
		}
	}
}
