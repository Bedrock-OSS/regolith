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
func testSwitchingExportTargets(t *testing.T, recycled bool) {
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
	// THE TEST
	// Run Regolith with targets: A, B, A
	err = regolith.Run("exact_export_A", recycled, true)
	if err != nil {
		t.Fatal(
			"Unable RunProfile failed on first attempt to export to A:", err)
	}
	err = regolith.Run("exact_export_B", recycled, true)
	if err != nil {
		t.Fatal("Unable RunProfile failed on attempt to export to B:", err)
	}
	err = regolith.Run("exact_export_A", recycled, true)
	if err != nil {
		t.Fatal(
			"Unable RunProfile failed on second attempt to export to A:", err)
	}
}

func TestSwitchingExportTargets(t *testing.T) {
	testMoveFilesAcl(t, false)
}

func TestSwitchingExportTargetsRecycled(t *testing.T) {
	testSwitchingExportTargets(t, true)
}

// TestTriggerFileProtection tests if the file protection system will get
// triggered when exporting to a target directory with files created not by
// Regolith. It performs the following:
// 1. Runs Regolith to export something to a target directory.
// 2. Creates a file in the target directory.
// 3. Runs Regolith to export again to the same target directory.
func testTriggerFileProtection(t *testing.T, recycled bool) {
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
	// THE TEST
	// Run Regolith (export to A)
	err = regolith.Run("exact_export_A", recycled, true)
	if err != nil {
		t.Fatal(
			"Unable RunProfile failed on first attempt to export to A:", err)
	}
	// 2. Create a file in the target directory
	file, err := os.Create(filepath.Join(tmpDir, "target-a/BP/test-file"))
	if err != nil {
		t.Fatal("Unable to create test file:", err)
	}
	file.Close()
	// 3. Run Regolith (export to A)
	err = regolith.Run("exact_export_A", recycled, true)
	if err == nil {
		t.Fatal("Expected RunProfile to fail on second attempt to export to A")
	}
}

func TestTriggerFileProtection(t *testing.T) {
	testTriggerFileProtection(t, false)
}

func TestTriggerFileProtectionRecycled(t *testing.T) {
	testTriggerFileProtection(t, true)
}
