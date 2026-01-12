package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

func TestLoadEnvFileFromPath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// First independent test run
	t.Run("FirstRun", func(t *testing.T) {
		// Create a .env file with test variables
		envContent := `TEST_PATH_VAR1=path_value1
TEST_PATH_VAR2="quoted_path_value"
`
		envFile := filepath.Join(tempDir, "custom.env")
		err := os.WriteFile(envFile, []byte(envContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create .env file: %v", err)
		}

		// Clear any existing test variables
		os.Unsetenv("TEST_PATH_VAR1")
		os.Unsetenv("TEST_PATH_VAR2")

		// Load the .env file from specific path
		err = regolith.LoadEnvFileFromPath(envFile)
		if err != nil {
			t.Errorf("LoadEnvFileFromPath() returned error: %v", err)
		}

		// Verify variables were loaded
		val, exists := os.LookupEnv("TEST_PATH_VAR1")
		if !exists {
			t.Errorf("TEST_PATH_VAR1 not set")
		}
		if val != "path_value1" {
			t.Errorf("TEST_PATH_VAR1 = %q, want 'path_value1'", val)
		}

		val, exists = os.LookupEnv("TEST_PATH_VAR2")
		if !exists {
			t.Errorf("TEST_PATH_VAR2 not set")
		}
		if val != "quoted_path_value" {
			t.Errorf("TEST_PATH_VAR2 = %q, want 'quoted_path_value'", val)
		}
	})

	// Second independent test run with same variables but different values
	t.Run("SecondRun", func(t *testing.T) {
		// Create a different .env file with same variable names but different values
		envContent2 := `TEST_PATH_VAR1=different_value1
TEST_PATH_VAR2='different_value2'
`
		envFile2 := filepath.Join(tempDir, "custom2.env")
		err := os.WriteFile(envFile2, []byte(envContent2), 0644)
		if err != nil {
			t.Fatalf("Failed to create second .env file: %v", err)
		}

		// Clear any existing test variables
		os.Unsetenv("TEST_PATH_VAR1")
		os.Unsetenv("TEST_PATH_VAR2")

		// Load the different .env file from specific path
		err = regolith.LoadEnvFileFromPath(envFile2)
		if err != nil {
			t.Errorf("LoadEnvFileFromPath() returned error: %v", err)
		}

		// Verify variables were loaded with the NEW values from the second file
		val, exists := os.LookupEnv("TEST_PATH_VAR1")
		if !exists {
			t.Errorf("TEST_PATH_VAR1 not set in second run")
		}
		if val != "different_value1" {
			t.Errorf("TEST_PATH_VAR1 = %q, want 'different_value1' (from second .env file)", val)
		}

		val, exists = os.LookupEnv("TEST_PATH_VAR2")
		if !exists {
			t.Errorf("TEST_PATH_VAR2 not set in second run")
		}
		if val != "different_value2" {
			t.Errorf("TEST_PATH_VAR2 = %q, want 'different_value2' (from second .env file)", val)
		}
	})

	// Test with non-existent file (should not error)
	err := regolith.LoadEnvFileFromPath(filepath.Join(tempDir, "nonexistent.env"))
	if err != nil {
		t.Errorf("LoadEnvFileFromPath() should not error for non-existent file, got: %v", err)
	}
}
