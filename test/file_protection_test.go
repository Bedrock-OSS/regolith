package test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"bedrock-oss.github.com/regolith/regolith"
	"github.com/otiai10/copy"
)

// TestSwitchingExportTargets tests if the file protection system won't get
// triggered when switching targets between exporting. It performs the
// following:
// 1. Runs Regolith with target A
// 2. Runs Regolith with target B
// 3. Runs Regolith with target A again
func TestSwitchingExportTargets(t *testing.T) {
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
	// Before "workingDir" the working dir of this test can't be there
	defer os.RemoveAll(tmpDir)
	defer os.Chdir(wd)
	workingDir := filepath.Join(tmpDir, "working-dir")
	os.Mkdir(workingDir, 0666)
	// Copy the test project to the working directory
	err = copy.Copy(
		multitargetProjectPath,
		workingDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	if err != nil {
		t.Fatalf(
			"Failed to copy test files %q into the working directory %q",
			multitargetProjectPath, workingDir,
		)
	}
	// Switch to the working directory
	os.Chdir(workingDir)
	// Run Regolith with targets: A, B, A
	regolith.InitLogging(true)
	err = regolith.RunProfile("exact_export_A")
	if err != nil {
		t.Fatal(
			"Unable RunProfile failed on first attempt to export to A:", err)
	}
	err = regolith.RunProfile("exact_export_B")
	if err != nil {
		t.Fatal("Unable RunProfile failed on attempt to export to B:", err)
	}
	err = regolith.RunProfile("exact_export_A")
	if err != nil {
		t.Fatal(
			"Unable RunProfile failed on second attempt to export to A:", err)
	}
}
