package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

func TestLoadEnvFile(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	defer os.Chdir(originalDir)

	// Change to temporary directory
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Test 1: No .env file exists (should return nil)
	err = regolith.LoadEnvFile()
	if err != nil {
		t.Errorf("LoadEnvFile() should not error when .env doesn't exist, got: %v", err)
	}

	// Test 2: Create a .env file with test variables
	envContent := `# This is a comment
TEST_VAR1=value1
TEST_VAR2="quoted_value"
TEST_VAR3='single_quoted'
# Another comment
TEST_VAR4=value4
EMPTY_VAR=
`
	envFile := filepath.Join(tempDir, ".env")
	err = os.WriteFile(envFile, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	// Clear any existing test variables
	os.Unsetenv("TEST_VAR1")
	os.Unsetenv("TEST_VAR2")
	os.Unsetenv("TEST_VAR3")
	os.Unsetenv("TEST_VAR4")
	os.Unsetenv("EMPTY_VAR")

	// Load the .env file
	err = regolith.LoadEnvFile()
	if err != nil {
		t.Errorf("LoadEnvFile() returned error: %v", err)
	}

	// Verify variables were loaded
	tests := []struct {
		key   string
		want  string
		want2 string
	}{
		{"TEST_VAR1", "value1", "value1"},
		{"TEST_VAR2", "quoted_value", "quoted_value"},
		{"TEST_VAR3", "single_quoted", "single_quoted"},
		{"TEST_VAR4", "value4", "value4"},
		{"EMPTY_VAR", "", ""},
	}

	for _, tt := range tests {
		got, exists := os.LookupEnv(tt.key)
		if !exists {
			t.Errorf("Environment variable %s not set", tt.key)
		}
		if got != tt.want {
			t.Errorf("Environment variable %s = %q, want %q", tt.key, got, tt.want)
		}
	}

	// Test 3: Verify priority - existing env vars should not be overwritten
	os.Setenv("PRIORITY_TEST", "original_value")
	envContent2 := `PRIORITY_TEST=new_value
NEW_VAR=new_value
`
	err = os.WriteFile(envFile, []byte(envContent2), 0644)
	if err != nil {
		t.Fatalf("Failed to update .env file: %v", err)
	}

	err = regolith.LoadEnvFile()
	if err != nil {
		t.Errorf("LoadEnvFile() returned error: %v", err)
	}

	// PRIORITY_TEST should still be "original_value"
	val, exists := os.LookupEnv("PRIORITY_TEST")
	if !exists {
		t.Errorf("PRIORITY_TEST not set")
	}
	if val != "original_value" {
		t.Errorf("PRIORITY_TEST = %q, want 'original_value' (should not be overwritten)", val)
	}

	// NEW_VAR should be set to "new_value"
	val, exists = os.LookupEnv("NEW_VAR")
	if !exists {
		t.Errorf("NEW_VAR not set")
	}
	if val != "new_value" {
		t.Errorf("NEW_VAR = %q, want 'new_value'", val)
	}
}

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
