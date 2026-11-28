package regolith

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureEmptyDirectoryCreatesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing", "nested")
	if err := ensureEmptyDirectory(dir); err != nil {
		t.Fatalf("ensureEmptyDirectory(%q) failed: %v", dir, err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("os.Stat(%q) failed: %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %q to be a directory", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) failed: %v", dir, err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected %q to be empty, got %d entries", dir, len(entries))
	}
}

func TestEnsureEmptyDirectoryEmptiesExistingDir(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "target")
	nestedDir := filepath.Join(dir, "sub", "deep")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stale.txt"), []byte("stale"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "deep.txt"), []byte("deep"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	externalDir := filepath.Join(base, "external")
	if err := os.MkdirAll(externalDir, 0755); err != nil {
		t.Fatalf("MkdirAll external failed: %v", err)
	}
	externalFile := filepath.Join(externalDir, "keep.txt")
	if err := os.WriteFile(externalFile, []byte("keep"), 0644); err != nil {
		t.Fatalf("WriteFile external failed: %v", err)
	}
	if err := os.Symlink(externalDir, filepath.Join(dir, "linked")); err != nil {
		t.Fatalf("Symlink failed: %v", err)
	}

	if err := ensureEmptyDirectory(dir); err != nil {
		t.Fatalf("ensureEmptyDirectory(%q) failed: %v", dir, err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) failed: %v", dir, err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected %q to be empty, got %d entries", dir, len(entries))
	}
	data, err := os.ReadFile(externalFile)
	if err != nil || string(data) != "keep" {
		t.Fatalf("expected external symlink target file %q to stay intact, got data=%q err=%v", externalFile, string(data), err)
	}
}

func TestEnsureEmptyDirectoryReplacesRegularFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not_a_dir")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := ensureEmptyDirectory(filePath); err != nil {
		t.Fatalf("ensureEmptyDirectory(%q) failed: %v", filePath, err)
	}
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("os.Stat(%q) failed: %v", filePath, err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %q to be replaced with a directory", filePath)
	}
	entries, err := os.ReadDir(filePath)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) failed: %v", filePath, err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected %q to be empty, got %d entries", filePath, len(entries))
	}
}

func TestEnsureEmptyDirectoryPreservesSymlinkAndEmptiesTarget(t *testing.T) {
	cases := []struct {
		name     string
		relative bool
	}{
		{name: "absolute symlink", relative: false},
		{name: "relative symlink", relative: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base := t.TempDir()
			targetDir := filepath.Join(base, "real_target")
			if err := os.MkdirAll(filepath.Join(targetDir, "nested"), 0755); err != nil {
				t.Fatalf("MkdirAll failed: %v", err)
			}
			if err := os.WriteFile(filepath.Join(targetDir, "stale.txt"), []byte("stale"), 0644); err != nil {
				t.Fatalf("WriteFile failed: %v", err)
			}
			if err := os.WriteFile(filepath.Join(targetDir, "nested", "item.txt"), []byte("item"), 0644); err != nil {
				t.Fatalf("WriteFile failed: %v", err)
			}

			linkParent := filepath.Join(base, "tmp")
			if err := os.MkdirAll(linkParent, 0755); err != nil {
				t.Fatalf("MkdirAll linkParent failed: %v", err)
			}
			linkPath := filepath.Join(linkParent, "RP")

			linkTarget := targetDir
			if tc.relative {
				rel, err := filepath.Rel(linkParent, targetDir)
				if err != nil {
					t.Fatalf("filepath.Rel failed: %v", err)
				}
				linkTarget = rel
			}
			if err := os.Symlink(linkTarget, linkPath); err != nil {
				t.Fatalf("os.Symlink failed: %v", err)
			}

			if err := ensureEmptyDirectory(linkPath); err != nil {
				t.Fatalf("ensureEmptyDirectory(%q) failed: %v", linkPath, err)
			}

			linfo, err := os.Lstat(linkPath)
			if err != nil {
				t.Fatalf("os.Lstat(%q) failed: %v", linkPath, err)
			}
			if linfo.Mode()&(fs.ModeSymlink|fs.ModeIrregular) == 0 {
				t.Fatalf("expected %q to remain a symlink, got mode %v", linkPath, linfo.Mode())
			}

			targetInfo, err := os.Stat(linkPath)
			if err != nil || !targetInfo.IsDir() {
				t.Fatalf("expected %q to resolve to a directory, got err=%v", linkPath, err)
			}

			entries, err := os.ReadDir(targetDir)
			if err != nil {
				t.Fatalf("os.ReadDir(%q) failed: %v", targetDir, err)
			}
			if len(entries) != 0 {
				t.Fatalf("expected target directory %q to be empty, got %d entries", targetDir, len(entries))
			}
		})
	}
}

func TestEnsureEmptyDirectoryRecreatesDanglingSymlinkTarget(t *testing.T) {
	base := t.TempDir()
	targetDir := filepath.Join(base, "missing_target")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	linkPath := filepath.Join(base, "BP")
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Fatalf("os.Symlink failed: %v", err)
	}
	if err := os.RemoveAll(targetDir); err != nil {
		t.Fatalf("os.RemoveAll failed: %v", err)
	}

	if err := ensureEmptyDirectory(linkPath); err != nil {
		t.Fatalf("ensureEmptyDirectory(%q) failed: %v", linkPath, err)
	}

	linfo, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("os.Lstat(%q) failed: %v", linkPath, err)
	}
	if linfo.Mode()&(fs.ModeSymlink|fs.ModeIrregular) == 0 {
		t.Fatalf("expected %q to remain a symlink, got mode %v", linkPath, linfo.Mode())
	}

	info, err := os.Stat(targetDir)
	if err != nil || !info.IsDir() {
		t.Fatalf("expected recreated target %q to be a directory, got err=%v", targetDir, err)
	}
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) failed: %v", targetDir, err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected recreated target %q to be empty, got %d entries", targetDir, len(entries))
	}
}

func TestEnsureEmptyDirectoryReplacesSymlinkToFile(t *testing.T) {
	base := t.TempDir()
	targetFile := filepath.Join(base, "target.txt")
	if err := os.WriteFile(targetFile, []byte("keep me"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	linkPath := filepath.Join(base, "RP")
	if err := os.Symlink(targetFile, linkPath); err != nil {
		t.Fatalf("os.Symlink failed: %v", err)
	}

	if err := ensureEmptyDirectory(linkPath); err != nil {
		t.Fatalf("ensureEmptyDirectory(%q) failed: %v", linkPath, err)
	}

	linfo, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("os.Lstat(%q) failed: %v", linkPath, err)
	}
	if linfo.Mode()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
		t.Fatalf("expected %q to no longer be a symlink, got mode %v", linkPath, linfo.Mode())
	}
	if !linfo.IsDir() {
		t.Fatalf("expected %q to be a real directory", linkPath)
	}
	entries, err := os.ReadDir(linkPath)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) failed: %v", linkPath, err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected %q to be empty, got %d entries", linkPath, len(entries))
	}

	data, err := os.ReadFile(targetFile)
	if err != nil || string(data) != "keep me" {
		t.Fatalf("expected target file %q to stay untouched, got data=%q err=%v", targetFile, string(data), err)
	}
}

func TestSetupTmpFilesMissingAndInvalidSources(t *testing.T) {
	base := t.TempDir()
	dotRegolith := filepath.Join(base, ".regolith")
	tmpRoot := filepath.Join(dotRegolith, "tmp")

	for _, sub := range []string{"RP", "BP", "data"} {
		subDir := filepath.Join(tmpRoot, sub, "nested")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
		if err := os.WriteFile(filepath.Join(subDir, "stale.txt"), []byte("stale"), 0644); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}

	ctx := RunContext{
		AbsoluteLocation: base,
		DotRegolithPath:  dotRegolith,
		Config: &Config{
			Packs: Packs{
				ResourceFolder: filepath.Join(base, "missing_rp"),
				BehaviorFolder: "",
			},
			RegolithProject: RegolithProject{
				DataPath: filepath.Join(base, "missing_data"),
			},
		},
	}

	if err := SetupTmpFiles(ctx); err != nil {
		t.Fatalf("SetupTmpFiles failed for missing/empty sources: %v", err)
	}

	for _, sub := range []string{"RP", "BP", "data"} {
		entries, err := os.ReadDir(filepath.Join(tmpRoot, sub))
		if err != nil {
			t.Fatalf("os.ReadDir(%s) failed: %v", sub, err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected tmp/%s to be empty, got %d entries", sub, len(entries))
		}
	}

	invalidCtx := RunContext{
		AbsoluteLocation: base,
		DotRegolithPath:  dotRegolith,
		Config: &Config{
			Packs: Packs{
				ResourceFolder: filepath.Join(base, "bad\x00path"),
				BehaviorFolder: "",
			},
			RegolithProject: RegolithProject{
				DataPath: "",
			},
		},
	}
	if err := SetupTmpFiles(invalidCtx); err == nil {
		t.Fatal("expected SetupTmpFiles to return an error for invalid stat path")
	}
}
