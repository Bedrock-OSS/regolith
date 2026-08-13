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

func mustUnmarshalObject(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatal(err)
	}
	return obj
}

func TestPackSortingAndIsZero(t *testing.T) {
	packs := regolith.Packs{
		BehaviorPacks: []regolith.Pack{
			{Name: "BP2", Source: "b2"},
			{Name: "BP", Source: "b0"},
			{Name: "BP1", Source: "b1"},
		},
		ResourcePacks: []regolith.Pack{
			{Name: "RP", Source: "r0"},
		},
	}
	regolith.SortPacksForTest(packs.BehaviorPacks)
	if packs.BehaviorPacks[0].Name != "BP" ||
		packs.BehaviorPacks[1].Name != "BP1" ||
		packs.BehaviorPacks[2].Name != "BP2" {
		t.Fatalf("packs not sorted by index: %+v", packs.BehaviorPacks)
	}
	if packs.IsZero() {
		t.Fatal("non-empty packs reported IsZero")
	}
	if !(regolith.Packs{}).IsZero() {
		t.Fatal("empty packs should report IsZero")
	}
}

func TestPacksFromObject_SingleStringSugar(t *testing.T) {
	obj := map[string]any{
		"behaviorPack": "./packs/BP",
		"resourcePack": "./packs/RP",
	}
	packs, err := regolith.PacksFromObject(obj, "1.9.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(packs.BehaviorPacks) != 1 || packs.BehaviorPacks[0].Name != "BP" ||
		packs.BehaviorPacks[0].Source != "./packs/BP" {
		t.Fatalf("unexpected behavior packs: %+v", packs.BehaviorPacks)
	}
	if len(packs.ResourcePacks) != 1 || packs.ResourcePacks[0].Name != "RP" {
		t.Fatalf("unexpected resource packs: %+v", packs.ResourcePacks)
	}
}

func TestPacksFromObject_MapForm(t *testing.T) {
	obj := map[string]any{
		"behaviorPacks": map[string]any{
			"BP":  "./packs/BP",
			"BP1": "./packs/addon",
		},
		"resourcePacks": map[string]any{
			"RP": "./packs/RP",
		},
	}
	packs, err := regolith.PacksFromObject(obj, "1.9.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(packs.BehaviorPacks) != 2 ||
		packs.BehaviorPacks[0].Name != "BP" ||
		packs.BehaviorPacks[1].Name != "BP1" ||
		packs.BehaviorPacks[1].Source != "./packs/addon" {
		t.Fatalf("unexpected behavior packs: %+v", packs.BehaviorPacks)
	}
}

func TestPacksFromObject_MapRejectedOnOldVersion(t *testing.T) {
	obj := map[string]any{
		"behaviorPacks": map[string]any{"BP": "./packs/BP"},
	}
	if _, err := regolith.PacksFromObject(obj, "1.8.0"); err == nil {
		t.Fatal("expected error for map form on formatVersion 1.8.0")
	}
}

func TestPacksFromObject_InvalidKey(t *testing.T) {
	for _, badKey := range []string{"BP0", "RP0", "pack", "BP01"} {
		obj := map[string]any{
			"behaviorPacks": map[string]any{badKey: "./x"},
		}
		if _, err := regolith.PacksFromObject(obj, "1.9.0"); err == nil {
			t.Fatalf("expected error for invalid key %q", badKey)
		}
	}
}

func TestPacksFromObject_MapWithoutPrimaryGetsEmptyPrimary(t *testing.T) {
	obj := map[string]any{
		"behaviorPacks": map[string]any{"BP1": "./packs/addon"},
	}
	packs, err := regolith.PacksFromObject(obj, "1.9.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(packs.BehaviorPacks) != 2 || packs.BehaviorPacks[0].Name != "BP" ||
		packs.BehaviorPacks[0].Source != "" {
		t.Fatalf("expected synthesized empty primary BP first: %+v", packs.BehaviorPacks)
	}
}

func TestPacksFromObject_MixedFormRejected(t *testing.T) {
	obj := map[string]any{
		"behaviorPack":  "./packs/BP",
		"behaviorPacks": map[string]any{"BP1": "./x"},
	}
	if _, err := regolith.PacksFromObject(obj, "1.9.0"); err == nil {
		t.Fatal("expected error when mixing behaviorPack and behaviorPacks")
	}
}

func TestPacksMarshal_SingleDefaultProducesStrings(t *testing.T) {
	packs := regolith.Packs{
		BehaviorPacks: []regolith.Pack{{Name: "BP", Source: "./packs/BP"}},
		ResourcePacks: []regolith.Pack{{Name: "RP", Source: "./packs/RP"}},
	}
	data, err := json.Marshal(packs)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatal(err)
	}
	if obj["behaviorPack"] != "./packs/BP" {
		t.Fatalf("expected behaviorPack string, got: %s", data)
	}
	if _, isMap := obj["behaviorPacks"]; isMap {
		t.Fatalf("single pack should not marshal as map: %s", data)
	}
}

func TestPacksMarshal_MultiProducesMaps(t *testing.T) {
	packs := regolith.Packs{
		BehaviorPacks: []regolith.Pack{
			{Name: "BP", Source: "./packs/BP"},
			{Name: "BP1", Source: "./packs/addon"},
		},
		ResourcePacks: []regolith.Pack{{Name: "RP", Source: "./packs/RP"}},
	}
	data, err := json.Marshal(packs)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := regolith.PacksFromObject(mustUnmarshalObject(t, data), "1.9.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(roundTrip.BehaviorPacks) != 2 ||
		roundTrip.BehaviorPacks[1].Name != "BP1" {
		t.Fatalf("round trip lost packs: %+v", roundTrip.BehaviorPacks)
	}
}

func TestExportTargetParsesPerPackMaps(t *testing.T) {
	obj := map[string]any{
		"target": "exact",
		"bpPath": "../out/BP",
		"bpPaths": map[string]any{
			"BP1": "../out/BP1",
		},
		"bpNames": map[string]any{
			"BP1": "'addon_bp'",
		},
	}
	target, err := regolith.ExportTargetFromObject(obj)
	if err != nil {
		t.Fatal(err)
	}
	if target.BpPaths["BP1"] != "../out/BP1" {
		t.Fatalf("expected bpPaths[BP1], got %+v", target.BpPaths)
	}
	if target.BpNames["BP1"] != "'addon_bp'" {
		t.Fatalf("expected bpNames[BP1], got %+v", target.BpNames)
	}
	if target.BpPath != "../out/BP" {
		t.Fatalf("expected bpPath preserved, got %q", target.BpPath)
	}
}

func TestGetExportNameDefaultsAndOverrides(t *testing.T) {
	regolith.InitLogging(true)
	ctx := regolith.RunContext{
		Config: &regolith.Config{Name: "proj"},
	}
	cases := []struct {
		pack     regolith.Pack
		packType string
		target   regolith.ExportTarget
		want     string
	}{
		{regolith.Pack{Name: "BP"}, "bp", regolith.ExportTarget{}, "proj_bp"},
		{regolith.Pack{Name: "BP1"}, "bp", regolith.ExportTarget{}, "proj_bp1"},
		{regolith.Pack{Name: "RP2"}, "rp", regolith.ExportTarget{}, "proj_rp2"},
		{
			regolith.Pack{Name: "BP1"}, "bp",
			regolith.ExportTarget{BpNames: map[string]string{"BP1": "'addon_bp'"}},
			"addon_bp",
		},
		{
			regolith.Pack{Name: "BP"}, "bp",
			regolith.ExportTarget{BpName: "'primary_bp'"},
			"primary_bp",
		},
	}
	for _, c := range cases {
		got, err := regolith.GetExportName(c.target, ctx, c.pack, c.packType)
		if err != nil {
			t.Fatalf("pack %s: %v", c.pack.Name, err)
		}
		if got != c.want {
			t.Fatalf("pack %s: want %q got %q", c.pack.Name, c.want, got)
		}
	}
}

func TestGetPackExportPathLocalAndExact(t *testing.T) {
	regolith.InitLogging(true)
	ctx := regolith.RunContext{
		Config: &regolith.Config{
			Name:            "proj",
			RegolithProject: regolith.RegolithProject{FormatVersion: "1.9.0"},
		},
	}
	// local target -> build/<name>/
	localBp, err := regolith.GetPackExportPath(
		regolith.ExportTarget{Target: "local"}, ctx,
		regolith.Pack{Name: "BP1"}, "bp")
	if err != nil {
		t.Fatal(err)
	}
	if localBp != "build/proj_bp1/" {
		t.Fatalf("local: got %q", localBp)
	}
	// exact target with a per-pack map
	exact := regolith.ExportTarget{
		Target:  "exact",
		BpPath:  "out/BP",
		BpPaths: map[string]string{"BP1": "out/BP1"},
	}
	primary, err := regolith.GetPackExportPath(exact, ctx, regolith.Pack{Name: "BP"}, "bp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(filepath.ToSlash(primary), "out/BP") {
		t.Fatalf("exact primary: got %q", primary)
	}
	addon, err := regolith.GetPackExportPath(exact, ctx, regolith.Pack{Name: "BP1"}, "bp")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(filepath.ToSlash(addon), "out/BP1") {
		t.Fatalf("exact addon: got %q", addon)
	}
	// exact target missing a path for an extra pack -> error
	_, err = regolith.GetPackExportPath(
		regolith.ExportTarget{Target: "exact", BpPath: "out/BP"}, ctx,
		regolith.Pack{Name: "BP1"}, "bp")
	if err == nil {
		t.Fatal("expected error for extra pack without exact path")
	}
}

func TestSetupTmpFilesCreatesAllPackFolders(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))
	tmpDir := prepareTestDirectory(
		fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()), t)
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(minimalProjectPath, workingDir, t)

	config := []byte(`{
		"$schema": "x",
		"name": "regolith_test_project",
		"author": "Bedrock-OSS",
		"packs": {
			"behaviorPacks": { "BP": "./packs/BP", "BP1": "./packs/BP" },
			"resourcePacks": { "RP": "./packs/RP" }
		},
		"regolith": {
			"formatVersion": "1.9.0",
			"profiles": {
				"dev": { "filters": [], "export": { "target": "local" } }
			},
			"dataPath": "./packs/data"
		}
	}`)
	if err := os.WriteFile(filepath.Join(workingDir, "config.json"), config, 0644); err != nil {
		t.Fatal(err)
	}
	os.Chdir(workingDir)
	if err := regolith.Run("dev", []string{}, true, "", false, false, false); err != nil {
		t.Fatal("run failed:", err)
	}
	for _, packName := range []string{"BP", "BP1", "RP", "data"} {
		assertDirExistsOrFatal(
			filepath.Join(workingDir, ".regolith", "tmp", packName), t)
	}
}

func TestMultiPackInplaceExportRestoresAllSources(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))
	regolith.InitLogging(true)
	defer regolith.ShutdownLogging()

	tmpDir := prepareTestDirectory(
		fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()), t)
	copyFilesOrFatal(minimalProjectPath, tmpDir, t)

	// Add a second behavior pack source.
	if err := os.MkdirAll(filepath.Join(tmpDir, "packs", "BP1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(tmpDir, "packs", "BP1", "old.txt"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	config := []byte(`{
		"$schema": "x",
		"name": "regolith_test_project",
		"author": "Bedrock-OSS",
		"packs": {
			"behaviorPacks": { "BP": "./packs/BP", "BP1": "./packs/BP1" },
			"resourcePacks": { "RP": "./packs/RP" }
		},
		"regolith": {
			"formatVersion": "1.9.0",
			"profiles": { "dev": { "filters": [], "export": { "target": "local" } } },
			"dataPath": "./packs/data"
		}
	}`)
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), config, 0644); err != nil {
		t.Fatal(err)
	}
	os.Chdir(tmpDir)

	// Populate the tmp pack folders as if filters had produced output.
	tmpRoot := filepath.Join(tmpDir, ".regolith", "tmp")
	for name, content := range map[string]string{
		"BP": "bp-out", "BP1": "bp1-out", "RP": "rp-out", "data": "data-out",
	} {
		dir := filepath.Join(tmpRoot, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "out.txt"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	configMap, err := regolith.LoadConfigAsMap()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := regolith.ConfigFromObject(configMap)
	if err != nil {
		t.Fatal(err)
	}
	if err := regolith.InplaceExportProject(parsed, ".regolith"); err != nil {
		t.Fatal("InplaceExportProject failed:", err)
	}

	// Each pack source now holds the tmp output; the old BP1 file is gone.
	assertFileContentForTest(t, filepath.Join(tmpDir, "packs", "BP", "out.txt"), "bp-out")
	assertFileContentForTest(t, filepath.Join(tmpDir, "packs", "BP1", "out.txt"), "bp1-out")
	assertFileContentForTest(t, filepath.Join(tmpDir, "packs", "RP", "out.txt"), "rp-out")
	if _, err := os.Stat(filepath.Join(tmpDir, "packs", "BP1", "old.txt")); !os.IsNotExist(err) {
		t.Fatal("expected old BP1 file to be replaced")
	}
}

func assertFileContentForTest(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %q: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("%q: want %q got %q", path, want, string(data))
	}
}

func TestWatchRootsIncludeAllPackSources(t *testing.T) {
	config := &regolith.Config{
		Packs: regolith.Packs{
			BehaviorPacks: []regolith.Pack{
				{Name: "BP", Source: "./packs/BP"},
				{Name: "BP1", Source: "./packs/addon"},
			},
			ResourcePacks: []regolith.Pack{{Name: "RP", Source: "./packs/RP"}},
		},
		RegolithProject: regolith.RegolithProject{DataPath: "./packs/data"},
	}
	roots := regolith.WatchRootsForTest(config)
	for _, want := range []string{"./packs/BP", "./packs/addon", "./packs/RP", "./packs/data"} {
		found := false
		for _, r := range roots {
			if r == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("watch roots missing %q: %v", want, roots)
		}
	}
}

func TestMultiPackLocalExport(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))
	tmpDir := prepareTestDirectory(
		fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()), t)
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(minimalProjectPath, workingDir, t)

	config := []byte(`{
		"$schema": "x",
		"name": "regolith_test_project",
		"author": "Bedrock-OSS",
		"packs": {
			"behaviorPacks": { "BP": "./packs/BP", "BP1": "./packs/BP" },
			"resourcePacks": { "RP": "./packs/RP" }
		},
		"regolith": {
			"formatVersion": "1.9.0",
			"profiles": {
				"dev": { "filters": [], "export": { "target": "local" } }
			},
			"dataPath": "./packs/data"
		}
	}`)
	if err := os.WriteFile(filepath.Join(workingDir, "config.json"), config, 0644); err != nil {
		t.Fatal(err)
	}
	os.Chdir(workingDir)
	if err := regolith.Run("dev", []string{}, true, "", false, false, false); err != nil {
		t.Fatal("run failed:", err)
	}
	srcBp := filepath.Join(workingDir, "packs", "BP")
	comparePaths(srcBp, filepath.Join(workingDir, "build", "regolith_test_project_bp"), t)
	comparePaths(srcBp, filepath.Join(workingDir, "build", "regolith_test_project_bp1"), t)
	comparePaths(
		filepath.Join(workingDir, "packs", "RP"),
		filepath.Join(workingDir, "build", "regolith_test_project_rp"), t)
	// running again must pass file-protection safety checks
	if err := regolith.Run("dev", []string{}, true, "", false, false, false); err != nil {
		t.Fatal("second run failed:", err)
	}
}

const multiPackProjectPath = "testdata/multi_pack/project"

func TestMultiPackProjectLocalBuild(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))
	tmpDir := prepareTestDirectory(
		fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()), t)
	copyFilesOrFatal(multiPackProjectPath, tmpDir, t)
	os.Chdir(tmpDir)
	if err := regolith.Run("dev", []string{}, true, "", false, false, false); err != nil {
		t.Fatal("run failed:", err)
	}
	comparePaths(
		filepath.Join(tmpDir, "packs", "BP"),
		filepath.Join(tmpDir, "build", "multi_pack_project_bp"), t)
	comparePaths(
		filepath.Join(tmpDir, "packs", "BP1"),
		filepath.Join(tmpDir, "build", "multi_pack_project_bp1"), t)
	comparePaths(
		filepath.Join(tmpDir, "packs", "RP"),
		filepath.Join(tmpDir, "build", "multi_pack_project_rp"), t)
}

func TestMultiPackExactExport(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))
	tmpDir := prepareTestDirectory(
		fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()), t)
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(multiPackProjectPath, workingDir, t)

	config := []byte(`{
		"$schema": "x",
		"name": "multi_pack_project",
		"author": "Bedrock-OSS",
		"packs": {
			"behaviorPacks": { "BP": "./packs/BP", "BP1": "./packs/BP1" },
			"resourcePacks": { "RP": "./packs/RP" }
		},
		"regolith": {
			"formatVersion": "1.9.0",
			"dataPath": "./packs/data",
			"profiles": {
				"exact": {
					"filters": [],
					"export": {
						"target": "exact",
						"bpPath": "../out/BP",
						"rpPath": "../out/RP",
						"bpPaths": { "BP1": "../out/BP1" }
					}
				}
			}
		}
	}`)
	if err := os.WriteFile(filepath.Join(workingDir, "config.json"), config, 0644); err != nil {
		t.Fatal(err)
	}
	os.Chdir(workingDir)
	if err := regolith.Run("exact", []string{}, true, "", false, false, false); err != nil {
		t.Fatal("exact run failed:", err)
	}
	comparePaths(filepath.Join(workingDir, "packs", "BP"), filepath.Join(tmpDir, "out", "BP"), t)
	comparePaths(filepath.Join(workingDir, "packs", "BP1"), filepath.Join(tmpDir, "out", "BP1"), t)
	comparePaths(filepath.Join(workingDir, "packs", "RP"), filepath.Join(tmpDir, "out", "RP"), t)
}

func TestMultiPackDevelopmentExport(t *testing.T) {
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
		"$schema": "x",
		"name": "regolith_test_project",
		"author": "Bedrock-OSS",
		"packs": {
			"behaviorPacks": { "BP": "./packs/BP", "BP1": "./packs/BP" },
			"resourcePacks": { "RP": "./packs/RP" }
		},
		"regolith": {
			"formatVersion": "1.9.0",
			"profiles": {
				"dev": {
					"filters": [],
					"export": { "target": "development", "build": "standard" }
				}
			},
			"dataPath": "./packs/data"
		}
	}`)
	if err := os.WriteFile(filepath.Join(workingDir, "config.json"), config, 0644); err != nil {
		t.Fatal("Unable to write multi-pack development config:", err)
	}

	os.Chdir(workingDir)
	if err := regolith.Run("dev", []string{}, false, "", false, false, false); err != nil {
		t.Fatal("First multi-pack development run failed:", err)
	}

	srcBp := filepath.Join(workingDir, "packs", "BP")
	srcRp := filepath.Join(workingDir, "packs", "RP")
	comparePaths(srcBp, filepath.Join(mojangDir, "development_behavior_packs", "regolith_test_project_bp"), t)
	comparePaths(srcBp, filepath.Join(mojangDir, "development_behavior_packs", "regolith_test_project_bp1"), t)
	comparePaths(srcRp, filepath.Join(mojangDir, "development_resource_packs", "regolith_test_project_rp"), t)

	if err := regolith.Run("dev", []string{}, false, "", false, false, false); err != nil {
		t.Fatal("Second multi-pack development run failed safety checks:", err)
	}
}

func TestMultiPackExactExportMissingPathFails(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))
	tmpDir := prepareTestDirectory(
		fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()), t)
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(multiPackProjectPath, workingDir, t)

	config := []byte(`{
		"$schema": "x",
		"name": "multi_pack_project",
		"author": "Bedrock-OSS",
		"packs": {
			"behaviorPacks": { "BP": "./packs/BP", "BP1": "./packs/BP1" },
			"resourcePacks": { "RP": "./packs/RP" }
		},
		"regolith": {
			"formatVersion": "1.9.0",
			"dataPath": "./packs/data",
			"profiles": {
				"exact": {
					"filters": [],
					"export": { "target": "exact", "bpPath": "../out/BP", "rpPath": "../out/RP" }
				}
			}
		}
	}`)
	if err := os.WriteFile(filepath.Join(workingDir, "config.json"), config, 0644); err != nil {
		t.Fatal(err)
	}
	os.Chdir(workingDir)
	if err := regolith.Run("exact", []string{}, true, "", false, false, false); err == nil {
		t.Fatal("expected error: BP1 has no exact path")
	}
}

func TestMultiPackWorldExport(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))
	tmpDir := prepareTestDirectory(
		fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()), t)
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(minimalProjectPath, workingDir, t)

	// worldPath is relative to workingDir; resolves to tmpDir/myworld.
	worldDir := filepath.Join(tmpDir, "myworld")

	config := []byte(`{
		"$schema": "x",
		"name": "regolith_test_project",
		"author": "Bedrock-OSS",
		"packs": {
			"behaviorPacks": { "BP": "./packs/BP", "BP1": "./packs/BP" },
			"resourcePacks": { "RP": "./packs/RP" }
		},
		"regolith": {
			"formatVersion": "1.9.0",
			"profiles": {
				"world": {
					"filters": [],
					"export": { "target": "world", "worldPath": "../myworld" }
				}
			},
			"dataPath": "./packs/data"
		}
	}`)
	if err := os.WriteFile(filepath.Join(workingDir, "config.json"), config, 0644); err != nil {
		t.Fatal(err)
	}
	os.Chdir(workingDir)
	if err := regolith.Run("world", []string{}, true, "", false, false, false); err != nil {
		t.Fatal("first world run failed:", err)
	}

	srcBp := filepath.Join(workingDir, "packs", "BP")
	srcRp := filepath.Join(workingDir, "packs", "RP")
	comparePaths(srcBp, filepath.Join(worldDir, "behavior_packs", "regolith_test_project_bp"), t)
	comparePaths(srcBp, filepath.Join(worldDir, "behavior_packs", "regolith_test_project_bp1"), t)
	comparePaths(srcRp, filepath.Join(worldDir, "resource_packs", "regolith_test_project_rp"), t)

	// Second run must pass file-protection safety checks.
	if err := regolith.Run("world", []string{}, true, "", false, false, false); err != nil {
		t.Fatal("second world run failed:", err)
	}
}
