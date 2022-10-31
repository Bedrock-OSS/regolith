package test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
	"github.com/otiai10/copy"
)

// TestConditionalFilter runs a test that checks whether the 'when' property
// of a filter properly locks/enables the execution of the filter.
func TestConditionalFilter(t *testing.T) {
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
	project, err := filepath.Abs(filepath.Join(conditionalFilterPath, "project"))
	if err != nil {
		t.Fatal(
			"Unable to get absolute path to the test project:", err)
	}
	expectedBuildResult, err := filepath.Abs(
		filepath.Join(conditionalFilterPath, "expected_build_result"))
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
	if err := regolith.Run("default", true); err != nil {
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
