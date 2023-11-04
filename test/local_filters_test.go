package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
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
	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestExeFilterRun", t)

	t.Log("Copying the project files into the testing directory...")
	project := absOrFatal(filepath.Join(exeFilterPath, "project"), t)
	copyFilesOrFatal(project, tmpDir, t)

	// Load abs path of the expected result and switch to the working directory
	expectedBuildResult := absOrFatal(
		filepath.Join(exeFilterPath, "expected_build_result"), t)
	os.Chdir(tmpDir)

	// THE TEST
	t.Log("Testing the 'regolith run' command...")
	if err := regolith.Run("dev", true); err != nil {
		t.Fatal("'regolith run' failed:", err.Error())
	}
	// TEST EVALUATION
	t.Log("Evaluating the test results...")
	comparePaths(expectedBuildResult, filepath.Join(tmpDir, "build"), t)
}

// TestProfileFilterRun tests valid and invalid profile filters. The invalid
// profile filter has circular dependencies and should fail, the valid profile
// filter runs the same exe file as the TestExeFilterRun test.
func TestProfileFilterRun(t *testing.T) {
	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestProfileFilterRun", t)

	t.Log("Copying the project files into the testing directory...")
	project := absOrFatal(filepath.Join(profileFilterPath, "project"), t)
	copyFilesOrFatal(project, tmpDir, t)

	// Load abs path of the expected result and switch to the working directory
	expectedBuildResult := absOrFatal(
		filepath.Join(exeFilterPath, "expected_build_result"), t)
	os.Chdir(tmpDir)

	// THE TEST
	// Invalid profile (shoud fail)
	t.Log("Running invalid profile filter with circular dependencies (this should fail).")
	err := regolith.Run("invalid_circular_profile_1", true)
	if err == nil {
		t.Fatal("'regolith run' didn't return an error after running"+
			" a circular profile filter:", err.Error())
	} else {
		t.Log("Task failed successfully")
	}
	// Valid profile (should succeed)
	t.Log("Running valid profile filter.")
	err = regolith.Run("correct_nested_profile", true)
	if err != nil {
		t.Fatal("'regolith run' failed:", err.Error())
	}

	// TEST EVALUATION
	t.Log("Evaluating the test results...")
	comparePaths(expectedBuildResult, filepath.Join(tmpDir, "build"), t)
}
