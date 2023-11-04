package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

// TestApplyFilter tests the 'regolith apply-filter' command
func TestApplyFilter(t *testing.T) {
	// Switch to current working directory at the end of the test
	defer os.Chdir(getWdOrFatal(t))

	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestApplyFilter", t)

	t.Log("Copying the project files into the testing directory...")
	project := absOrFatal(filepath.Join(applyFilterPath, "project"), t)
	copyFilesOrFatal(project, tmpDir, t)

	// Load abs path of the expected result and switch to the working directory
	expectedResult := absOrFatal(
		filepath.Join(applyFilterPath, "filtered_project"), t)
	os.Chdir(tmpDir)

	// THE TEST
	t.Log("Running 'rego apply-filter'...")
	if err := regolith.ApplyFilter("test_filter", []string{"Regolith"}, true); err != nil {
		t.Fatal("'regolith apply-filter' failed:", err.Error())
	}
	// TEST EVALUATION
	t.Log("Evaluating the test results...")
	comparePaths(expectedResult, tmpDir, t)
}
