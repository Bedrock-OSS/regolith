package test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bedrock-OSS/regolith/regolith"
)

const generatePackFilesFilterPy = `from pathlib import Path
import sys

marker = Path("data/marker.txt").read_text(encoding="utf-8").strip()

for pack in ("BP", "RP"):
    pack_dir = Path(pack)
    existing = [p.name for p in pack_dir.iterdir()]
    if existing:
        sys.stderr.write(f"Expected {pack} to be empty, found: {existing}\n")
        sys.exit(1)

    (pack_dir / f"{marker}.txt").write_text(f"generated {marker}\n", encoding="utf-8")
    nested_dir = pack_dir / f"nested_{marker}"
    nested_dir.mkdir(parents=True, exist_ok=True)
    (nested_dir / "item.txt").write_text(f"nested {marker}\n", encoding="utf-8")
`

const checkEmptyDataFilterPy = `from pathlib import Path
import sys

data_dir = Path("data")
existing = [p.name for p in data_dir.iterdir()]
if existing:
    sys.stderr.write(f"Expected data to be empty, found: {existing}\n")
    sys.exit(1)

(data_dir / "stale.txt").write_text("stale\n", encoding="utf-8")
`

func writeFileOrFatal(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create parent dirs for %q: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %q: %v", path, err)
	}
}

func setupTmpCleanupProject(t *testing.T, packsJSON, dataPath, filterScript string) string {
	t.Helper()
	projectDir := t.TempDir()

	config := fmt.Sprintf(`{
	"$schema": "https://raw.githubusercontent.com/Bedrock-OSS/regolith-schemas/main/config/v1.9.json",
	"author": "Bedrock-OSS",
	"name": "tmp_cleanup_test",
	"packs": %s,
	"regolith": {
		"formatVersion": "1.9.0",
		"dataPath": %q,
		"filterDefinitions": {
			"test_filter": {
				"runWith": "python",
				"script": "filters/run.py"
			}
		},
		"profiles": {
			"default": {
				"export": {
					"target": "local",
					"readOnly": false,
					"bpName": "'BP'",
					"rpName": "'RP'"
				},
				"filters": [
					{
						"filter": "test_filter"
					}
				]
			}
		}
	}
}
`, packsJSON, dataPath)

	writeFileOrFatal(t, filepath.Join(projectDir, "config.json"), config)
	writeFileOrFatal(t, filepath.Join(projectDir, "filters", "run.py"), filterScript)
	return projectDir
}

func updateMarker(t *testing.T, markerPath, content string) {
	t.Helper()
	if err := os.WriteFile(markerPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to update marker %q: %v", markerPath, err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(markerPath, future, future); err != nil {
		t.Fatalf("failed to update mtime for %q: %v", markerPath, err)
	}
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %q to exist, got: %v", path, err)
	}
}

func assertPathNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %q not to exist, got err=%v", path, err)
	}
}

func assertMarkerOutputs(t *testing.T, projectDir, marker string, absentMarkers ...string) {
	t.Helper()
	roots := []string{
		filepath.Join(projectDir, ".regolith", "tmp", "BP"),
		filepath.Join(projectDir, ".regolith", "tmp", "RP"),
		filepath.Join(projectDir, "build", "BP"),
		filepath.Join(projectDir, "build", "RP"),
	}
	for _, root := range roots {
		assertPathExists(t, filepath.Join(root, marker+".txt"))
		assertPathExists(t, filepath.Join(root, "nested_"+marker, "item.txt"))
		for _, old := range absentMarkers {
			assertPathNotExists(t, filepath.Join(root, old+".txt"))
			assertPathNotExists(t, filepath.Join(root, "nested_"+old))
		}
	}
}

func assertIsSymlinkOrJunction(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("os.Lstat(%q) failed: %v", path, err)
	}
	if info.Mode()&(fs.ModeSymlink|fs.ModeIrregular) == 0 {
		t.Fatalf("expected %q to remain a symlink or junction, got mode %v", path, info.Mode())
	}
}

func TestTmpCleanupMissingPackDirs(t *testing.T) {
	origWd := getWdOrFatal(t)
	defer os.Chdir(origWd)

	projectDir := setupTmpCleanupProject(
		t,
		`{"behaviorPack": "./packs/BP", "resourcePack": "./packs/RP"}`,
		"./data",
		generatePackFilesFilterPy,
	)
	markerPath := filepath.Join(projectDir, "data", "marker.txt")
	writeFileOrFatal(t, markerPath, "run1")

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := regolith.Run("default", []string{}, true, "", false, false, false); err != nil {
		t.Fatalf("run 1 failed: %v", err)
	}
	assertMarkerOutputs(t, projectDir, "run1")

	updateMarker(t, markerPath, "run2\n")
	if err := regolith.Run("default", []string{}, true, "", false, false, false); err != nil {
		t.Fatalf("run 2 failed: %v", err)
	}
	assertMarkerOutputs(t, projectDir, "run2", "run1")
}

func TestTmpCleanupOmittedPackPaths(t *testing.T) {
	origWd := getWdOrFatal(t)
	defer os.Chdir(origWd)

	projectDir := setupTmpCleanupProject(
		t,
		`{}`,
		"./data",
		generatePackFilesFilterPy,
	)
	markerPath := filepath.Join(projectDir, "data", "marker.txt")
	writeFileOrFatal(t, markerPath, "run1")

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := regolith.Run("default", []string{}, true, "", false, false, false); err != nil {
		t.Fatalf("run 1 failed: %v", err)
	}
	assertMarkerOutputs(t, projectDir, "run1")

	updateMarker(t, markerPath, "run2\n")
	if err := regolith.Run("default", []string{}, true, "", false, false, false); err != nil {
		t.Fatalf("run 2 failed: %v", err)
	}
	assertMarkerOutputs(t, projectDir, "run2", "run1")
}

func TestTmpCleanupSymlinkExport(t *testing.T) {
	cases := []struct {
		name      string
		packsJSON string
	}{
		{
			name:      "missing pack dirs",
			packsJSON: `{"behaviorPack": "./packs/BP", "resourcePack": "./packs/RP"}`,
		},
		{
			name:      "omitted pack paths",
			packsJSON: `{}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			origWd := getWdOrFatal(t)
			defer os.Chdir(origWd)

			projectDir := setupTmpCleanupProject(
				t,
				tc.packsJSON,
				"./data",
				generatePackFilesFilterPy,
			)
			markerPath := filepath.Join(projectDir, "data", "marker.txt")
			writeFileOrFatal(t, markerPath, "run1")

			if err := os.Chdir(projectDir); err != nil {
				t.Fatalf("chdir failed: %v", err)
			}

			if err := regolith.Run("default", []string{}, true, "", false, true, false); err != nil {
				t.Fatalf("run 1 failed: %v", err)
			}
			assertIsSymlinkOrJunction(t, filepath.Join(projectDir, ".regolith", "tmp", "BP"))
			assertIsSymlinkOrJunction(t, filepath.Join(projectDir, ".regolith", "tmp", "RP"))
			assertMarkerOutputs(t, projectDir, "run1")

			updateMarker(t, markerPath, "run2\n")
			if err := regolith.Run("default", []string{}, true, "", false, true, false); err != nil {
				t.Fatalf("run 2 failed: %v", err)
			}
			assertIsSymlinkOrJunction(t, filepath.Join(projectDir, ".regolith", "tmp", "BP"))
			assertIsSymlinkOrJunction(t, filepath.Join(projectDir, ".regolith", "tmp", "RP"))
			assertMarkerOutputs(t, projectDir, "run2", "run1")
		})
	}
}

func TestTmpCleanupMissingDataDir(t *testing.T) {
	origWd := getWdOrFatal(t)
	defer os.Chdir(origWd)

	projectDir := setupTmpCleanupProject(
		t,
		`{}`,
		"./missing_data",
		checkEmptyDataFilterPy,
	)

	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	if err := regolith.Run("default", []string{}, true, "", false, false, false); err != nil {
		t.Fatalf("run 1 failed: %v", err)
	}
	assertPathExists(t, filepath.Join(projectDir, ".regolith", "tmp", "data", "stale.txt"))

	if err := regolith.Run("default", []string{}, true, "", false, false, false); err != nil {
		t.Fatalf("run 2 failed (tmp/data was not cleaned): %v", err)
	}
	assertPathExists(t, filepath.Join(projectDir, ".regolith", "tmp", "data", "stale.txt"))
}
