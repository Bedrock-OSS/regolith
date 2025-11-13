package test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bedrock-OSS/regolith/regolith"
)

// createBigFile creates a text file filled with a's with the given size in MB.
func createBigFile(sizeMB int, path string) error {
	// Create the file
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(bytes.Repeat([]byte("a"), sizeMB*1024*1024))
	if err != nil {
		return err
	}
	return nil
}

// TestSizeTimeCheckOptimizationCorectness tests if running Regolith with the
// size_time_check experiment enabled exports the files correctly.
func TestSizeTimeCheckOptimizationCorectness(t *testing.T) {
	// Switch to current working directory at the end of the test
	defer os.Chdir(getWdOrFatal(t))

	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestSizeTimeCheckOptimizationCorectness", t)

	t.Log("Copying the project files into the testing directory...")
	project := absOrFatal(filepath.Join(sizeTimeCheckOptimizationPath, "project"), t)
	copyFilesOrFatal(project, tmpDir, t)

	// Load abs path of the expected result and switch to the working directory
	expectedBuildResult := absOrFatal(
		filepath.Join(sizeTimeCheckOptimizationPath, "project_after_run"), t)
	os.Chdir(tmpDir)

	// THE TEST
	// Enable the experiment
	regolith.EnabledExperiments = append(regolith.EnabledExperiments, "size_time_check")

	// Run the project
	t.Log("Running Regolith...")

	if err := regolith.Run("default", []string{}, true); err != nil {
		t.Fatal("'regolith run' failed:", err.Error())
	}
	// TEST EVALUATION
	t.Log("Evaluating the result...")
	comparePaths(expectedBuildResult, tmpDir, t)
}

// TestSizeTimeCheckOptimizationSpeed tests if running Regolith with the
// size_time_check experiment enabled is faster on the second run (when the
// files are not changed in relation to the first run).
func TestSizeTimeCheckOptimizationSpeed(t *testing.T) {
	// Switch to current working directory at the end of the test
	defer os.Chdir(getWdOrFatal(t))

	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestSizeTimeCheckOptimizationSpeed", t)

	t.Log("Copying the project files into the testing directory...")
	project := absOrFatal(filepath.Join(sizeTimeCheckOptimizationPath, "project"), t)
	copyFilesOrFatal(project, tmpDir, t)

	// Add a big file to the project 10 MB
	t.Log("Creating a big file in the test project...")
	bigFilePath := filepath.Join(tmpDir, "packs/BP/big_file.txt")
	if err := createBigFile(10, bigFilePath); err != nil {
		t.Fatalf("Creating a big file failed: %v", err)
	}
	os.Chdir(tmpDir)

	// THE TEST
	// Enable the experiment
	regolith.EnabledExperiments = append(regolith.EnabledExperiments, "size_time_check")

	// Run the project twice, the second run should be faster
	runtimes := make([]time.Duration, 0)
	for i := range 2 {
		// Run the project
		t.Logf("Running Regolith for the %d. time...", i+1)

		// Start the timer
		start := time.Now()
		if err := regolith.Run("default", []string{}, true); err != nil {
			t.Fatal("'regolith run' failed:", err.Error())
		}
		// Stop the timer
		runtimes = append(runtimes, time.Since(start))
	}
	// Check if the second run was faster. It should be because in the second
	// run files are not copied (because they don't change).
	if runtimes[0] < runtimes[1] {
		t.Fatalf("The second run was slower than the first one: %v < %v",
			runtimes[0], runtimes[1])
	} else {
		t.Logf("The second run was faster than the first one: %v > %v",
			runtimes[0], runtimes[1])
	}
}
