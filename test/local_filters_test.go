package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
	"github.com/otiai10/copy"
)

// TestRegolithInit tests the results of InitializeRegolithProject against
// the values from test/testdata/fresh_project.
func TestRegolithInit(t *testing.T) {
	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestRegolithInit", t)

	expectedPath := absOrFatal(freshProjectPath, t)
	os.Chdir(tmpDir)

	// THE TEST
	t.Log("Testing the 'regolith init' command...")
	err := regolith.Init(true, false)
	if err != nil {
		t.Fatal("'regolith init' failed:", err.Error())
	}
	comparePaths(expectedPath, ".", t)
}

// TestRegolithRunMissingRp tests the behavior of RunProfile when the packs/RP
// directory is missing. The test just checks if the command runs without
// errors.
func TestRegolithRunMissingRp(t *testing.T) {
	// TEST PREPARATOIN
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestRegolithRunMissingRp", t)

	t.Log("Copying the project files into the testing directory...")
	copyFilesOrFatal(runMissingRpProjectPath, tmpDir, t)
	os.Chdir(tmpDir)

	// THE TEST
	err := regolith.Run("dev", true)
	if err != nil {
		t.Fatal("'regolith run' failed:", err)
	}
}

// TestLocalRequirementsInstallAndRun tests if Regolith properly installs the
// project that uses local script with requirements.txt by running
// "regolith install" first and then "regolith run" on that project.
func TestLocalRequirementsInstallAndRun(t *testing.T) {
	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestLocalRequirementsInstallAndRun", t)

	t.Log("Copying the project files into the testing directory...")
	copyFilesOrFatal(localRequirementsPath, tmpDir, t)
	os.Chdir(filepath.Join(tmpDir, "project"))

	// THE TEST
	t.Log("Testing the 'regolith install-all' command...")
	err := regolith.InstallAll(false, true, false)
	if err != nil {
		t.Fatal("'regolith install-all' failed", err.Error())
	}
	t.Log("Testing the 'regolith run' command...")
	if err := regolith.Run("dev", true); err != nil {
		t.Fatal("'regolith run' failed:", err.Error())
	}
}

// TextExeFilterRun tests if Regolith can properly run an Exe filter
func TestExeFilterRun(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("Unable to get current working directory")
	}
	defer os.Chdir(wd)
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "regolith-test")
	if err != nil {
		t.Fatal("Unable to create temporary directory:", err)
	}
	t.Log("Created temporary directory:", tmpDir)
	// Before deleting "workingDir" the test must stop using it
	defer os.RemoveAll(tmpDir)
	defer os.Chdir(wd)
	// Copy the test project to the working directory
	project, err := filepath.Abs(filepath.Join(exeFilterPath, "project"))
	if err != nil {
		t.Fatal(
			"Unable to get absolute path to the test project:", err)
	}
	expectedBuildResult, err := filepath.Abs(
		filepath.Join(exeFilterPath, "expected_build_result"))
	if err != nil {
		t.Fatal(
			"Unable to get absolute path to the expected build result:", err)
	}
	err = copy.Copy(
		project,
		tmpDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	if err != nil {
		t.Fatalf(
			"Failed to copy test files from %q into the working directory %q",
			project, tmpDir,
		)
	}
	// THE TEST
	os.Chdir(tmpDir)
	if err := regolith.Run("dev", true); err != nil {
		t.Fatal("'regolith run' failed:", err.Error())
	}
	// Load expected result
	expectedPaths, err := getPathHashes(expectedBuildResult)
	if err != nil {
		t.Fatalf("Failed to load the expected results: %s", err)
	}
	// Load actual result
	tmpDirBuild := filepath.Join(tmpDir, "build")
	actualPaths, err := getPathHashes(tmpDirBuild)
	if err != nil {
		t.Fatalf("Failed to load the actual results: %s", err)
	}
	// Compare the results
	comparePathMaps(expectedPaths, actualPaths, t)
}

// TestProfileFilterRun tests valid and invalid profile filters. The invalid
// profile filter has circular dependencies and should fail, the valid profile
// filter runs the same exe file as the TestExeFilterRun test.
func TestProfileFilterRun(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("Unable to get current working directory")
	}
	defer os.Chdir(wd)
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "regolith-test")
	if err != nil {
		t.Fatal("Unable to create temporary directory:", err)
	}
	t.Log("Created temporary directory:", tmpDir)
	// Before deleting "workingDir" the test must stop using it
	defer os.RemoveAll(tmpDir)
	defer os.Chdir(wd)
	// Copy the test project to the working directory
	project, err := filepath.Abs(filepath.Join(profileFilterPath, "project"))
	if err != nil {
		t.Fatal(
			"Unable to get absolute path to the test project:", err)
	}
	expectedBuildResult, err := filepath.Abs(
		filepath.Join(exeFilterPath, "expected_build_result"))
	if err != nil {
		t.Fatal(
			"Unable to get absolute path to the expected build result:", err)
	}
	err = copy.Copy(
		project,
		tmpDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	if err != nil {
		t.Fatalf(
			"Failed to copy test files from %q into the working directory %q",
			project, tmpDir,
		)
	}
	// THE TEST
	os.Chdir(tmpDir)
	t.Log("Running invalid profile filter with circular " +
		"dependencies (this should fail)")
	if err := regolith.Run(
		"invalid_circular_profile_1", true); err == nil {
		t.Fatal("'regolith run' didn't return an error after running"+
			" a circular profile filter:", err.Error())
	} else {
		t.Log("Task failed successfully")
	}
	t.Log("Running valid profile filter ")
	if err := regolith.Run(
		"correct_nested_profile", true); err != nil {
		t.Fatal("'regolith run' failed:", err.Error())
	}
	// Load expected result
	expectedPaths, err := getPathHashes(expectedBuildResult)
	if err != nil {
		t.Fatalf("Failed to load the expected results: %s", err)
	}
	// Load actual result
	tmpDirBuild := filepath.Join(tmpDir, "build")
	actualPaths, err := getPathHashes(tmpDirBuild)
	if err != nil {
		t.Fatalf("Failed to load the actual results: %s", err)
	}
	// Compare the results
	comparePathMaps(expectedPaths, actualPaths, t)
}
