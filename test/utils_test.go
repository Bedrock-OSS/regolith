package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

func TestResolvePath(t *testing.T) {
	// Save original env vars and restore after tests
	originalTestVar := os.Getenv("TEST_VAR")
	originalTestVar2 := os.Getenv("TEST_VAR2")
	defer func() {
		if originalTestVar != "" {
			os.Setenv("TEST_VAR", originalTestVar)
		} else {
			os.Unsetenv("TEST_VAR")
		}
		if originalTestVar2 != "" {
			os.Setenv("TEST_VAR2", originalTestVar2)
		} else {
			os.Unsetenv("TEST_VAR2")
		}
	}()

	tests := []struct {
		name        string
		path        string
		setup       func()
		expected    string
		expectError bool
	}{
		{
			name:        "empty path",
			path:        "",
			setup:       func() {},
			expected:    filepath.Clean("."),
			expectError: false,
		},
		{
			name:        "relative path no env",
			path:        "some/path",
			setup:       func() {},
			expected:    filepath.Clean("some/path"),
			expectError: false,
		},
		{
			name:        "absolute path no env",
			path:        "/absolute/path",
			setup:       func() {},
			expected:    filepath.Clean("/absolute/path"),
			expectError: false,
		},
		{
			name: "path with %VAR% existing",
			path: "/path/%TEST_VAR%/file",
			setup: func() {
				os.Setenv("TEST_VAR", "value")
			},
			expected:    filepath.Clean("/path/value/file"),
			expectError: false,
		},
		{
			name: "path with %VAR% non-existing",
			path: "/path/%TEST_VAR%/file",
			setup: func() {
				os.Unsetenv("TEST_VAR")
			},
			expected:    filepath.Clean(""),
			expectError: true,
		},
		{
			name: "path with $VAR",
			path: "/path/$TEST_VAR/file",
			setup: func() {
				os.Setenv("TEST_VAR", "expanded")
			},
			expected:    filepath.Clean("/path/expanded/file"),
			expectError: false,
		},
		{
			name: "path with ${VAR}",
			path: "/path/${TEST_VAR}/file",
			setup: func() {
				os.Setenv("TEST_VAR", "braced")
			},
			expected:    filepath.Clean("/path/braced/file"),
			expectError: false,
		},
		{
			name: "path with both %VAR% and $VAR",
			path: "/path/%TEST_VAR%/$TEST_VAR2",
			setup: func() {
				os.Setenv("TEST_VAR", "first")
				os.Setenv("TEST_VAR2", "second")
			},
			expected:    filepath.Clean("/path/first/second"),
			expectError: false,
		},
		{
			name: "path with non-existing $VAR",
			path: "/path/$NON_EXISTING/file",
			setup: func() {
				os.Unsetenv("NON_EXISTING")
			},
			expected:    filepath.Clean("/path//file"),
			expectError: false,
		},
		{
			name:        "path with .. for cleaning",
			path:        "some/path/../other",
			setup:       func() {},
			expected:    filepath.Clean("some/other"),
			expectError: false,
		},
		{
			name: "circular %TEST_VAR% references",
			path: "/path/%TEST_VAR%/file",
			setup: func() {
				os.Setenv("TEST_VAR", "%TEST_VAR2%")
				os.Setenv("TEST_VAR2", "%TEST_VAR%")
			},
			expected:    filepath.Clean("/path/%TEST_VAR2%/file"),
			expectError: false,
		},
		{
			name: "starting with %VAR%",
			path: "%TEST_VAR%/file",
			setup: func() {
				os.Setenv("TEST_VAR", "starting")
			},
			expected:    filepath.Clean("starting/file"),
			expectError: false,
		},
		{
			name: "ending with %VAR%",
			path: "path/%TEST_VAR%",
			setup: func() {
				os.Setenv("TEST_VAR", "ending")
			},
			expected:    filepath.Clean("path/ending"),
			expectError: false,
		},
		{
			name:        "path with single % ending with %",
			path:        "path/%",
			setup:       func() {},
			expected:    filepath.Clean("path/%"),
			expectError: false,
		},
		{
			name:        "path with single % starting with %",
			path:        "%/file",
			setup:       func() {},
			expected:    filepath.Clean("%/file"),
			expectError: false,
		},
		{
			name:        "path with single % in the middle",
			path:        "path/%/file",
			setup:       func() {},
			expected:    filepath.Clean("path/%/file"),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result, err := regolith.ResolvePath(tt.path)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}
