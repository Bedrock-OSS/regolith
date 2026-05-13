package test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

const prePostShellOsSpecificPath = "testdata/pre_post_shell_os_specific"

// TestPrePostShellCommandsOSSpecific tests OS-specific shell commands
func TestPrePostShellCommandsOSSpecific(t *testing.T) {
	// Switch to current working directory at the end of the test
	defer os.Chdir(getWdOrFatal(t))
	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestPrePostShellCommandsOSSpecific", t)

	t.Log("Copying the project files into the testing directory...")
	project := absOrFatal(filepath.Join(prePostShellOsSpecificPath, "project"), t)
	copyFilesOrFatal(project, tmpDir, t)
	os.Chdir(tmpDir)

	// THE TEST
	t.Log("Testing OS-specific preShell and postShell commands...")
	if err := regolith.Run("default", nil, true, "", false, false); err != nil {
		t.Fatal("'regolith run' failed:", err.Error())
	}

	// TEST EVALUATION
	t.Log("Evaluating the test results...")

	// Check if OS-specific preShell output file exists
	osOutputPath := filepath.Join(tmpDir, "os_output.txt")
	if _, err := os.Stat(osOutputPath); os.IsNotExist(err) {
		t.Fatal("OS-specific preShell command did not create os_output.txt")
	}
	osContent, err := os.ReadFile(osOutputPath)
	if err != nil {
		t.Fatal("Failed to read os_output.txt:", err)
	}

	// Files are now UTF-8 encoded
	osContentStr := string(osContent)
	t.Logf("OS-specific preShell output: %s", osContentStr)

	// Verify correct OS was detected
	switch runtime.GOOS {
	case "windows":
		if !strings.Contains(osContentStr, "Windows") {
			t.Fatal("Windows-specific preShell did not execute correctly")
		}
	case "linux":
		if !strings.Contains(osContentStr, "Linux") {
			t.Fatal("Linux-specific preShell did not execute correctly")
		}
	case "darwin":
		if !strings.Contains(osContentStr, "macOS") {
			t.Fatal("macOS-specific preShell did not execute correctly")
		}
	}

	// Check if OS-specific postShell output file exists
	postOsOutputPath := filepath.Join(tmpDir, "post_os_output.txt")
	if _, err := os.Stat(postOsOutputPath); os.IsNotExist(err) {
		t.Fatal("OS-specific postShell command did not create post_os_output.txt")
	}
	postOsContent, err := os.ReadFile(postOsOutputPath)
	if err != nil {
		t.Fatal("Failed to read post_os_output.txt:", err)
	}

	// Files are now UTF-8 encoded
	postOsContentStr := string(postOsContent)
	t.Logf("OS-specific postShell output: %s", postOsContentStr)

	// Verify postShell received the OS variable from preShell
	switch runtime.GOOS {
	case "windows":
		if !strings.Contains(postOsContentStr, "Windows") {
			t.Fatal("Windows-specific postShell did not receive OS_NAME from preShell")
		}
	case "linux":
		if !strings.Contains(postOsContentStr, "Linux") {
			t.Fatal("Linux-specific postShell did not receive OS_NAME from preShell")
		}
	case "darwin":
		if !strings.Contains(postOsContentStr, "macOS") {
			t.Fatal("macOS-specific postShell did not receive OS_NAME from preShell")
		}
	}

	t.Log("✓ OS-specific shell commands executed correctly!")
	t.Log("Test passed successfully!")
}
