package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

func TestRevertibleDeleteDirRollback(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backup")

	// create directory structure to delete
	dir := filepath.Join(tmpDir, "dir")
	err := os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	if err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	err = os.WriteFile(filepath.Join(dir, "sub", "file.txt"), []byte("hello"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	ops, err := regolith.NewRevertibleFsOperations(backupDir)
	if err != nil {
		t.Fatalf("failed to init revertible ops: %v", err)
	}

	if err := ops.Delete(dir); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("directory still exists or stat error: %v", err)
	}

	if err := ops.Undo(); err != nil {
		t.Fatalf("undo failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "file.txt")); err != nil {
		t.Fatalf("directory not restored: %v", err)
	}

	if err := ops.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}
