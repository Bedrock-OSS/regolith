package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func TestProfileFromObject_SingleExportSetsCompatibilityField(t *testing.T) {
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
	if len(profile.ExportTargets) != 1 {
		t.Fatalf("Expected 1 target, got %d", len(profile.ExportTargets))
	}
	if profile.ExportTargets[0].Target != "local" {
		t.Fatalf("Expected ExportTargets[0] to be local, got %q", profile.ExportTargets[0].Target)
	}
	if profile.ExportTarget.Target != "local" {
		t.Fatalf("Expected deprecated ExportTarget fallback to be local, got %q", profile.ExportTarget.Target)
	}
}

func TestProfileMarshalJSON_DeprecatedExportTargetFallback(t *testing.T) {
	profile := regolith.Profile{
		FilterCollection: regolith.FilterCollection{
			Filters: []regolith.FilterRunner{},
		},
		ExportTarget: regolith.ExportTarget{
			Target: "local",
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
	if err := regolith.Run("multi", []string{}, true, "", false); err != nil {
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

	if err := regolith.Run("multi", []string{}, true, "", false); err != nil {
		t.Fatal("Second multi-target run failed safety checks:", err)
	}

	unexpectedFile := filepath.Join(tmpDir, "target-a", "BP", "unexpected.txt")
	if err := os.WriteFile(unexpectedFile, []byte("not created by regolith"), 0644); err != nil {
		t.Fatal("Unable to create unexpected target file:", err)
	}
	if err := regolith.Run("multi", []string{}, true, "", false); err == nil {
		t.Fatal("Expected file protection to reject unexpected file in first target")
	}
}

func TestRunWithMultipleTargetsIgnoresSymlinkExport(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))
	oldExperiments := regolith.EnabledExperiments
	regolith.EnabledExperiments = []string{"symlink_export"}
	t.Cleanup(func() {
		regolith.EnabledExperiments = oldExperiments
	})

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
	if err := regolith.Run("multi", []string{}, true, "", false); err != nil {
		t.Fatal("Multi-target run with symlink_export enabled failed:", err)
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
	if err := regolith.Run("local_and_development", []string{}, false, "", false); err != nil {
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

	if err := regolith.Run("local_and_development", []string{}, false, "", false); err != nil {
		t.Fatal("Second mixed-target run failed safety checks:", err)
	}
}
