package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

// TestConditionalFilter runs a test that checks whether the 'when' property
// of a filter properly locks/enables the execution of the filter.
func TestConditionalFilter(t *testing.T) {
	// Switch to current working directory at the end of the test
	defer os.Chdir(getWdOrFatal(t))

	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestConditionalFilter", t)

	t.Log("Copying the project files into the testing directory...")
	project := absOrFatal(filepath.Join(conditionalFilterPath, "project"), t)
	copyFilesOrFatal(project, tmpDir, t)

	// Load abs path of the expected result and switch to the working directory
	expectedBuildResult := absOrFatal(
		filepath.Join(conditionalFilterPath, "expected_build_result"), t)
	os.Chdir(tmpDir)

	// THE TEST
	t.Log("Running Regolith with a conditional filter...")
	if err := regolith.Run("default", true); err != nil {
		t.Fatal("'regolith run' failed:", err.Error())
	}

	// TEST EVALUATION
	t.Log("Evaluating the test results...")
	comparePaths(expectedBuildResult, filepath.Join(tmpDir, "build"), t)
}
