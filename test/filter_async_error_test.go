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

	// The implementation now waits for all filters to complete before returning
	// This ensures no goroutines are orphaned and all resources are cleaned up properly
	// Even though one filter fails quickly, we wait for the slow filters (5 seconds)
	if duration < 4*time.Second {
		t.Fatalf("'regolith run' completed too quickly: %v, expected ~5s to wait for all filters", duration)
	}
	if duration > 10*time.Second {
		t.Fatalf("'regolith run' took too long: %v, expected ~5s", duration)
	}

	t.Logf("Duration: %v (waited for all filters to complete)", duration)
}
