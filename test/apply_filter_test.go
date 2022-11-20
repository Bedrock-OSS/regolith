package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
	"github.com/otiai10/copy"
)

// TestApplyFilter tests the 'regolith apply-filter' command
func TestApplyFilter(t *testing.T) {
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
	project, err := filepath.Abs(filepath.Join(applyFilterPath, "project"))
	if err != nil {
		t.Fatal(
			"Unable to get absolute path to the test project:", err)
	}
	expectedResult, err := filepath.Abs(
		filepath.Join(applyFilterPath, "filtered_project"))
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
	if err := regolith.ApplyFilter("test_filter", []string{"Regolith"}, true); err != nil {
		t.Fatal("'regolith apply-filter' failed:", err.Error())
	}
	// Load expected result
	expectedPaths, err := listPaths(expectedResult, expectedResult)
	if err != nil {
		t.Fatalf("Failed to load the expected results: %s", err)
	}
	// Load actual result
	actualPaths, err := listPaths(tmpDir, tmpDir)
	if err != nil {
		t.Fatalf("Failed to load the actual results: %s", err)
	}
	// Compare the results
	comparePathMaps(expectedPaths, actualPaths, t)
}
