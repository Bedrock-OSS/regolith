package test

import (
	"io/ioutil"
	"os"
	"testing"

	"bedrock-oss.github.com/regolith/regolith"
	"github.com/otiai10/copy"
)

// TestRegolithRunMissingRp tests the behavior of RunProfile when the packs/RP
// directory is missing.
func TestRegolithRunMissingRp(t *testing.T) {
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
	os.Mkdir(tmpDir, 0666)
	// Copy the test project to the working directory
	err = copy.Copy(
		runMissingRpProjectPath,
		tmpDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	if err != nil {
		t.Fatalf(
			"Failed to copy test files %q into the working directory %q",
			multitargetProjectPath, tmpDir,
		)
	}
	// Switch to the working directory
	os.Chdir(tmpDir)

	// THE TEST
	// 1. Run Regolith (export to A)
	regolith.InitLogging(true)
	err = regolith.RunProfile("dev")
	if err != nil {
		t.Fatal(
			"RunProfile failed:", err)
	}
}
