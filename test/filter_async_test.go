package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bedrock-OSS/regolith/regolith"
)

func TestAsyncFilter(t *testing.T) {
	// Switch to current working directory at the end of the test
	defer os.Chdir(getWdOrFatal(t))

	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestAsyncFilter", t)

	t.Log("Copying the project files into the testing directory...")
	project := absOrFatal(filepath.Join(asyncFilterPath, "project"), t)
	copyFilesOrFatal(project, tmpDir, t)

	// Load abs path of the expected result and switch to the working directory
	expectedBuildResult := absOrFatal(
		filepath.Join(asyncFilterPath, "expected_build_result"), t)
	os.Chdir(tmpDir)

	// THE TEST
	t.Log("Running Regolith with a conditional filter...")

	start := time.Now()
	if err := regolith.Run("default", []string{}, true); err != nil {
		t.Fatal("'regolith run' failed:", err.Error())
	}
	duration := time.Since(start)
	if duration >= 9*time.Second {
		t.Fatalf("'regolith run' took too long: %v, expected less than 15s", duration)
	}

	// TEST EVALUATION
	t.Log("Evaluating the test results...")
	comparePaths(expectedBuildResult, filepath.Join(tmpDir, "build"), t)
}
