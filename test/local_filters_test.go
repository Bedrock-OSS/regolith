package test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"bedrock-oss.github.com/regolith/regolith"
	"github.com/otiai10/copy"
)

// TestRegolithInit tests the results of InitializeRegolithProject against
// the values from test/testdata/fresh_project.
func TestRegolithInit(t *testing.T) {
	// Switching working directories in this test, make sure to go back
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("Unable to get current working directory")
	}
	defer os.Chdir(wd)
	// Get paths expected in initialized project
	expectedPaths, err := listPaths(
		freshProjectPath, freshProjectPath)
	if err != nil {
		t.Fatal("Unable to get list of created paths:", err)
	}
	// Create temporary directory
	tmpDir, err := ioutil.TempDir("", "regolith-test")
	if err != nil {
		t.Fatal("Unable to create temporary directory:", err)
	}
	t.Log("Created temporary path:", tmpDir)
	// Before removing working dir make sure the script isn't using it anymore
	defer os.RemoveAll(tmpDir)
	defer os.Chdir(wd)

	// Change working directory to the tmp path
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal("Unable to change working directory:", err.Error())
	}
	// THE TEST
	err = regolith.Init(true)
	if err != nil {
		t.Fatal("'regolith init' failed:", err.Error())
	}
	createdPaths, err := listPaths(".", ".")
	if err != nil {
		t.Fatal("Unable to get list of created paths:", err)
	}
	comparePathMaps(expectedPaths, createdPaths, t)
}

// TestRegolithRunMissingRp tests the behavior of RunProfile when the packs/RP
// directory is missing.
func testRegolithRunMissingRp(t *testing.T, recycled bool) {
	// SETUP
	// Switching working directories in this test, make sure to go back
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("Unable to get current working directory")
	}
	defer os.Chdir(wd)
	// Create a temporary directory
	tmpDir, err := ioutil.TempDir("", "regolith-test")
	if err != nil {
		t.Fatal("Unable to create temporary directory:", err)
	}
	t.Log("Created temporary directory:", tmpDir)
	// Before deleting "workingDir" the test must stop using it
	defer os.RemoveAll(tmpDir)
	defer os.Chdir(wd)
	os.Mkdir(tmpDir, 0755)
	// Copy the test project to the working directory
	err = copy.Copy(
		runMissingRpProjectPath,
		tmpDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	if err != nil {
		t.Fatalf(
			"Failed to copy test files %q into the working directory %q",
			runMissingRpProjectPath, tmpDir,
		)
	}
	// Switch to the working directory
	os.Chdir(tmpDir)
	// THE TEST
	err = regolith.Run("dev", recycled, true)
	if err != nil {
		t.Fatal("'regolith run' failed:", err)
	}
}

func TestRegolithRunMissingRp(t *testing.T) {
	testRegolithRunMissingRp(t, false)
}

func TestRegolithRunMissingRpRecycled(t *testing.T) {
	testRegolithRunMissingRp(t, true)
}

// TestLocalRequirementsInstallAndRun tests if Regolith properly installs the
// project that uses local script with requirements.txt by running
// "regolith install" first and then "regolith run" on that project.
func TestLocalRequirementsInstallAndRun(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("Unable to get current working directory")
	}
	defer os.Chdir(wd)
	// Create a temporary directory
	tmpDir, err := ioutil.TempDir("", "regolith-test")
	if err != nil {
		t.Fatal("Unable to create temporary directory:", err)
	}
	t.Log("Created temporary directory:", tmpDir)
	// Before deleting "workingDir" the test must stop using it
	defer os.RemoveAll(tmpDir)
	defer os.Chdir(wd)
	// Copy the test project to the working directory
	err = copy.Copy(
		localRequirementsPath,
		tmpDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	if err != nil {
		t.Fatalf(
			"Failed to copy test files %q into the working directory %q",
			localRequirementsPath, tmpDir,
		)
	}
	// Switch to the working directory
	os.Chdir(filepath.Join(tmpDir, "project"))
	// THE TEST
	err = regolith.InstallAll(false, true)
	if err != nil {
		t.Fatal("'regolith install-all' failed", err.Error())
	}
	if err := regolith.Unlock(true); err != nil {
		t.Fatal("'regolith unlock' failed:", err.Error())
	}
	if err := regolith.Run("dev", false, true); err != nil {
		t.Fatal("'regolith run' failed:", err.Error())
	}
}

// TextExeFilterRun tests if Regolith can properly run an Exe filter
func testExeFilterRun(t *testing.T, recycled bool) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("Unable to get current working directory")
	}
	defer os.Chdir(wd)
	// Create a temporary directory
	tmpDir, err := ioutil.TempDir("", "regolith-test")
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
	if err := regolith.Unlock(true); err != nil {
		t.Fatal("'regolith unlock' failed:", err.Error())
	}
	if err := regolith.Run("dev", recycled, true); err != nil {
		t.Fatal("'regolith run' failed:", err.Error())
	}
	// Load expected result
	expectedPaths, err := listPaths(expectedBuildResult, expectedBuildResult)
	if err != nil {
		t.Fatalf("Failed to load the expected results: %s", err)
	}
	// Load actual result
	tmpDirBuild := filepath.Join(tmpDir, "build")
	actualPaths, err := listPaths(tmpDirBuild, tmpDirBuild)
	if err != nil {
		t.Fatalf("Failed to load the actual results: %s", err)
	}
	// Compare the results
	comparePathMaps(expectedPaths, actualPaths, t)
}

func TestExeFilterRun(t *testing.T) {
	testExeFilterRun(t, false)
}

func TestExeFilterRunRecycled(t *testing.T) {
	testExeFilterRun(t, true)
}

// TestProfileFilterRun tests valid and invalid profile filters. The invalid
// profile filter has circular dependencies and should fail, the valid profile
// filter runs the same exe file as the TestExeFilterRun test.
func testProfileFilterRun(t *testing.T, recycled bool) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("Unable to get current working directory")
	}
	defer os.Chdir(wd)
	// Create a temporary directory
	tmpDir, err := ioutil.TempDir("", "regolith-test")
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
	if err := regolith.Unlock(true); err != nil {
		t.Fatal("'regolith unlock' failed:", err.Error())
	}
	t.Log("Running invalid profile filter with circular " +
		"dependencies (this should fail)")
	if err := regolith.Run(
		"invalid_circular_profile_1", recycled, true); err == nil {
		t.Fatal("'regolith run' didn't return an error after running"+
			" a circular profile filter:", err.Error())
	} else {
		t.Log("Task failed successfully")
	}
	t.Log("Running valid profile filter ")
	if err := regolith.Run(
		"correct_nested_profile", recycled, true); err != nil {
		t.Fatal("'regolith run' failed:", err.Error())
	}
	// Load expected result
	expectedPaths, err := listPaths(expectedBuildResult, expectedBuildResult)
	if err != nil {
		t.Fatalf("Failed to load the expected results: %s", err)
	}
	// Load actual result
	tmpDirBuild := filepath.Join(tmpDir, "build")
	actualPaths, err := listPaths(tmpDirBuild, tmpDirBuild)
	if err != nil {
		t.Fatalf("Failed to load the actual results: %s", err)
	}
	// Compare the results
	comparePathMaps(expectedPaths, actualPaths, t)
}

func TestProfileFilterRun(t *testing.T) {
	testProfileFilterRun(t, false)
}

func TestProfileFilterRunRecycled(t *testing.T) {
	testProfileFilterRun(t, true)
}
