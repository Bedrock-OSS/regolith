package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

// TestConditionalFilter runs a test that checks whether the 'when' property
// of a filter properly locks/enables the execution of the filter.
func TestConditionalFilter(t *testing.T) {
	// Switch to current working directory at the end of the test
	defer os.Chdir(getWdOrFatal(t))

	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestConditionalFilter", t)

	t.Log("Copying the project files into the testing directory...")
	project := absOrFatal(filepath.Join(conditionalFilterPath, "project"), t)
	copyFilesOrFatal(project, tmpDir, t)

	// Load abs path of the expected result and switch to the working directory
	expectedBuildResult := absOrFatal(
		filepath.Join(conditionalFilterPath, "expected_build_result"), t)
	os.Chdir(tmpDir)

	// THE TEST
	t.Log("Running Regolith with a conditional filter...")
	if err := regolith.Run("default", []string{}, true, "", false, false, false); err != nil {
		t.Fatal("'regolith run' failed:", err.Error())
	}

	// TEST EVALUATION
	t.Log("Evaluating the test results...")
	comparePaths(expectedBuildResult, filepath.Join(tmpDir, "build"), t)
}

func setupNestedInitialProject(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	for _, dir := range []string{
		filepath.Join(tmpDir, "packs", "BP"),
		filepath.Join(tmpDir, "packs", "RP"),
		filepath.Join(tmpDir, "packs", "data"),
		filepath.Join(tmpDir, "filters"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}

	writeMarkerPy := "from pathlib import Path\n\n(Path(\"BP\") / \"initial_marker.txt\").write_text(\"ran\", encoding=\"utf-8\")\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "filters", "write_marker.py"), []byte(writeMarkerPy), 0644); err != nil {
		t.Fatalf("failed to write filter script: %v", err)
	}

	configJSON := `{
	"name": "nested_initial_test",
	"author": "Bedrock-OSS",
	"packs": {
		"behaviorPack": "./packs/BP",
		"resourcePack": "./packs/RP"
	},
	"regolith": {
		"formatVersion": "1.9.0",
		"dataPath": "./packs/data",
		"filterDefinitions": {
			"write_marker": {
				"runWith": "python",
				"script": "filters/write_marker.py"
			}
		},
		"profiles": {
			"outer": {
				"export": {
					"target": "local"
				},
				"filters": [
					{
						"profile": "inner"
					}
				]
			},
			"inner": {
				"export": {
					"target": "local"
				},
				"filters": [
					{
						"filter": "write_marker",
						"when": "initial"
					}
				]
			}
		}
	}
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config.json: %v", err)
	}
	return tmpDir
}

func TestConditionalFilterNestedProfileInitial(t *testing.T) {
	runProfileImplCases := []struct {
		name         string
		initial      bool
		expectMarker bool
	}{
		{name: "RunProfileImpl initial true", initial: true, expectMarker: true},
		{name: "RunProfileImpl initial false", initial: false, expectMarker: false},
	}

	for _, tc := range runProfileImplCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := setupNestedInitialProject(t)

			cfgMap, err := regolith.LoadConfigAsMap()
			if err != nil {
				t.Fatalf("LoadConfigAsMap failed: %v", err)
			}
			cfg, err := regolith.ConfigFromObject(cfgMap)
			if err != nil {
				t.Fatalf("ConfigFromObject failed: %v", err)
			}

			dotRegolithPath := filepath.Join(tmpDir, ".regolith")
			ctx := regolith.RunContext{
				Initial:              tc.initial,
				AbsoluteLocation:     tmpDir,
				Config:               cfg,
				Profile:              "outer",
				DotRegolithPath:      dotRegolithPath,
				Settings:             map[string]any{},
				DisableSizeTimeCheck: true,
			}
			if err := regolith.SetupTmpFiles(ctx); err != nil {
				t.Fatalf("SetupTmpFiles failed: %v", err)
			}
			interrupted, err := regolith.RunProfileImpl(ctx)
			if err != nil {
				t.Fatalf("RunProfileImpl failed: %v", err)
			}
			if interrupted {
				t.Fatal("RunProfileImpl unexpectedly reported interruption")
			}

			absWorkingDir, err := regolith.GetAbsoluteWorkingDirectory(dotRegolithPath)
			if err != nil {
				t.Fatalf("GetAbsoluteWorkingDirectory failed: %v", err)
			}
			markerPath := filepath.Join(absWorkingDir, "BP", "initial_marker.txt")
			content, err := os.ReadFile(markerPath)
			if tc.expectMarker {
				if err != nil {
					t.Fatalf("expected marker file %s to exist: %v", markerPath, err)
				}
				if string(content) != "ran" {
					t.Fatalf("marker content = %q, want %q", string(content), "ran")
				}
			} else if !os.IsNotExist(err) {
				t.Fatalf("expected marker file %s not to exist, got err=%v content=%q", markerPath, err, string(content))
			}
		})
	}

	t.Run("regolith.Run", func(t *testing.T) {
		tmpDir := setupNestedInitialProject(t)
		if err := regolith.Run("outer", []string{}, true, "", false, false, false); err != nil {
			t.Fatalf("'regolith run' failed: %v", err)
		}
		markerPath := filepath.Join(tmpDir, "build", "nested_initial_test_bp", "initial_marker.txt")
		content, err := os.ReadFile(markerPath)
		if err != nil {
			t.Fatalf("expected marker file %s to exist: %v", markerPath, err)
		}
		if string(content) != "ran" {
			t.Fatalf("marker content = %q, want %q", string(content), "ran")
		}
	})
}

func TestProfileFilterRunExtraArguments(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	for _, dir := range []string{
		filepath.Join(tmpDir, "packs", "BP"),
		filepath.Join(tmpDir, "packs", "RP"),
		filepath.Join(tmpDir, "packs", "data"),
		filepath.Join(tmpDir, "filters"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}

	writeArgsPy := "import json\nimport sys\nfrom pathlib import Path\n\n(Path(\"BP\") / \"args.json\").write_text(json.dumps(sys.argv[1:]), encoding=\"utf-8\")\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "filters", "write_args.py"), []byte(writeArgsPy), 0644); err != nil {
		t.Fatalf("failed to write filter script: %v", err)
	}

	configJSON := `{
	"name": "nested_extra_args_test",
	"author": "Bedrock-OSS",
	"packs": {
		"behaviorPack": "./packs/BP",
		"resourcePack": "./packs/RP"
	},
	"regolith": {
		"formatVersion": "1.9.0",
		"dataPath": "./packs/data",
		"filterDefinitions": {
			"write_args": {
				"runWith": "python",
				"script": "filters/write_args.py"
			}
		},
		"profiles": {
			"outer": {
				"export": {
					"target": "local"
				},
				"filters": [
					{
						"profile": "inner"
					}
				]
			},
			"inner": {
				"export": {
					"target": "local"
				},
				"filters": [
					{
						"filter": "write_args",
						"arguments": ["base"],
						"extraArguments": "append"
					}
				]
			}
		}
	}
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config.json: %v", err)
	}

	if err := regolith.Run("outer", []string{"hello"}, true, "", false, false, false); err != nil {
		t.Fatalf("'regolith run' failed: %v", err)
	}

	argsPath := filepath.Join(tmpDir, "build", "nested_extra_args_test_bp", "args.json")
	content, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", argsPath, err)
	}
	var got []string
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("failed to unmarshal %s: %v", argsPath, err)
	}
	want := []string{"base", "hello"}
	if !slices.Equal(got, want) {
		t.Fatalf("args = %v, want %v", got, want)
	}
}
