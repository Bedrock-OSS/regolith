package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bedrock-OSS/regolith/regolith"
)

func TestExportTargetsFromObject_SingleObject(t *testing.T) {
	input := map[string]any{
		"target":   "development",
		"readOnly": false,
	}
	targets, err := regolith.ExportTargetsFromObject(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("Expected 1 target, got %d", len(targets))
	}
	if targets[0].Target != "development" {
		t.Fatalf("Expected target \"development\", got %q", targets[0].Target)
	}
}

func TestExportTargetsFromObject_Array(t *testing.T) {
	input := []any{
		map[string]any{"target": "development", "build": "standard"},
		map[string]any{"target": "local"},
	}
	targets, err := regolith.ExportTargetsFromObject(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 {
		t.Fatalf("Expected 2 targets, got %d", len(targets))
	}
	if targets[0].Target != "development" {
		t.Fatalf("Expected first target \"development\", got %q", targets[0].Target)
	}
	if targets[0].Build != "standard" {
		t.Fatalf("Expected first target build \"standard\", got %q", targets[0].Build)
	}
	if targets[1].Target != "local" {
		t.Fatalf("Expected second target \"local\", got %q", targets[1].Target)
	}
}

func TestExportTargetsFromObject_EmptyArray(t *testing.T) {
	input := []any{}
	_, err := regolith.ExportTargetsFromObject(input)
	if err == nil {
		t.Fatal("Expected error for empty array, got nil")
	}
}

func TestExportTargetsFromObject_InvalidType(t *testing.T) {
	_, err := regolith.ExportTargetsFromObject("invalid")
	if err == nil {
		t.Fatal("Expected error for invalid type, got nil")
	}
}

func TestExportTargets_MarshalJSON_Single(t *testing.T) {
	targets := regolith.ExportTargets{
		{Target: "development", ReadOnly: false},
	}
	data, err := json.Marshal(targets)
	if err != nil {
		t.Fatal(err)
	}
	// Single target should marshal as an object, not an array
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("Single target should marshal as object, got: %s", string(data))
	}
	if obj["target"] != "development" {
		t.Fatalf("Expected target \"development\", got %v", obj["target"])
	}
}

func TestExportTargets_MarshalJSON_Multiple(t *testing.T) {
	targets := regolith.ExportTargets{
		{Target: "development"},
		{Target: "local"},
	}
	data, err := json.Marshal(targets)
	if err != nil {
		t.Fatal(err)
	}
	// Multiple targets should marshal as an array
	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("Multiple targets should marshal as array, got: %s", string(data))
	}
	if len(arr) != 2 {
		t.Fatalf("Expected 2 elements, got %d", len(arr))
	}
}

func TestExportTargets_UnmarshalJSON_Single(t *testing.T) {
	data := []byte(`{"target": "development", "readOnly": true}`)
	var targets regolith.ExportTargets
	if err := json.Unmarshal(data, &targets); err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("Expected 1 target, got %d", len(targets))
	}
	if targets[0].Target != "development" {
		t.Fatalf("Expected \"development\", got %q", targets[0].Target)
	}
	if !targets[0].ReadOnly {
		t.Fatal("Expected ReadOnly to be true")
	}
}

func TestExportTargets_UnmarshalJSON_Array(t *testing.T) {
	data := []byte(`[{"target": "development"}, {"target": "exact", "bpPath": "./build/BP", "rpPath": "./build/RP"}]`)
	var targets regolith.ExportTargets
	if err := json.Unmarshal(data, &targets); err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 {
		t.Fatalf("Expected 2 targets, got %d", len(targets))
	}
	if targets[1].BpPath != "./build/BP" {
		t.Fatalf("Expected bpPath \"./build/BP\", got %q", targets[1].BpPath)
	}
}

func TestExportTargets_UnmarshalJSON_InvalidObject(t *testing.T) {
	data := []byte(`{"readOnly": true}`)
	var targets regolith.ExportTargets
	if err := json.Unmarshal(data, &targets); err == nil {
		t.Fatal("Expected error for export target without target property")
	}
}

func TestExportTargets_UnmarshalJSON_EmptyArray(t *testing.T) {
	data := []byte(`[]`)
	var targets regolith.ExportTargets
	if err := json.Unmarshal(data, &targets); err == nil {
		t.Fatal("Expected error for empty export target array")
	}
}

func TestExportTargets_RoundTrip(t *testing.T) {
	original := regolith.ExportTargets{
		{Target: "development", Build: "standard", ReadOnly: true},
		{Target: "exact", BpPath: "./bp", RpPath: "./rp"},
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded regolith.ExportTargets
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != len(original) {
		t.Fatalf("Expected %d targets, got %d", len(original), len(decoded))
	}
	for i := range original {
		if decoded[i].Target != original[i].Target {
			t.Fatalf("Target %d: expected %q, got %q", i, original[i].Target, decoded[i].Target)
		}
	}
}

func TestProfileFromObject_SingleExportSetsExportTarget(t *testing.T) {
	profile, err := regolith.ProfileFromObject(
		map[string]any{
			"filters": []any{},
			"export": map[string]any{
				"target": "local",
			},
		},
		map[string]regolith.FilterInstaller{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.ExportTarget) != 1 {
		t.Fatalf("Expected 1 target, got %d", len(profile.ExportTarget))
	}
	if profile.ExportTarget[0].Target != "local" {
		t.Fatalf("Expected ExportTarget[0] to be local, got %q", profile.ExportTarget[0].Target)
	}
}

func TestProfileMarshalJSON_SingleExportTarget(t *testing.T) {
	profile := regolith.Profile{
		FilterCollection: regolith.FilterCollection{
			Filters: []regolith.FilterRunner{},
		},
		ExportTarget: regolith.ExportTargets{
			{Target: "local"},
		},
	}
	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatal(err)
	}
	exportObj, ok := obj["export"].(map[string]any)
	if !ok {
		t.Fatalf("Expected export to marshal as an object, got: %s", data)
	}
	if exportObj["target"] != "local" {
		t.Fatalf("Expected target local, got %v", exportObj["target"])
	}
}

func TestRunWithMultipleExactExportTargets(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))

	tmpDir := prepareTestDirectory(
		fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()), t)
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(minimalProjectPath, workingDir, t)

	config := []byte(`{
		"$schema": "https://raw.githubusercontent.com/Bedrock-OSS/regolith-schemas/main/config/v1.2.json",
		"name": "regolith_test_project",
		"author": "Bedrock-OSS",
		"packs": {
			"behaviorPack": "./packs/BP",
			"resourcePack": "./packs/RP"
		},
		"regolith": {
			"profiles": {
				"multi": {
					"filters": [],
					"export": [
						{
							"target": "exact",
							"rpPath": "../target-a/RP",
							"bpPath": "../target-a/BP"
						},
						{
							"target": "exact",
							"rpPath": "../target-b/RP",
							"bpPath": "../target-b/BP"
						}
					]
				}
			},
			"dataPath": "./packs/data"
		}
	}`)
	if err := os.WriteFile(filepath.Join(workingDir, "config.json"), config, 0644); err != nil {
		t.Fatal("Unable to write multi-target config:", err)
	}

	os.Chdir(workingDir)
	if err := regolith.Run("multi", []string{}, true, "", false, false, false); err != nil {
		t.Fatal("First multi-target run failed:", err)
	}

	for _, target := range []string{"target-a", "target-b"} {
		comparePaths(
			filepath.Join(workingDir, "packs", "BP"),
			filepath.Join(tmpDir, target, "BP"),
			t,
		)
		comparePaths(
			filepath.Join(workingDir, "packs", "RP"),
			filepath.Join(tmpDir, target, "RP"),
			t,
		)
	}

	if err := regolith.Run("multi", []string{}, true, "", false, false, false); err != nil {
		t.Fatal("Second multi-target run failed safety checks:", err)
	}

	unexpectedFile := filepath.Join(tmpDir, "target-a", "BP", "unexpected.txt")
	if err := os.WriteFile(unexpectedFile, []byte("not created by regolith"), 0644); err != nil {
		t.Fatal("Unable to create unexpected target file:", err)
	}
	if err := regolith.Run("multi", []string{}, true, "", false, false, false); err == nil {
		t.Fatal("Expected file protection to reject unexpected file in first target")
	}
}

func TestRunWithOverlappingExportTargetsFails(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))

	tmpDir := prepareTestDirectory(
		fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()), t)
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(minimalProjectPath, workingDir, t)

	config := []byte(`{
		"$schema": "https://raw.githubusercontent.com/Bedrock-OSS/regolith-schemas/main/config/v1.2.json",
		"name": "regolith_test_project",
		"author": "Bedrock-OSS",
		"packs": {
			"behaviorPack": "./packs/BP",
			"resourcePack": "./packs/RP"
		},
		"regolith": {
			"profiles": {
				"multi": {
					"filters": [],
					"export": [
						{
							"target": "exact",
							"rpPath": "../target-a/RP",
							"bpPath": "../target-a/BP"
						},
						{
							"target": "exact",
							"rpPath": "../target-b/RP",
							"bpPath": "../target-a/BP/nested"
						}
					]
				}
			},
			"dataPath": "./packs/data"
		}
	}`)
	if err := os.WriteFile(filepath.Join(workingDir, "config.json"), config, 0644); err != nil {
		t.Fatal("Unable to write overlapping-target config:", err)
	}

	os.Chdir(workingDir)
	err := regolith.Run("multi", []string{}, true, "", false, false, false)
	if err == nil {
		t.Fatal("Expected overlapping export paths to fail")
	}
	t.Logf("Got expected export path collision error:\n%v", err)
	if !strings.Contains(err.Error(), "Export path collision detected") {
		t.Fatalf("Expected export path collision error, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "target-a")); !os.IsNotExist(err) {
		t.Fatal("Expected export to fail before creating target-a")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "target-b")); !os.IsNotExist(err) {
		t.Fatal("Expected export to fail before creating target-b")
	}
}

func TestRunWithMultipleTargetsIgnoresSymlinkExport(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))

	tmpDir := prepareTestDirectory(
		fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()), t)
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(minimalProjectPath, workingDir, t)

	config := []byte(`{
		"$schema": "https://raw.githubusercontent.com/Bedrock-OSS/regolith-schemas/main/config/v1.2.json",
		"name": "regolith_test_project",
		"author": "Bedrock-OSS",
		"packs": {
			"behaviorPack": "./packs/BP",
			"resourcePack": "./packs/RP"
		},
		"regolith": {
			"profiles": {
				"multi": {
					"filters": [],
					"export": [
						{
							"target": "exact",
							"rpPath": "../target-a/RP",
							"bpPath": "../target-a/BP"
						},
						{
							"target": "exact",
							"rpPath": "../target-b/RP",
							"bpPath": "../target-b/BP"
						}
					]
				}
			},
			"dataPath": "./packs/data"
		}
	}`)
	if err := os.WriteFile(filepath.Join(workingDir, "config.json"), config, 0644); err != nil {
		t.Fatal("Unable to write multi-target config:", err)
	}

	os.Chdir(workingDir)
	if err := regolith.Run("multi", []string{}, true, "", false, true, false); err != nil {
		t.Fatal("Multi-target run with --symlink-export enabled failed:", err)
	}

	for _, tmpPack := range []string{"BP", "RP"} {
		info, err := os.Lstat(filepath.Join(workingDir, ".regolith", "tmp", tmpPack))
		if err != nil {
			t.Fatalf("Unable to stat tmp %s path: %v", tmpPack, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("Expected tmp %s path to be a directory, got symlink", tmpPack)
		}
	}

	for _, target := range []string{"target-a", "target-b"} {
		comparePaths(
			filepath.Join(workingDir, "packs", "BP"),
			filepath.Join(tmpDir, target, "BP"),
			t,
		)
		comparePaths(
			filepath.Join(workingDir, "packs", "RP"),
			filepath.Join(tmpDir, target, "RP"),
			t,
		)
	}
}

func TestRunWithLocalAndDevelopmentExportTargets(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))

	tmpDir := prepareTestDirectory(
		fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()), t)
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(minimalProjectPath, workingDir, t)

	mojangDir := filepath.Join(tmpDir, "com.mojang")
	if err := os.MkdirAll(mojangDir, 0755); err != nil {
		t.Fatal("Unable to create fake com.mojang directory:", err)
	}
	t.Setenv("COM_MOJANG_PACKS", mojangDir)

	config := []byte(`{
		"$schema": "https://raw.githubusercontent.com/Bedrock-OSS/regolith-schemas/main/config/v1.4.json",
		"name": "regolith_test_project",
		"author": "Bedrock-OSS",
		"packs": {
			"behaviorPack": "./packs/BP",
			"resourcePack": "./packs/RP"
		},
		"regolith": {
			"formatVersion": "1.4.0",
			"profiles": {
				"local_and_development": {
					"filters": [],
					"export": [
						{
							"target": "local"
						},
						{
							"target": "development",
							"build": "standard"
						}
					]
				}
			},
			"dataPath": "./packs/data"
		}
	}`)
	if err := os.WriteFile(filepath.Join(workingDir, "config.json"), config, 0644); err != nil {
		t.Fatal("Unable to write mixed-target config:", err)
	}

	os.Chdir(workingDir)
	if err := regolith.Run("local_and_development", []string{}, false, "", false, false, false); err != nil {
		t.Fatal("First mixed-target run failed:", err)
	}

	expectedBp := filepath.Join(workingDir, "packs", "BP")
	expectedRp := filepath.Join(workingDir, "packs", "RP")
	bpName := "regolith_test_project_bp"
	rpName := "regolith_test_project_rp"

	comparePaths(expectedBp, filepath.Join(workingDir, "build", bpName), t)
	comparePaths(expectedRp, filepath.Join(workingDir, "build", rpName), t)
	comparePaths(
		expectedBp,
		filepath.Join(mojangDir, "development_behavior_packs", bpName),
		t,
	)
	comparePaths(
		expectedRp,
		filepath.Join(mojangDir, "development_resource_packs", rpName),
		t,
	)

	if err := regolith.Run("local_and_development", []string{}, false, "", false, false, false); err != nil {
		t.Fatal("Second mixed-target run failed safety checks:", err)
	}
}

func assertSentinelUnchanged(path string, expected []byte, t *testing.T) {
	t.Helper()
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Expected sentinel file %q to remain readable: %v", path, err)
	}
	if string(actual) != string(expected) {
		t.Fatalf("Expected sentinel file %q contents %q, got %q", path, string(expected), string(actual))
	}
}

func assertEditedFilesNoEmptyKeys(dotRegolithPath string, expectedRpCount, expectedBpCount int, t *testing.T) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dotRegolithPath, regolith.EditedFilesPath))
	if err != nil {
		t.Fatalf("Unable to read edited_files.json: %v", err)
	}
	var ef regolith.EditedFiles
	if err := json.Unmarshal(data, &ef); err != nil {
		t.Fatalf("Unable to parse edited_files.json: %v", err)
	}
	if _, hasEmpty := ef.Rp[""]; hasEmpty {
		t.Fatalf("edited_files.json unexpectedly contains an empty key in rp: %s", string(data))
	}
	if _, hasEmpty := ef.Bp[""]; hasEmpty {
		t.Fatalf("edited_files.json unexpectedly contains an empty key in bp: %s", string(data))
	}
	if len(ef.Rp) != expectedRpCount {
		t.Fatalf("Expected %d rp entries in edited_files.json, got %d (%v)", expectedRpCount, len(ef.Rp), ef.Rp)
	}
	if len(ef.Bp) != expectedBpCount {
		t.Fatalf("Expected %d bp entries in edited_files.json, got %d (%v)", expectedBpCount, len(ef.Bp), ef.Bp)
	}
}

func TestGetExactExportPathsRejectsEmptyAndUnsetEnvVar(t *testing.T) {
	const unsetEnvVar = "REGOLITH_TEST_UNSET_EXACT_PATH"
	t.Setenv(unsetEnvVar, "")
	if err := os.Unsetenv(unsetEnvVar); err != nil {
		t.Fatalf("Unable to unset %s: %v", unsetEnvVar, err)
	}

	tests := []struct {
		name        string
		target      regolith.ExportTarget
		expectError bool
		wantProp    string
		wantBp      string
		wantRp      string
	}{
		{
			name:        "empty bpPath",
			target:      regolith.ExportTarget{Target: "exact", BpPath: "", RpPath: "./rp"},
			expectError: true,
			wantProp:    "bpPath",
		},
		{
			name:        "empty rpPath",
			target:      regolith.ExportTarget{Target: "exact", BpPath: "./bp", RpPath: ""},
			expectError: true,
			wantProp:    "rpPath",
		},
		{
			name:        "whitespace bpPath",
			target:      regolith.ExportTarget{Target: "exact", BpPath: "   ", RpPath: "./rp"},
			expectError: true,
			wantProp:    "bpPath",
		},
		{
			name:        "whitespace rpPath",
			target:      regolith.ExportTarget{Target: "exact", BpPath: "./bp", RpPath: "\t"},
			expectError: true,
			wantProp:    "rpPath",
		},
		{
			name:        "unset env var bpPath",
			target:      regolith.ExportTarget{Target: "exact", BpPath: "$" + unsetEnvVar, RpPath: "./rp"},
			expectError: true,
			wantProp:    "bpPath",
		},
		{
			name:        "unset env var rpPath",
			target:      regolith.ExportTarget{Target: "exact", BpPath: "./bp", RpPath: "${" + unsetEnvVar + "}"},
			expectError: true,
			wantProp:    "rpPath",
		},
		{
			name:        "valid paths",
			target:      regolith.ExportTarget{Target: "exact", BpPath: "./bp", RpPath: "./rp"},
			expectError: false,
			wantBp:      filepath.Clean("./bp"),
			wantRp:      filepath.Clean("./rp"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bp, rp, err := regolith.GetExactExportPaths(tt.target)
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error, got nil (bp=%q, rp=%q)", bp, rp)
				}
				if !strings.Contains(err.Error(), tt.wantProp) {
					t.Fatalf("Expected error mentioning %q, got: %v", tt.wantProp, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if bp != tt.wantBp || rp != tt.wantRp {
				t.Fatalf("Expected (%q, %q), got (%q, %q)", tt.wantBp, tt.wantRp, bp, rp)
			}
		})
	}
}

func TestRunWithExactTargetMissingBothPaths(t *testing.T) {
	cases := []struct {
		name          string
		exportJSON    string
		unsafeMode    bool
		symlinkExport bool
	}{
		{
			name:       "omitted safe",
			exportJSON: `{"target": "exact"}`,
		},
		{
			name:       "omitted unsafe",
			exportJSON: `{"target": "exact"}`,
			unsafeMode: true,
		},
		{
			name:          "omitted unsafe symlink",
			exportJSON:    `{"target": "exact"}`,
			unsafeMode:    true,
			symlinkExport: true,
		},
		{
			name:       "empty safe",
			exportJSON: `{"target": "exact", "bpPath": "", "rpPath": ""}`,
		},
		{
			name:       "empty unsafe",
			exportJSON: `{"target": "exact", "bpPath": "", "rpPath": ""}`,
			unsafeMode: true,
		},
		{
			name:          "empty unsafe symlink",
			exportJSON:    `{"target": "exact", "bpPath": "", "rpPath": ""}`,
			unsafeMode:    true,
			symlinkExport: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer os.Chdir(getWdOrFatal(t))

			tmpDir := prepareTestDirectory(
				fmt.Sprintf("%s-%d", strings.ReplaceAll(t.Name(), "/", "-"), time.Now().UnixNano()), t)
			workingDir := filepath.Join(tmpDir, "working-dir")
			copyFilesOrFatal(minimalProjectPath, workingDir, t)

			sentinelPath := filepath.Join(workingDir, "sentinel.txt")
			sentinelContent := []byte("keep project root intact")
			if err := os.WriteFile(sentinelPath, sentinelContent, 0644); err != nil {
				t.Fatal("Unable to write sentinel file:", err)
			}

			config := fmt.Sprintf(`{
				"$schema": "https://raw.githubusercontent.com/Bedrock-OSS/regolith-schemas/main/config/v1.4.json",
				"name": "regolith_test_project",
				"author": "Bedrock-OSS",
				"packs": {
					"behaviorPack": "./packs/BP",
					"resourcePack": "./packs/RP"
				},
				"regolith": {
					"formatVersion": "1.4.0",
					"profiles": {
						"dev": {
							"filters": [],
							"export": %s
						}
					},
					"dataPath": "./packs/data"
				}
			}`, tc.exportJSON)
			if err := os.WriteFile(filepath.Join(workingDir, "config.json"), []byte(config), 0644); err != nil {
				t.Fatal("Unable to write config:", err)
			}

			os.Chdir(workingDir)
			err := regolith.Run("dev", []string{}, true, "", tc.unsafeMode, tc.symlinkExport, false)
			if err == nil {
				t.Fatal("Expected error for exact target missing both bpPath and rpPath, got nil")
			}
			msg := err.Error()
			if !strings.Contains(msg, "bpPath") || !strings.Contains(msg, "rpPath") {
				t.Fatalf("Expected error referencing bpPath and rpPath, got: %v", err)
			}
			if strings.Contains(msg, "Safety mechanism") {
				t.Fatalf("Expected path validation error instead of file protection error, got: %v", err)
			}

			assertSentinelUnchanged(sentinelPath, sentinelContent, t)
			if _, err := os.Stat(filepath.Join(workingDir, "config.json")); err != nil {
				t.Fatalf("Expected config.json to remain intact: %v", err)
			}
			if _, err := os.Stat(filepath.Join(workingDir, "packs", "BP", "manifest.json")); err != nil {
				t.Fatalf("Expected BP manifest.json to remain intact: %v", err)
			}
			if _, err := os.Stat(filepath.Join(workingDir, "packs", "RP", "manifest.json")); err != nil {
				t.Fatalf("Expected RP manifest.json to remain intact: %v", err)
			}
		})
	}
}

func TestRunWithExactTargetMissingSinglePackPath(t *testing.T) {
	cases := []struct {
		name         string
		exportJSON   string
		unsafeMode   bool
		wantProp     string
		untouchedDir string
	}{
		{
			name:         "missing rpPath safe",
			exportJSON:   `{"target": "exact", "bpPath": "../target-bp/BP"}`,
			wantProp:     "rpPath",
			untouchedDir: "target-bp",
		},
		{
			name:         "empty rpPath unsafe",
			exportJSON:   `{"target": "exact", "bpPath": "../target-bp/BP", "rpPath": ""}`,
			unsafeMode:   true,
			wantProp:     "rpPath",
			untouchedDir: "target-bp",
		},
		{
			name:         "missing bpPath safe",
			exportJSON:   `{"target": "exact", "rpPath": "../target-rp/RP"}`,
			wantProp:     "bpPath",
			untouchedDir: "target-rp",
		},
		{
			name:         "empty bpPath unsafe",
			exportJSON:   `{"target": "exact", "bpPath": "", "rpPath": "../target-rp/RP"}`,
			unsafeMode:   true,
			wantProp:     "bpPath",
			untouchedDir: "target-rp",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer os.Chdir(getWdOrFatal(t))

			tmpDir := prepareTestDirectory(
				fmt.Sprintf("%s-%d", strings.ReplaceAll(t.Name(), "/", "-"), time.Now().UnixNano()), t)
			workingDir := filepath.Join(tmpDir, "working-dir")
			copyFilesOrFatal(minimalProjectPath, workingDir, t)

			sentinelPath := filepath.Join(workingDir, "sentinel.txt")
			sentinelContent := []byte("keep project root intact")
			if err := os.WriteFile(sentinelPath, sentinelContent, 0644); err != nil {
				t.Fatal("Unable to write sentinel file:", err)
			}

			config := fmt.Sprintf(`{
				"$schema": "https://raw.githubusercontent.com/Bedrock-OSS/regolith-schemas/main/config/v1.4.json",
				"name": "regolith_test_project",
				"author": "Bedrock-OSS",
				"packs": {
					"behaviorPack": "./packs/BP",
					"resourcePack": "./packs/RP"
				},
				"regolith": {
					"formatVersion": "1.4.0",
					"profiles": {
						"dev": {
							"filters": [],
							"export": %s
						}
					},
					"dataPath": "./packs/data"
				}
			}`, tc.exportJSON)
			if err := os.WriteFile(filepath.Join(workingDir, "config.json"), []byte(config), 0644); err != nil {
				t.Fatal("Unable to write config:", err)
			}

			os.Chdir(workingDir)
			err := regolith.Run("dev", []string{}, true, "", tc.unsafeMode, false, false)
			if err == nil {
				t.Fatalf("Expected error referencing %q, got nil", tc.wantProp)
			}
			msg := err.Error()
			if !strings.Contains(msg, tc.wantProp) {
				t.Fatalf("Expected error referencing %q, got: %v", tc.wantProp, err)
			}
			if strings.Contains(msg, "Safety mechanism") {
				t.Fatalf("Expected path validation error instead of file protection error, got: %v", err)
			}
			if _, statErr := os.Stat(filepath.Join(tmpDir, tc.untouchedDir)); !os.IsNotExist(statErr) {
				t.Fatalf("Expected target directory %q not to be created on validation failure", tc.untouchedDir)
			}
			assertSentinelUnchanged(sentinelPath, sentinelContent, t)
		})
	}
}

func TestRunWithExactTargetUnsetEnvVarPath(t *testing.T) {
	const unsetEnvVar = "REGOLITH_TEST_UNSET_EXACT_PATH"
	t.Setenv(unsetEnvVar, "")
	if err := os.Unsetenv(unsetEnvVar); err != nil {
		t.Fatalf("Unable to unset %s: %v", unsetEnvVar, err)
	}

	cases := []struct {
		name         string
		bpPath       string
		rpPath       string
		wantProp     string
		untouchedDir string
	}{
		{
			name:         "unset env var bpPath",
			bpPath:       "$" + unsetEnvVar,
			rpPath:       "../target-rp/RP",
			wantProp:     "bpPath",
			untouchedDir: "target-rp",
		},
		{
			name:         "unset env var braced rpPath",
			bpPath:       "../target-bp/BP",
			rpPath:       "${" + unsetEnvVar + "}",
			wantProp:     "rpPath",
			untouchedDir: "target-bp",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer os.Chdir(getWdOrFatal(t))

			tmpDir := prepareTestDirectory(
				fmt.Sprintf("%s-%d", strings.ReplaceAll(t.Name(), "/", "-"), time.Now().UnixNano()), t)
			workingDir := filepath.Join(tmpDir, "working-dir")
			copyFilesOrFatal(minimalProjectPath, workingDir, t)

			sentinelPath := filepath.Join(workingDir, "sentinel.txt")
			sentinelContent := []byte("keep project root intact")
			if err := os.WriteFile(sentinelPath, sentinelContent, 0644); err != nil {
				t.Fatal("Unable to write sentinel file:", err)
			}

			config := fmt.Sprintf(`{
				"$schema": "https://raw.githubusercontent.com/Bedrock-OSS/regolith-schemas/main/config/v1.4.json",
				"name": "regolith_test_project",
				"author": "Bedrock-OSS",
				"packs": {
					"behaviorPack": "./packs/BP",
					"resourcePack": "./packs/RP"
				},
				"regolith": {
					"formatVersion": "1.4.0",
					"profiles": {
						"dev": {
							"filters": [],
							"export": {
								"target": "exact",
								"bpPath": %q,
								"rpPath": %q
							}
						}
					},
					"dataPath": "./packs/data"
				}
			}`, tc.bpPath, tc.rpPath)
			if err := os.WriteFile(filepath.Join(workingDir, "config.json"), []byte(config), 0644); err != nil {
				t.Fatal("Unable to write config:", err)
			}

			os.Chdir(workingDir)
			err := regolith.Run("dev", []string{}, true, "", true, false, false)
			if err == nil {
				t.Fatalf("Expected error for unset env var in %q, got nil", tc.wantProp)
			}
			msg := err.Error()
			if !strings.Contains(msg, tc.wantProp) {
				t.Fatalf("Expected error referencing %q, got: %v", tc.wantProp, err)
			}
			if strings.Contains(msg, "Safety mechanism") {
				t.Fatalf("Expected path validation error instead of file protection error, got: %v", err)
			}
			if _, statErr := os.Stat(filepath.Join(tmpDir, tc.untouchedDir)); !os.IsNotExist(statErr) {
				t.Fatalf("Expected target directory %q not to be created on validation failure", tc.untouchedDir)
			}
			assertSentinelUnchanged(sentinelPath, sentinelContent, t)
		})
	}
}

func TestRunWithExactTargetBpOnlyProject(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))

	tmpDir := prepareTestDirectory(
		fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()), t)
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(minimalProjectPath, workingDir, t)

	if err := os.RemoveAll(filepath.Join(workingDir, "packs", "RP")); err != nil {
		t.Fatal("Unable to remove RP folder:", err)
	}
	sentinelPath := filepath.Join(workingDir, "sentinel.txt")
	sentinelContent := []byte("keep project root intact")
	if err := os.WriteFile(sentinelPath, sentinelContent, 0644); err != nil {
		t.Fatal("Unable to write sentinel file:", err)
	}

	config := []byte(`{
		"$schema": "https://raw.githubusercontent.com/Bedrock-OSS/regolith-schemas/main/config/v1.4.json",
		"name": "regolith_test_project",
		"author": "Bedrock-OSS",
		"packs": {
			"behaviorPack": "./packs/BP"
		},
		"regolith": {
			"formatVersion": "1.4.0",
			"profiles": {
				"bp_only": {
					"filters": [],
					"export": {
						"target": "exact",
						"bpPath": "../target-bp/BP"
					}
				}
			},
			"dataPath": "./packs/data"
		}
	}`)
	if err := os.WriteFile(filepath.Join(workingDir, "config.json"), config, 0644); err != nil {
		t.Fatal("Unable to write config:", err)
	}

	os.Chdir(workingDir)
	dotRegolithPath, err := regolith.GetDotRegolith(".")
	if err != nil {
		t.Fatal("Unable to resolve dotRegolithPath:", err)
	}
	expectedBpSrc := filepath.Join(workingDir, "packs", "BP")
	exportedBpDir := filepath.Join(tmpDir, "target-bp", "BP")

	// Run 1 & 2: default flags (size/time check enabled, no symlink)
	for runIdx := 1; runIdx <= 2; runIdx++ {
		if err := regolith.Run("bp_only", []string{}, true, "", false, false, false); err != nil {
			t.Fatalf("Run %d failed: %v", runIdx, err)
		}
		comparePaths(expectedBpSrc, exportedBpDir, t)
		assertEditedFilesNoEmptyKeys(dotRegolithPath, 0, 1, t)
		assertSentinelUnchanged(sentinelPath, sentinelContent, t)
	}

	// Run 3: --disable-size-time-check
	if err := regolith.Run("bp_only", []string{}, true, "", false, false, true); err != nil {
		t.Fatalf("Run with disableSizeTimeCheck failed: %v", err)
	}
	comparePaths(expectedBpSrc, exportedBpDir, t)
	assertEditedFilesNoEmptyKeys(dotRegolithPath, 0, 1, t)
	assertSentinelUnchanged(sentinelPath, sentinelContent, t)

	// Run 4 & 5: --symlink-export
	absTmpPath, err := regolith.GetAbsoluteWorkingDirectory(dotRegolithPath)
	if err != nil {
		t.Fatal("Unable to get tmp path:", err)
	}
	bpTmpPath := filepath.Join(absTmpPath, "BP")
	rpTmpPath := filepath.Join(absTmpPath, "RP")

	for runIdx := 4; runIdx <= 5; runIdx++ {
		if err := regolith.Run("bp_only", []string{}, true, "", false, true, false); err != nil {
			t.Fatalf("Symlink export run %d failed: %v", runIdx, err)
		}
		bpInfo, err := os.Lstat(bpTmpPath)
		if err != nil {
			t.Fatalf("Unable to lstat tmp BP path on run %d: %v", runIdx, err)
		}
		if bpInfo.Mode()&(os.ModeSymlink|os.ModeIrregular) == 0 {
			t.Fatalf("Expected tmp BP path to be a symlink on run %d", runIdx)
		}
		rpInfo, err := os.Lstat(rpTmpPath)
		if err != nil {
			t.Fatalf("Unable to lstat tmp RP path on run %d: %v", runIdx, err)
		}
		if rpInfo.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			t.Fatalf("Expected tmp RP path NOT to be a symlink on run %d", runIdx)
		}
		comparePaths(expectedBpSrc, exportedBpDir, t)
		assertEditedFilesNoEmptyKeys(dotRegolithPath, 0, 1, t)
		assertSentinelUnchanged(sentinelPath, sentinelContent, t)
	}

	// Stale symlink on tmp/RP should be removed without deleting its target contents
	staleRpTarget := filepath.Join(tmpDir, "stale-rp-target")
	if err := os.MkdirAll(staleRpTarget, 0755); err != nil {
		t.Fatal("Unable to create stale RP target dir:", err)
	}
	staleMarker := filepath.Join(staleRpTarget, "marker.txt")
	if err := os.WriteFile(staleMarker, []byte("keep me"), 0644); err != nil {
		t.Fatal("Unable to write stale marker file:", err)
	}
	if err := os.RemoveAll(rpTmpPath); err != nil {
		t.Fatal("Unable to remove tmp RP dir:", err)
	}
	if err := os.Symlink(staleRpTarget, rpTmpPath); err != nil {
		t.Fatal("Unable to create stale tmp RP symlink:", err)
	}

	if err := regolith.Run("bp_only", []string{}, true, "", false, true, false); err != nil {
		t.Fatalf("Symlink export run with stale tmp RP symlink failed: %v", err)
	}
	rpInfo, err := os.Lstat(rpTmpPath)
	if err != nil {
		t.Fatalf("Unable to lstat tmp RP path after stale symlink cleanup: %v", err)
	}
	if rpInfo.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 || !rpInfo.IsDir() {
		t.Fatalf("Expected stale tmp RP symlink to be replaced with a directory")
	}
	assertSentinelUnchanged(staleMarker, []byte("keep me"), t)
	assertSentinelUnchanged(sentinelPath, sentinelContent, t)
}

func TestRunWithExactTargetNoPacksConfigured(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))

	tmpDir := prepareTestDirectory(
		fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()), t)
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(minimalProjectPath, workingDir, t)

	if err := os.RemoveAll(filepath.Join(workingDir, "packs", "BP")); err != nil {
		t.Fatal("Unable to remove BP folder:", err)
	}
	if err := os.RemoveAll(filepath.Join(workingDir, "packs", "RP")); err != nil {
		t.Fatal("Unable to remove RP folder:", err)
	}
	sentinelPath := filepath.Join(workingDir, "sentinel.txt")
	sentinelContent := []byte("keep project root intact")
	if err := os.WriteFile(sentinelPath, sentinelContent, 0644); err != nil {
		t.Fatal("Unable to write sentinel file:", err)
	}

	config := []byte(`{
		"$schema": "https://raw.githubusercontent.com/Bedrock-OSS/regolith-schemas/main/config/v1.4.json",
		"name": "regolith_test_project",
		"author": "Bedrock-OSS",
		"packs": {},
		"regolith": {
			"formatVersion": "1.4.0",
			"profiles": {
				"no_paths": {
					"filters": [],
					"export": {
						"target": "exact"
					}
				},
				"bp_path_only": {
					"filters": [],
					"export": {
						"target": "exact",
						"bpPath": "../target-nopacks/BP"
					}
				}
			},
			"dataPath": "./packs/data"
		}
	}`)
	if err := os.WriteFile(filepath.Join(workingDir, "config.json"), config, 0644); err != nil {
		t.Fatal("Unable to write config:", err)
	}

	os.Chdir(workingDir)
	err := regolith.Run("no_paths", []string{}, true, "", false, false, false)
	if err == nil {
		t.Fatal("Expected error for exact target with neither path and no packs configured, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "\"none\"") || !strings.Contains(msg, "bpPath") || !strings.Contains(msg, "rpPath") {
		t.Fatalf("Expected error recommending \"none\" or specifying bpPath/rpPath, got: %v", err)
	}
	if strings.Contains(msg, "Safety mechanism") {
		t.Fatalf("Expected path validation error instead of file protection error, got: %v", err)
	}
	assertSentinelUnchanged(sentinelPath, sentinelContent, t)

	if err := regolith.Run("bp_path_only", []string{}, true, "", false, false, false); err != nil {
		t.Fatalf("Expected exact target with only bpPath on packless project to succeed, got: %v", err)
	}
	dotRegolithPath, err := regolith.GetDotRegolith(".")
	if err != nil {
		t.Fatal("Unable to resolve dotRegolithPath:", err)
	}
	assertEditedFilesNoEmptyKeys(dotRegolithPath, 0, 1, t)
	assertSentinelUnchanged(sentinelPath, sentinelContent, t)
}
