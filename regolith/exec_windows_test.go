//go:build windows
// +build windows

package regolith

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBatchFileCommand checks that the arguments of a batch file survive
// cmd.exe: npm's launcher on Windows is npm.cmd, and package specifications
// contain characters that cmd.exe treats as operators.
func TestBatchFileCommand(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dir with space")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(dir, "echo.cmd")
	err := os.WriteFile(shim, []byte("@echo off\r\necho [%1][%2][%3]\r\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"@scope/name@^1.0.0", ">=1.0.0 <2.0.0", "a|b&c"}
	cmd, err := batchFileCommand(shim, args...)
	if err != nil {
		t.Fatal(err)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("The batch file failed: %v\n%s", err, output)
	}
	// Batch files keep the quotes of their arguments; executables (node,
	// in npm's case) strip them.
	got := strings.TrimSpace(string(output))
	want := "[\"" + strings.Join(args, "\"][\"") + "\"]"
	if got != want {
		t.Fatalf("Arguments were changed by cmd.exe:\ngot:  %s\nwant: %s", got, want)
	}
	if _, err := batchFileCommand(shim, "a\"b"); err == nil {
		t.Fatal("An argument with a quote should be rejected")
	}
	// '%' is fine until it names a defined variable
	t.Setenv("REGOLITH_TEST_VAR", "x")
	for _, token := range []string{"100%", "%", "a%b%c", "%UNDEFINED_REGOLITH_VAR%"} {
		if _, err := batchFileCommand(shim, token); err != nil {
			t.Fatalf("%q should be accepted: %v", token, err)
		}
	}
	for _, token := range []string{"%REGOLITH_TEST_VAR%", `C:\%REGOLITH_TEST_VAR%\x`, "%a%REGOLITH_TEST_VAR%"} {
		if _, err := batchFileCommand(shim, token); err == nil {
			t.Fatalf("%q would be expanded by cmd.exe and should be rejected", token)
		}
	}
	// Regular executables don't go through cmd.exe (the test binary is one)
	cmd, err = batchFileCommand(os.Args[0], "a^b")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.SysProcAttr != nil || cmd.Args[1] != "a^b" {
		t.Fatal("A regular executable should keep Go's default argument handling")
	}
}
