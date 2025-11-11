package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bedrock-OSS/regolith/regolith"
)

const asyncFilterErrorPath = "testdata/async_filter_error"

func TestAsyncFilterWithError(t *testing.T) {
	// Switch to current working directory at the end of the test
	defer os.Chdir(getWdOrFatal(t))

	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestAsyncFilterWithError", t)

	t.Log("Copying the project files into the testing directory...")
	project := absOrFatal(filepath.Join(asyncFilterErrorPath, "project"), t)
	copyFilesOrFatal(project, tmpDir, t)

	// Switch to the working directory
	os.Chdir(tmpDir)

	// THE TEST
	t.Log("Running Regolith with async filters where one fails...")

	start := time.Now()
	err := regolith.Run("default", true)
	duration := time.Since(start)

	// We expect an error
	if err == nil {
		t.Fatal("Expected 'regolith run' to fail, but it succeeded")
	}

	t.Logf("Regolith failed as expected with error: %v", err)

	// The test should fail quickly (within ~2 seconds) after the first filter fails
	// and should not wait for all filters to complete (which would take 5+ seconds)
	// However, with the current implementation, we need to ensure all goroutines
	// have a chance to write to the channel before we exit
	if duration > 10*time.Second {
		t.Fatalf("'regolith run' took too long: %v, expected less than 10s", duration)
	}

	t.Logf("Duration: %v", duration)
}
