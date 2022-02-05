package test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"bedrock-oss.github.com/regolith/regolith"
	"github.com/otiai10/copy"
)

// TestDoubleRemoteFilter tests how regolith handles installing a remote
// with dependencies that point at another remote filter.
/*
func TestDoubleRemoteFilter(t *testing.T) {
	// Switching working directories in this test, make sure to go back
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("Unable to get current working directory")
	}
	defer os.Chdir(wd)
	// Load expected output
	expectedPaths, err := listPaths(
		doubleRemoteProjectInstalledPath, doubleRemoteProjectInstalledPath)
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
	os.Mkdir(workingDir, 0666)
	// Copy the test project to the working directory
	err = copy.Copy(
		doubleRemoteProjectPath,
		workingDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	if err != nil {
		t.Fatalf(
			"Failed to copy test files %q into the working directory %q",
			doubleRemoteProjectPath, workingDir,
		)
	}
	// Switch to the working directory
	os.Chdir(workingDir)
	// Run InstallDependencies
	regolith.InitLogging(true)
	config := regolith.ConfigFromObject(regolith.LoadConfigAsMap())
	config.InstallFilters(false, false)
	// Load created paths for comparison with expected output
	createdPaths, err := listPaths(".", ".")
	if err != nil {
		t.Fatal("Unable to load the created paths:", err)
	}
	// Compare the installed dependencies with the expected dependencies
	comparePathMaps(expectedPaths, createdPaths, t)
}
*/

func TestVersionedRemoteFilter(t *testing.T) {
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
	os.Mkdir(workingDir, 0666)
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
	// Run InstallDependencies
	regolith.InitLogging(true)
	config := regolith.ConfigFromObject(regolith.LoadConfigAsMap())
	err = config.InstallFilters(false, false)
	if err != nil {
		t.Fatal("Unable to install filters:", err)
	}
	err = regolith.Unlock()
	if err != nil {
		t.Fatal("Unable to unlock:", err)
	}
	err = regolith.RunProfile("dev")
	if err != nil {
		t.Fatal("Unable to run profile:", err)
	}
	// Load created paths for comparison with expected output
	createdPaths, err := listPaths(".", ".")
	if err != nil {
		t.Fatal("Unable to load the created paths:", err)
	}
	// Compare the installed dependencies with the expected dependencies
	comparePathMaps(expectedPaths, createdPaths, t)
}
