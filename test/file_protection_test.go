package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

// TestSwitchingExportTargets tests if the file protection system won't get
// triggered when switching targets between exporting. It performs the
// following:
// 1. Runs Regolith with target A
// 2. Runs Regolith with target B
// 3. Runs Regolith with target A again
// This test only checks if the regolith run command runs successfuly. It
// doesn't check if the files are exported correctly.
func TestSwitchingExportTargets(t *testing.T) {
	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestSwitchingExportTargets", t)

	t.Log("Copying the project files into the testing directory...")
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(multitargetProjectPath, workingDir, t)

	// Switch to the working directory
	os.Chdir(workingDir)

	// THE TEST
	// Run Regolith with targets: A, B, A
	t.Log("Testing the 'regolith run' with changing export targets...")
	t.Log("Running Regolith with target A...")
	err := regolith.Run("exact_export_A", true)
	if err != nil {
		t.Fatal(
			"Unable RunProfile failed on first attempt to export to A:", err)
	}
	t.Log("Running Regolith with target B...")
	err = regolith.Run("exact_export_B", true)
	if err != nil {
		t.Fatal("Unable RunProfile failed on attempt to export to B:", err)
	}
	t.Log("Running Regolith with target A (2nd time)...")
	err = regolith.Run("exact_export_A", true)
	if err != nil {
		t.Fatal(
			"Unable RunProfile failed on second attempt to export to A:", err)
	}
}

// TestTriggerFileProtection tests if the file protection system will get
// triggered when exporting to a target directory with files created not by
// Regolith. It performs the following:
// 1. Runs Regolith to export something to a target directory.
// 2. Creates a file in the target directory.
// 3. Runs Regolith to export again to the same target directory.
// This test only checks if the regolith run command runs successfuly. It
// doesn't check if the files are exported correctly.
func TestTriggerFileProtection(t *testing.T) {
	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestTriggerFileProtection", t)

	t.Log("Copying the project files into the testing directory...")
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(multitargetProjectPath, workingDir, t)

	// Switch to the working directory
	os.Chdir(workingDir)

	// THE TEST
	// 1. Run Regolith (export to A)
	t.Log("Testing the 'regolith run' with file protection...")
	t.Log("Running Regolith...")
	err := regolith.Run("exact_export_A", true)
	if err != nil {
		t.Fatal(
			"Unable RunProfile failed on first attempt to export to A:", err)
	}

	// 2. Create a file in the target directory (simulate user action).
	t.Log("Creating a file in the target directory (simulating user action)...")
	file, err := os.Create(filepath.Join(tmpDir, "target-a/BP/test-file"))
	if err != nil {
		t.Fatal("Unable to create test file:", err)
	}
	file.Close()

	// 3. Run Regolith (export to A), expect failure.
	t.Log("Running Regolith (this should be stopped by file protection system)...")
	err = regolith.Run("exact_export_A", true)
	if err == nil {
		t.Fatal("Expected RunProfile to fail on second attempt to export to A")
	}
}
