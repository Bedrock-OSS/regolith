package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

const prePostShellPath = "testdata/pre_post_shell"

// TestPrePostShellCommands tests if preShell and postShell commands are executed
// properly before and after the filter pipeline.
func TestPrePostShellCommands(t *testing.T) {
	// Switch to current working directory at the end of the test
	defer os.Chdir(getWdOrFatal(t))
	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestPrePostShellCommands", t)

	t.Log("Copying the project files into the testing directory...")
	project := absOrFatal(filepath.Join(prePostShellPath, "project"), t)
	copyFilesOrFatal(project, tmpDir, t)
	os.Chdir(tmpDir)

	// THE TEST
	t.Log("Testing the 'regolith run' command with preShell and postShell...")
	if err := regolith.Run("default", nil, true, ""); err != nil {
		t.Fatal("'regolith run' failed:", err.Error())
	}

	// TEST EVALUATION
	t.Log("Evaluating the test results...")

	// Check if preShell output file exists and contains expected content
	preOutputPath := filepath.Join(tmpDir, "pre_output.txt")
	if _, err := os.Stat(preOutputPath); os.IsNotExist(err) {
		t.Fatal("preShell command did not create pre_output.txt")
	}
	preContent, err := os.ReadFile(preOutputPath)
	if err != nil {
		t.Fatal("Failed to read pre_output.txt:", err)
	}

	// Files are now UTF-8 encoded
	preContentStr := string(preContent)
	t.Logf("preShell output: %s", preContentStr)

	// Verify that environment variables were set in preShell
	if !strings.Contains(preContentStr, "hello_from_preshell") {
		t.Fatalf("preShell did not set TEST_VAR correctly")
	}
	if !strings.Contains(preContentStr, "another_value") {
		t.Fatalf("preShell did not set ANOTHER_VAR correctly")
	}

	// Check if postShell output file exists and contains expected content
	postOutputPath := filepath.Join(tmpDir, "post_output.txt")
	if _, err := os.Stat(postOutputPath); os.IsNotExist(err) {
		t.Fatal("postShell command did not create post_output.txt")
	}
	postContent, err := os.ReadFile(postOutputPath)
	if err != nil {
		t.Fatal("Failed to read post_output.txt:", err)
	}

	// Files are now UTF-8 encoded
	postContentStr := string(postContent)
	t.Logf("postShell output: %s", postContentStr)

	// Verify that environment variables from preShell are available in postShell
	if !strings.Contains(postContentStr, "hello_from_preshell") {
		t.Fatal("postShell did not receive TEST_VAR from preShell - environment variables did not persist!")
	}
	if !strings.Contains(postContentStr, "another_value") {
		t.Fatal("postShell did not receive ANOTHER_VAR from preShell - environment variables did not persist!")
	}

	t.Log("✓ Environment variables successfully persisted from preShell to postShell!")

	// Verify that export happened (build directory should exist)
	// Note: Regolith appends project name suffix to pack directories
	buildBPPath := filepath.Join(tmpDir, "build", "pre_post_shell_test_bp")
	if _, err := os.Stat(buildBPPath); os.IsNotExist(err) {
		t.Fatal("Export did not create build/pre_post_shell_test_bp directory")
	}
	buildRPPath := filepath.Join(tmpDir, "build", "pre_post_shell_test_rp")
	if _, err := os.Stat(buildRPPath); os.IsNotExist(err) {
		t.Fatal("Export did not create build/pre_post_shell_test_rp directory")
	}

	t.Log("Test passed successfully!")
}
