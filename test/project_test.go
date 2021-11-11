package test

import (
	"io/ioutil"
	"os"
	"testing"

	"bedrock-oss.github.com/regolith/regolith"
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
	// Test regolith init
	regolith.InitLogging(true)
	regolith.InitializeRegolithProject(false)
	createdPaths, err := listPaths(".", ".")
	if err != nil {
		t.Fatal("Unable to get list of created paths:", err)
	}
	comparePathMaps(expectedPaths, createdPaths, t)
}
