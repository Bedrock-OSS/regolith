package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

// TestApplyFilter tests the 'regolith apply-filter' command
func TestApplyFilter(t *testing.T) {
	// Switch to current working directory at the end of the test
	defer os.Chdir(getWdOrFatal(t))

	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestApplyFilter", t)

	t.Log("Copying the project files into the testing directory...")
	project := absOrFatal(filepath.Join(applyFilterPath, "project"), t)
	copyFilesOrFatal(project, tmpDir, t)

	// Load abs path of the expected result and switch to the working directory
	expectedResult := absOrFatal(
		filepath.Join(applyFilterPath, "filtered_project"), t)
	os.Chdir(tmpDir)

	// THE TEST
	t.Log("Running 'rego apply-filter'...")
	if err := regolith.ApplyFilter("test_filter", []string{"Regolith"}, true, ""); err != nil {
		t.Fatal("'regolith apply-filter' failed:", err.Error())
	}
	// TEST EVALUATION
	t.Log("Evaluating the test results...")
	comparePaths(expectedResult, tmpDir, t)
}

func TestApplyFilterRemoteSubfilterInitial(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	for _, dir := range []string{
		filepath.Join(tmpDir, "packs", "BP"),
		filepath.Join(tmpDir, "packs", "RP"),
		filepath.Join(tmpDir, "packs", "data"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}

	configJSON := `{
	"name": "apply_filter_remote_initial_test",
	"author": "Bedrock-OSS",
	"packs": {
		"behaviorPack": "./packs/BP",
		"resourcePack": "./packs/RP"
	},
	"regolith": {
		"formatVersion": "1.9.0",
		"dataPath": "./packs/data",
		"filterDefinitions": {
			"remote_initial": {
				"url": "github.com/example/test",
				"version": "1.0.0"
			}
		},
		"profiles": {
			"default": {
				"export": {
					"target": "local"
				},
				"filters": []
			}
		}
	}
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config.json: %v", err)
	}

	dotRegolithPath, err := regolith.GetDotRegolith(".")
	if err != nil {
		t.Fatalf("GetDotRegolith failed: %v", err)
	}
	if dotRegolithPath != ".regolith" {
		t.Cleanup(func() { _ = os.RemoveAll(dotRegolithPath) })
	}

	filterDir := filepath.Join(dotRegolithPath, "cache", "filters", "remote_initial")
	if err := os.MkdirAll(filterDir, 0755); err != nil {
		t.Fatalf("failed to create filter dir: %v", err)
	}
	filterJSON := `{
	"version": "1.0.0",
	"filters": [
		{
			"runWith": "python",
			"script": "main.py",
			"when": "initial"
		}
	]
}`
	if err := os.WriteFile(filepath.Join(filterDir, "filter.json"), []byte(filterJSON), 0644); err != nil {
		t.Fatalf("failed to write filter.json: %v", err)
	}
	mainPy := "from pathlib import Path\n\n(Path(\"BP\") / \"applied_initial.txt\").write_text(\"applied\", encoding=\"utf-8\")\n"
	if err := os.WriteFile(filepath.Join(filterDir, "main.py"), []byte(mainPy), 0644); err != nil {
		t.Fatalf("failed to write main.py: %v", err)
	}

	if err := regolith.ApplyFilter("remote_initial", []string{}, true, ""); err != nil {
		t.Fatalf("'regolith apply-filter' failed: %v", err)
	}

	markerPath := filepath.Join(tmpDir, "packs", "BP", "applied_initial.txt")
	content, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("expected marker file %s to exist: %v", markerPath, err)
	}
	if string(content) != "applied" {
		t.Fatalf("marker content = %q, want %q", string(content), "applied")
	}
}
