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
	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestInstallAllAndRun", t)

	t.Log("Copying the project files into the testing directory...")
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(versionedRemoteFilterProject, workingDir, t)

	// Load abs path of the expected result and switch to the working directory
	expectedPath := absOrFatal(versionedRemoteFilterProjectAfterRun, t)
	os.Chdir(workingDir)

	// THE TEST
	t.Log("Testing the 'regolith install-all' command...")
	err := regolith.InstallAll(false, true, false)
	if err != nil {
		t.Fatal("'regolith install-all' failed:", err)
	}

	t.Log("Testing the 'regolith run' command...")
	err = regolith.Run("dev", true)
	if err != nil {
		t.Fatal("'regolith run' failed:", err)
	}

	// TEST EVALUATION
	t.Log("Evaluating the test results...")
	comparePaths(expectedPath, ".", t) // expected vs created paths
}

// TestDataModifyRemoteFilter installs a project with one filter using
// 'regolith install-all' and runs it. The filter uses the 'exportData'
// property, which means that the 'data' folder that it modifies should be
// copied back to the source files.
func TestDataModifyRemoteFilter(t *testing.T) {
	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestDataModifyRemoteFilter", t)

	t.Log("Copying the project files into the testing directory...")
	projectPath := filepath.Join(dataModifyRemoteFilter, "project")
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(projectPath, workingDir, t)

	// Load expected output, and switch to the working directory
	dataModifyRemoteFilterAfterRun := filepath.Join(
		dataModifyRemoteFilter, "after_run")
	expectedPath := absOrFatal(dataModifyRemoteFilterAfterRun, t)
	os.Chdir(workingDir)

	// THE TEST
	t.Log("Testing the 'regolith install-all' command...")
	err := regolith.InstallAll(false, true, false)
	if err != nil {
		t.Fatal("'regolith install-all' failed:", err)
	}

	t.Log("Testing the 'regolith run' command...")
	err = regolith.Run("default", true)
	if err != nil {
		t.Fatal("'regolith run' failed:", err)
	}
	// TEST EVALUATION
	comparePaths(expectedPath, ".", t)
}

// TestInstall tests the 'regolith install' command. It forcefully installs
// a filter with various versions and compares the outputs with the expected
// results.
func TestInstall(t *testing.T) {
	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestInstall", t)

	t.Log("Copying the project files into the testing directory...")
	copyFilesOrFatal(freshProjectPath, tmpDir, t)

	// Switch to the working directory
	originalWd := getWdOrFatal(t)
	os.Chdir(tmpDir)

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
		t.Logf("Testing 'regolith install', filter version %q", version)
		expectedResultPath = filepath.Join(originalWd, expectedResultPath)
		// Install the filter with given version
		err := regolith.Install(
			[]string{filterName + "==" + version}, // Filters list
			true,                                  // Force
			false,                                 // Refresh resolvers
			false,                                 // Refresh filters
			false,                                 // Add to config
			[]string{"default"},                   // Profiles
			true,                                  // Debug
		)
		if err != nil {
			t.Fatal("'regolith install' failed:", err)
		}
		// TEST EVALUATION
		comparePaths(expectedResultPath, ".", t)
	}
}

// TestInstallAll tests the filter updating feature of the 'regolith install-all'
// command. It switches versions of a filter in the config.json file, runs
// 'regolith install-all', and compares the outputs with the expected results.
func TestInstallAll(t *testing.T) {
	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestInstallAll", t)

	t.Log("Copying the project files into the testing directory...")
	copyFilesOrFatal(
		filepath.Join(regolithUpdatePath, "fresh_project"), tmpDir, t)

	// Switch to the working directory
	originalWd := getWdOrFatal(t)
	os.Chdir(tmpDir)

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
	for version, expectedResultPath := range regolithInstallProjects {
		t.Logf("Testing 'regolith install-all', filter version %q", version)
		expectedResultPath = filepath.Join(originalWd, expectedResultPath)

		// User's action: change the config.json file
		t.Log("Simulating user's action (changing the config)...")
		err := copy.Copy(
			filepath.Join(expectedResultPath, "config.json"),
			filepath.Join(tmpDir, "config.json"))
		if err != nil {
			t.Fatal("Failed to copy config file for the test setup:", err)
		}

		// Run 'regolith update' / 'regolith update-all'
		t.Log("Running 'regolith update'...")
		err = regolith.InstallAll(false, true, false)
		if err != nil {
			t.Fatal("'regolith update' failed:", err)
		}
		// TEST EVALUATION
		comparePaths(expectedResultPath, ".", t)
	}
}
