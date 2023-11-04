package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
	"github.com/otiai10/copy"
)

// TestInstallAllAndRun runs a project with a modified config.json file.
// The config file uses a remote filter with a specific version. This test
// tests 'regolith install-all', and 'regolith run'. The
// results of running the filter are compared to a project with an expected
// result.
func TestInstallAllAndRun(t *testing.T) {
	// Switching working directories in this test, make sure to go back
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("Unable to get current working directory")
	}
	defer os.Chdir(wd)
	// Load expected output
	expectedPaths, err := getPathHashes(versionedRemoteFilterProjectAfterRun)
	if err != nil {
		t.Fatal("Unable load the expected paths:", err)
	}
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "regolith-test")
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
	err = regolith.InstallAll(false, true, false)
	if err != nil {
		t.Fatal("'regolith install-all' failed:", err)
	}
	err = regolith.Run("dev", true)
	if err != nil {
		t.Fatal("'regolith run' failed:", err)
	}
	// Load created paths for comparison with expected output
	createdPaths, err := getPathHashes(".")
	if err != nil {
		t.Fatal("Unable to load the created paths:", err)
	}
	// Compare the installed dependencies with the expected dependencies
	comparePathMaps(expectedPaths, createdPaths, t)
}

// TestDataModifyRemoteFilter installs a project with one filter using
// 'regolith install-all' and runs it. The filter uses the 'exportData'
// property, which means that the 'data' folder that it modifies should be
// copied back to the source files.
func TestDataModifyRemoteFilter(t *testing.T) {
	// Switching working directories in this test, make sure to go back
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("Unable to get current working directory")
	}
	defer os.Chdir(wd)
	// Load expected output
	expected := filepath.Join(dataModifyRemoteFilter, "after_run")
	expectedPaths, err := getPathHashes(expected)
	if err != nil {
		t.Fatal("Unable load the expected paths:", err)
	}
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "regolith-test")
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
	projectPath := filepath.Join(dataModifyRemoteFilter, "project")
	err = copy.Copy(
		projectPath,
		workingDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	if err != nil {
		t.Fatalf(
			"Failed to copy test files %q into the working directory %q",
			projectPath, workingDir,
		)
	}
	// Switch to the working directory
	os.Chdir(workingDir)
	// THE TEST
	// Run InstallDependencies
	err = regolith.InstallAll(false, true, false)
	if err != nil {
		t.Fatal("'regolith install-all' failed:", err)
	}
	err = regolith.Run("default", true)
	if err != nil {
		t.Fatal("'regolith run' failed:", err)
	}
	// Load created paths for comparison with expected output
	createdPaths, err := getPathHashes(".")
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
	tmpDir, err2 := os.MkdirTemp("", "regolith-test")
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

		// The expected result of the HEAD branch might change in the future
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
			[]string{filterName + "==" + version}, true, false, false, false, []string{"default"}, true)
		if err != nil {
			t.Fatal("'regolith install' failed:", err)
		}
		// Load expected result
		expectedPaths, err := getPathHashes(expectedResultPath)
		if err != nil {
			t.Fatalf(
				"Failed to load expected results for version %q: %v",
				version, err)
		}
		// Load created paths for comparison with expected output
		createdPaths, err := getPathHashes(".")
		if err != nil {
			t.Fatal("Unable to load the created paths:", err)
		}
		// Compare the installed dependencies with the expected dependencies
		comparePathMaps(expectedPaths, createdPaths, t)
	}
}

// TestInstallAll tests the filter updating feature of the 'regolith install-all'
// command. It switches versions of a filter in the config.json file, runs
// 'regolith install-all', and compares the outputs with the expected results.
func TestInstallAll(t *testing.T) {
	// SETUP
	wd, err1 := os.Getwd()
	defer os.Chdir(wd) // Go back before the test ends
	tmpDir, err2 := os.MkdirTemp("", "regolith-test")
	defer os.RemoveAll(tmpDir)
	defer os.Chdir(wd) // 'tmpDir' can't be used when we delete it
	err3 := copy.Copy( // Copy the test files
		filepath.Join(regolithUpdatePath, "fresh_project"),
		tmpDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	err4 := os.Chdir(tmpDir)
	if err := firstErr(err1, err2, err3, err4); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	t.Logf("The testing directory is in: %s", tmpDir)

	// THE TEST
	var regolithInstallProjects = map[string]string{
		"1.0.0":  filepath.Join(regolithUpdatePath, "1.0.0"),
		"1.0.1":  filepath.Join(regolithUpdatePath, "1.0.1"),
		"latest": filepath.Join(regolithUpdatePath, "latest"),

		// The expected result of the HEAD branch might change in the future
		// once the test repository is updated. This means that the test data
		// also needs to be updated.
		"HEAD": filepath.Join(regolithUpdatePath, "HEAD"),
		"0c129227eb90e2f10a038755e4756fdd47e765e6": filepath.Join(
			regolithUpdatePath, "sha"),
		"TEST_TAG_1": filepath.Join(regolithUpdatePath, "tag"),
	}
	t.Log("Testing 'regolith install-all'")
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
		err = regolith.InstallAll(false, true, false)
		if err != nil {
			t.Fatal("'regolith update' failed:", err)
		}
		// Load expected result
		expectedPaths, err := getPathHashes(expectedResultPath)
		if err != nil {
			t.Fatalf(
				"Failed to load expected results for version %q: %v",
				version, err)
		}
		// Load created paths for comparison with expected output
		createdPaths, err := getPathHashes(".")
		if err != nil {
			t.Fatal("Unable to load the created paths:", err)
		}
		// Compare the installed dependencies with the expected
		// dependencies
		comparePathMaps(expectedPaths, createdPaths, t)
	}
}
