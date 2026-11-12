package regolith

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// testTarEntry describes an entry of a tarball created by createTestTarball.
type testTarEntry struct {
	name     string // Full name of the entry, including the top-level directory
	content  string
	typeflag byte // 0 means tar.TypeReg
	mode     int64
	linkname string
}

// createTestTarball creates a .tgz file with the given entries at path.
func createTestTarball(t *testing.T, path string, entries []testTarEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create tarball: %v", err)
	}
	defer f.Close()
	gzWriter := gzip.NewWriter(f)
	defer gzWriter.Close()
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()
	for _, entry := range entries {
		typeflag := entry.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		mode := entry.mode
		if mode == 0 {
			mode = 0644
		}
		header := &tar.Header{
			Name:     entry.name,
			Mode:     mode,
			Typeflag: typeflag,
			Linkname: entry.linkname,
		}
		if typeflag == tar.TypeReg {
			header.Size = int64(len(entry.content))
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("Failed to write tar header for %s: %v", entry.name, err)
		}
		if typeflag == tar.TypeReg {
			if _, err := tarWriter.Write([]byte(entry.content)); err != nil {
				t.Fatalf("Failed to write tar content for %s: %v", entry.name, err)
			}
		}
	}
}

// packageTarball creates a tarball shaped like the output of "npm pack" (all
// entries under "package/") from a map of file names to contents.
func packageTarball(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	entries := make([]testTarEntry, 0, len(files))
	for name, content := range files {
		entries = append(entries, testTarEntry{name: "package/" + name, content: content})
	}
	path := filepath.Join(dir, "test-package-1.0.0.tgz")
	createTestTarball(t, path, entries)
	return path
}

func readFileOrFatal(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", path, err)
	}
	return string(content)
}

func TestExtractNpmTarball(t *testing.T) {
	tmpDir := t.TempDir()
	files := map[string]string{
		"filter.json":      `{"filters": [{"runWith": "nodejs", "script": "filter.js"}]}`,
		"filter.js":        `console.log("Hello from npm filter!");`,
		"package.json":     `{"name": "@test/my-filter", "version": "1.0.0"}`,
		"lib/helper.js":    `module.exports = {};`,
		"data/config.json": `{"setting": true}`,
	}
	tarballPath := packageTarball(t, tmpDir, files)
	destPath := filepath.Join(tmpDir, "extracted")
	if err := extractNpmTarball(tarballPath, destPath); err != nil {
		t.Fatalf("extractNpmTarball failed: %v", err)
	}
	for name, content := range files {
		got := readFileOrFatal(t, filepath.Join(destPath, filepath.FromSlash(name)))
		if got != content {
			t.Fatalf("Content of %s differs: got %q, want %q", name, got, content)
		}
	}
	if _, err := os.Stat(filepath.Join(destPath, "package")); !os.IsNotExist(err) {
		t.Fatal("The top-level \"package\" directory should be stripped")
	}
}

func TestExtractNpmTarball_StripsAnyTopLevelDirectory(t *testing.T) {
	// Some tools name the top-level directory after the package instead of
	// "package". Exactly one leading component is stripped either way.
	tmpDir := t.TempDir()
	tarballPath := filepath.Join(tmpDir, "p.tgz")
	createTestTarball(t, tarballPath, []testTarEntry{
		{name: "my-filter/", typeflag: tar.TypeDir, mode: 0755},
		{name: "my-filter/filter.json", content: "{}"},
		{name: "./my-filter/nested/deep/file.txt", content: "deep"},
	})
	destPath := filepath.Join(tmpDir, "extracted")
	if err := extractNpmTarball(tarballPath, destPath); err != nil {
		t.Fatalf("extractNpmTarball failed: %v", err)
	}
	if readFileOrFatal(t, filepath.Join(destPath, "filter.json")) != "{}" {
		t.Fatal("filter.json not extracted to the package root")
	}
	if readFileOrFatal(t, filepath.Join(destPath, "nested", "deep", "file.txt")) != "deep" {
		t.Fatal("nested file not extracted")
	}
}

func TestExtractNpmTarball_RejectsEntriesOutsideOfDestination(t *testing.T) {
	cases := []string{
		"package/../evil.txt",
		"package/sub/../../evil.txt",
		"package/../../evil.txt",
	}
	for _, name := range cases {
		tmpDir := t.TempDir()
		tarballPath := filepath.Join(tmpDir, "p.tgz")
		createTestTarball(t, tarballPath, []testTarEntry{
			{name: "package/filter.json", content: "{}"},
			{name: name, content: "evil"},
		})
		destPath := filepath.Join(tmpDir, "extracted")
		err := extractNpmTarball(tarballPath, destPath)
		if err == nil {
			t.Fatalf("Expected an error for the %q entry, got nil", name)
		}
		if _, err := os.Stat(filepath.Join(tmpDir, "evil.txt")); !os.IsNotExist(err) {
			t.Fatalf("The %q entry was extracted outside of the destination", name)
		}
	}
}

func TestExtractNpmTarball_SkipsLinks(t *testing.T) {
	tmpDir := t.TempDir()
	tarballPath := filepath.Join(tmpDir, "p.tgz")
	createTestTarball(t, tarballPath, []testTarEntry{
		{name: "package/filter.json", content: "{}"},
		{name: "package/link.json", typeflag: tar.TypeSymlink, linkname: "../../outside.json"},
		{name: "package/hard.json", typeflag: tar.TypeLink, linkname: "filter.json"},
	})
	destPath := filepath.Join(tmpDir, "extracted")
	if err := extractNpmTarball(tarballPath, destPath); err != nil {
		t.Fatalf("extractNpmTarball failed: %v", err)
	}
	for _, name := range []string{"link.json", "hard.json"} {
		if _, err := os.Lstat(filepath.Join(destPath, name)); !os.IsNotExist(err) {
			t.Fatalf("The %q link should be skipped", name)
		}
	}
}

func TestExtractNpmTarball_KeepsExecutableBit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("File modes aren't preserved on Windows")
	}
	tmpDir := t.TempDir()
	tarballPath := filepath.Join(tmpDir, "p.tgz")
	createTestTarball(t, tarballPath, []testTarEntry{
		{name: "package/filter.json", content: "{}"},
		{name: "package/run.sh", content: "#!/bin/sh\n", mode: 0755},
		{name: "package/odd.txt", content: "odd", mode: 0040},
	})
	destPath := filepath.Join(tmpDir, "extracted")
	if err := extractNpmTarball(tarballPath, destPath); err != nil {
		t.Fatalf("extractNpmTarball failed: %v", err)
	}
	info, err := os.Stat(filepath.Join(destPath, "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0100 == 0 {
		t.Fatalf("run.sh should be executable, mode is %v", info.Mode())
	}
	info, err = os.Stat(filepath.Join(destPath, "odd.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0600 != 0600 {
		t.Fatalf("odd.txt should be readable and writable by the owner, mode is %v", info.Mode())
	}
}

func TestExtractNpmTarball_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.tgz")
	os.WriteFile(invalidPath, []byte("this is not a tarball"), 0644)
	err := extractNpmTarball(invalidPath, filepath.Join(tmpDir, "extracted"))
	if err == nil {
		t.Fatal("Expected error for invalid tarball, got nil")
	}
}

func TestNpmFilterDefinitionFromObject(t *testing.T) {
	cases := []struct {
		name    string
		obj     map[string]any
		wantErr bool
		check   func(t *testing.T, def *NpmFilterDefinition)
	}{
		{
			name: "valid",
			obj:  map[string]any{"npm": "@scope/my-filter", "version": "1.0.0"},
			check: func(t *testing.T, def *NpmFilterDefinition) {
				if def.Package != "@scope/my-filter" || def.Version != "1.0.0" || def.Id != "my-filter" {
					t.Fatalf("Unexpected definition: %+v", def)
				}
			},
		},
		{
			name: "venvSlot",
			obj:  map[string]any{"npm": "@scope/my-filter", "version": "2.0.0", "venvSlot": float64(3)},
			check: func(t *testing.T, def *NpmFilterDefinition) {
				if def.VenvSlot != 3 {
					t.Fatalf("Expected VenvSlot 3, got %d", def.VenvSlot)
				}
			},
		},
		{name: "range", obj: map[string]any{"npm": "x", "version": "^1.0.0"}},
		{name: "dist-tag", obj: map[string]any{"npm": "x", "version": "latest"}},
		{name: "npm not a string", obj: map[string]any{"npm": 1, "version": "1.0.0"}, wantErr: true},
		{name: "npm empty", obj: map[string]any{"npm": "", "version": "1.0.0"}, wantErr: true},
		{name: "missing version", obj: map[string]any{"npm": "x"}, wantErr: true},
		{name: "wrong version type", obj: map[string]any{"npm": "x", "version": 123}, wantErr: true},
		{name: "empty version", obj: map[string]any{"npm": "x", "version": ""}, wantErr: true},
		{name: "HEAD", obj: map[string]any{"npm": "x", "version": "HEAD"}, wantErr: true},
		{name: "npm and url", obj: map[string]any{"npm": "x", "url": "github.com/a/b", "version": "1.0.0"}, wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			def, err := NpmFilterDefinitionFromObject("my-filter", c.obj)
			if c.wantErr {
				if err == nil {
					t.Fatal("Expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if c.check != nil {
				c.check(t, def)
			}
		})
	}
}

func TestFilterInstallerFromObject_NpmDetection(t *testing.T) {
	// An object with "npm" and no "runWith" produces an NpmFilterDefinition
	installer, err := FilterInstallerFromObject(
		"my-filter", "my-filter",
		map[string]any{"npm": "@scope/my-filter", "version": "1.0.0"})
	if err != nil {
		t.Fatalf("FilterInstallerFromObject failed: %v", err)
	}
	if _, ok := installer.(*NpmFilterDefinition); !ok {
		t.Fatalf("Expected *NpmFilterDefinition, got %T", installer)
	}
	// An object with "url" and no "runWith" still produces a
	// RemoteFilterDefinition
	installer, err = FilterInstallerFromObject(
		"some-filter", "some-filter",
		map[string]any{"url": "github.com/Bedrock-OSS/regolith-filters", "version": "1.0.0"})
	if err != nil {
		t.Fatalf("FilterInstallerFromObject failed: %v", err)
	}
	if _, ok := installer.(*RemoteFilterDefinition); !ok {
		t.Fatalf("Expected *RemoteFilterDefinition, got %T", installer)
	}
	// An object with both is an error
	_, err = FilterInstallerFromObject(
		"some-filter", "some-filter",
		map[string]any{"npm": "x", "url": "github.com/a/b", "version": "1.0.0"})
	if err == nil {
		t.Fatal("Expected an error for a definition with both \"npm\" and \"url\"")
	}
}

func TestNpmFilterDefinition_Json(t *testing.T) {
	// The definition is what "regolith install" writes to config.json
	def := &NpmFilterDefinition{
		FilterDefinition: FilterDefinition{Id: "my-filter"},
		Package:          "@scope/my-filter",
		Version:          "1.0.0",
	}
	data, err := json.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"npm":"@scope/my-filter","version":"1.0.0"}` {
		t.Fatalf("Unexpected JSON: %s", data)
	}
}

func TestIsExactNpmVersion(t *testing.T) {
	exact := []string{"1.0.0", "0.0.1", "10.20.30", "1.0.0-beta.1", "1.0.0+build.5", "1.0.0-rc.1+build"}
	notExact := []string{"latest", "next", "HEAD", "^1.0.0", "~1.0.0", "1.0", "1", "1.x", "1.0.x", "*", ">=1.0.0 <2.0.0", "v1.0.0", "", "1.0.0 "}
	for _, v := range exact {
		if !isExactNpmVersion(v) {
			t.Errorf("%q should be an exact version", v)
		}
	}
	for _, v := range notExact {
		if isExactNpmVersion(v) {
			t.Errorf("%q should not be an exact version", v)
		}
	}
}

func TestHighestNpmViewVersion(t *testing.T) {
	cases := []struct {
		output  string
		want    string
		wantErr bool
	}{
		{output: `"1.2.3"` + "\n", want: "1.2.3"},
		{output: `["1.0.0", "1.1.0", "1.1.1"]`, want: "1.1.1"},
		{output: `["1.10.0", "1.9.0"]`, want: "1.10.0"},
		{output: `["2.0.0-beta.1", "2.0.0-alpha.2"]`, want: "2.0.0-beta.1"},
		{output: "", want: ""},
		{output: "  \n", want: ""},
		{output: `{"not": "expected"}`, wantErr: true},
	}
	for _, c := range cases {
		got, err := highestNpmViewVersion([]byte(c.output))
		if c.wantErr {
			if err == nil {
				t.Errorf("highestNpmViewVersion(%q): expected an error", c.output)
			}
			continue
		}
		if err != nil {
			t.Errorf("highestNpmViewVersion(%q): unexpected error %v", c.output, err)
		} else if got != c.want {
			t.Errorf("highestNpmViewVersion(%q) = %q, want %q", c.output, got, c.want)
		}
	}
}

func TestNpmPackTarballPath(t *testing.T) {
	packPath := t.TempDir()
	// The file name from the JSON output of "npm pack --json"
	output := `[{"id": "@scope/my-filter@1.0.0", "name": "@scope/my-filter",` +
		`"version": "1.0.0", "filename": "scope-my-filter-1.0.0.tgz", "files": []}]` + "\n"
	os.WriteFile(filepath.Join(packPath, "scope-my-filter-1.0.0.tgz"), []byte{}, 0644)
	got, err := npmPackTarballPath([]byte(output), packPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(packPath, "scope-my-filter-1.0.0.tgz") {
		t.Fatalf("Unexpected path: %s", got)
	}
	// A reported name that doesn't exist (the "@scope/name-1.0.0.tgz" of
	// some npm versions) falls back to the only .tgz file of the directory
	misreported := `[{"filename": "@scope/my-filter-1.0.0.tgz"}]`
	got, err = npmPackTarballPath([]byte(misreported), packPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(packPath, "scope-my-filter-1.0.0.tgz") {
		t.Fatalf("Unexpected path for a misreported name: %s", got)
	}
	os.Remove(filepath.Join(packPath, "scope-my-filter-1.0.0.tgz"))
	// Without usable output, the only .tgz file of the directory is used
	if _, err := npmPackTarballPath([]byte("not json"), packPath); err == nil {
		t.Fatal("Expected an error when there is no tarball")
	}
	os.WriteFile(filepath.Join(packPath, "a-1.0.0.tgz"), []byte{}, 0644)
	got, err = npmPackTarballPath([]byte("not json"), packPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(packPath, "a-1.0.0.tgz") {
		t.Fatalf("Unexpected path: %s", got)
	}
	os.WriteFile(filepath.Join(packPath, "b-1.0.0.tgz"), []byte{}, 0644)
	if _, err := npmPackTarballPath([]byte("not json"), packPath); err == nil {
		t.Fatal("Expected an error when there are several tarballs")
	}
}

func TestNpmErrorHint(t *testing.T) {
	if hint := npmErrorHint("npm error code E404\nnpm error 404 Not Found"); hint == "" || hint == npmErrorHint("") {
		t.Fatal("Expected a hint for E404")
	}
	if hint := npmErrorHint("npm error code E401"); hint == npmErrorHint("") {
		t.Fatal("Expected a hint for E401")
	}
	if hint := npmErrorHint("npm error code ENOTFOUND"); hint == npmErrorHint("") {
		t.Fatal("Expected a hint for ENOTFOUND")
	}
}

func TestParseNpmInstallArg(t *testing.T) {
	cases := []struct {
		spec        string
		wantPackage string
		wantVersion string
		wantErr     bool
	}{
		{spec: "@scope/name", wantPackage: "@scope/name"},
		{spec: "@scope/name==1.0.0", wantPackage: "@scope/name", wantVersion: "1.0.0"},
		{spec: "@scope/name@1.0.0", wantPackage: "@scope/name", wantVersion: "1.0.0"},
		{spec: "name@^1.0.0", wantPackage: "name", wantVersion: "^1.0.0"},
		{spec: "name==latest", wantPackage: "name", wantVersion: "latest"},
		{spec: "name", wantPackage: "name"},
		{spec: "", wantErr: true},
		{spec: "@", wantErr: true},
		{spec: "@scope/", wantErr: true},
		{spec: "a==b==c", wantErr: true},
	}
	for _, c := range cases {
		pkg, version, err := parseNpmInstallArg("npm:"+c.spec, c.spec)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseNpmInstallArg(%q): expected an error", c.spec)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseNpmInstallArg(%q): unexpected error %v", c.spec, err)
		} else if pkg != c.wantPackage || version != c.wantVersion {
			t.Errorf("parseNpmInstallArg(%q) = %q, %q; want %q, %q",
				c.spec, pkg, version, c.wantPackage, c.wantVersion)
		}
	}
}

func TestNpmPackageFilterName(t *testing.T) {
	if npmPackageFilterName("@scope/name") != "name" {
		t.Fatal("The scope should be stripped")
	}
	if npmPackageFilterName("name") != "name" {
		t.Fatal("An unscoped name should be kept")
	}
}

func TestParseInstallFilterArgs_Npm(t *testing.T) {
	// npm arguments don't touch the resolvers, so no network is needed
	parsed, err := parseInstallFilterArgs(
		[]string{"npm:@scope/my-filter==1.2.3", "npm:other"}, false)
	if err != nil {
		t.Fatalf("parseInstallFilterArgs failed: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("Expected 2 parsed arguments, got %d", len(parsed))
	}
	if parsed[0].npmPackage != "@scope/my-filter" || parsed[0].name != "my-filter" ||
		parsed[0].version != "1.2.3" || parsed[0].url != "" {
		t.Fatalf("Unexpected parsed argument: %+v", parsed[0])
	}
	if parsed[1].npmPackage != "other" || parsed[1].name != "other" || parsed[1].version != "" {
		t.Fatalf("Unexpected parsed argument: %+v", parsed[1])
	}
	_, err = parseInstallFilterArgs([]string{"npm:x", "npm:x==1.0.0"}, false)
	if err == nil {
		t.Fatal("Expected an error for a duplicate filter")
	}
	_, err = parseInstallFilterArgs([]string{"npm:"}, false)
	if err == nil {
		t.Fatal("Expected an error for an empty package name")
	}
}

func TestNpmSaveAndLoadVersionInfo(t *testing.T) {
	dotRegolith := filepath.Join(t.TempDir(), ".regolith")
	def := &NpmFilterDefinition{
		FilterDefinition: FilterDefinition{Id: "test-filter"},
		Package:          "@test/filter",
		Version:          "1.0.0",
	}
	filterDir := def.GetDownloadPath(dotRegolith)
	if filterDir != filepath.Join(dotRegolith, "cache/filters", "test-filter") {
		t.Fatalf("Unexpected download path: %s", filterDir)
	}
	os.MkdirAll(filterDir, 0755)
	os.WriteFile(filepath.Join(filterDir, "filter.json"), []byte(`{"filters": []}`), 0644)
	if err := def.SaveVersionInfo("1.0.0", dotRegolith); err != nil {
		t.Fatalf("SaveVersionInfo failed: %v", err)
	}
	version, err := def.InstalledVersion(dotRegolith)
	if err != nil {
		t.Fatalf("InstalledVersion failed: %v", err)
	}
	if version != "1.0.0" {
		t.Fatalf("Expected version %q, got %q", "1.0.0", version)
	}
}

func TestNpmCreateFilterRunner(t *testing.T) {
	def := &NpmFilterDefinition{
		FilterDefinition: FilterDefinition{Id: "my-filter"},
		Package:          "@scope/my-filter",
		Version:          "1.0.0",
		VenvSlot:         2,
	}
	runner, err := def.CreateFilterRunner(map[string]any{}, "my-filter")
	if err != nil {
		t.Fatalf("CreateFilterRunner failed: %v", err)
	}
	remoteFilter, ok := runner.(*RemoteFilter)
	if !ok {
		t.Fatalf("Expected *RemoteFilter, got %T", runner)
	}
	if remoteFilter.Definition.Version != "1.0.0" || remoteFilter.Definition.VenvSlot != 2 ||
		remoteFilter.Definition.Id != "my-filter" {
		t.Fatalf("Unexpected runner definition: %+v", remoteFilter.Definition)
	}
	// A range or a dist-tag doesn't name the cached version; the runner
	// skips its version check for them like it does for "latest".
	for _, version := range []string{"^1.0.0", "next"} {
		def.Version = version
		runner, _ := def.CreateFilterRunner(map[string]any{}, "my-filter")
		if runner.(*RemoteFilter).Definition.Version != "latest" {
			t.Fatalf("The runner of the %q version should skip the version check", version)
		}
	}
}
