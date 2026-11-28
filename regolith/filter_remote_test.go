package regolith

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// createTestZip creates a .zip file at path with the given entries. Keys are
// the names of the entries, values are the contents. Names ending with "/"
// are directories.
func createTestZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create zip: %v", err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatalf("Failed to create zip entry %q: %v", name, err)
		}
		if _, err := entry.Write([]byte(entries[name])); err != nil {
			t.Fatalf("Failed to write zip entry %q: %v", name, err)
		}
	}
}

func TestGhReleaseFileNames(t *testing.T) {
	got := ghReleaseFileNames("my-filter-1.2.3", "windows", "amd64")
	want := []string{
		"my-filter-1.2.3-windows-amd64.zip",
		"my-filter-1.2.3-windows.zip",
		"my-filter-1.2.3.zip",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPickGhReleaseFile(t *testing.T) {
	wanted := ghReleaseFileNames("f-1.0.0", "linux", "arm64")
	cases := []struct {
		name      string
		available []string
		want      string
	}{
		{"os and arch first", []string{"f-1.0.0.zip", "f-1.0.0-linux.zip", "f-1.0.0-linux-arm64.zip"}, "f-1.0.0-linux-arm64.zip"},
		{"os when no arch", []string{"f-1.0.0.zip", "f-1.0.0-linux.zip", "f-1.0.0-linux-amd64.zip"}, "f-1.0.0-linux.zip"},
		{"generic fallback", []string{"f-1.0.0-windows.zip", "f-1.0.0.zip"}, "f-1.0.0.zip"},
		{"no match", []string{"f-1.0.0-windows.zip", "f-1.0.0.tar.gz", "f-1.0.1.zip"}, ""},
		{"empty", nil, ""},
	}
	for _, c := range cases {
		if got := pickGhReleaseFile(c.available, wanted); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestGetGhRepoName(t *testing.T) {
	cases := map[string]string{
		"github.com/Bedrock-OSS/regolith-filters":     "Bedrock-OSS/regolith-filters",
		"github.com/user/repo/sub":                    "user/repo/sub",
		"gitlab.com/user/repo":                        "",
		"github.com":                                  "",
		"raw.githubusercontent.com/Bedrock-OSS/stuff": "",
	}
	for url, want := range cases {
		f := RemoteFilterDefinition{Url: url}
		if got := f.getGhRepoName(); got != want {
			t.Errorf("getGhRepoName(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestUnzip(t *testing.T) {
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "filter.zip")
	createTestZip(t, zipPath, map[string]string{
		"filter.json":     `{"filters": []}`,
		"data/":           "",
		"src/main.py":     "print('hi')",
		"src/lib/util.py": "x = 1",
	})
	target := filepath.Join(tmp, "out")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := unzip(zipPath, target); err != nil {
		t.Fatalf("unzip failed: %v", err)
	}
	for name, content := range map[string]string{
		"filter.json":     `{"filters": []}`,
		"src/main.py":     "print('hi')",
		"src/lib/util.py": "x = 1",
	} {
		got, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		if string(got) != content {
			t.Errorf("%s: got %q, want %q", name, got, content)
		}
	}
	if stat, err := os.Stat(filepath.Join(target, "data")); err != nil || !stat.IsDir() {
		t.Errorf("directory entry 'data' was not created: %v", err)
	}
}

func TestUnzipErrors(t *testing.T) {
	tmp := t.TempDir()
	goodZip := filepath.Join(tmp, "good.zip")
	createTestZip(t, goodZip, map[string]string{"filter.json": "{}"})

	t.Run("target does not exist", func(t *testing.T) {
		if err := unzip(goodZip, filepath.Join(tmp, "missing")); err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("target is a file", func(t *testing.T) {
		file := filepath.Join(tmp, "file.txt")
		os.WriteFile(file, []byte("x"), 0644)
		if err := unzip(goodZip, file); err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("not a zip", func(t *testing.T) {
		notZip := filepath.Join(tmp, "not.zip")
		os.WriteFile(notZip, []byte("definitely not a zip"), 0644)
		target := t.TempDir()
		if err := unzip(notZip, target); err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("file already exists", func(t *testing.T) {
		target := t.TempDir()
		existing := filepath.Join(target, "filter.json")
		os.WriteFile(existing, []byte("old"), 0644)
		if err := unzip(goodZip, target); err == nil {
			t.Fatal("expected an error")
		}
		if got, _ := os.ReadFile(existing); string(got) != "old" {
			t.Errorf("existing file was overwritten: %q", got)
		}
	})
	for _, name := range []string{"../evil.txt", "a/../../evil.txt", "/abs/evil.txt"} {
		t.Run("non-local path "+name, func(t *testing.T) {
			evilZip := filepath.Join(t.TempDir(), "evil.zip")
			createTestZip(t, evilZip, map[string]string{name: "evil"})
			parent := t.TempDir()
			target := filepath.Join(parent, "target")
			os.Mkdir(target, 0755)
			if err := unzip(evilZip, target); err == nil {
				t.Fatal("expected an error")
			}
			if _, err := os.Stat(filepath.Join(parent, "evil.txt")); err == nil {
				t.Error("file was written outside of the target directory")
			}
		})
	}
}

// pythonWriteFileScript returns a cross-platform Python snippet that writes
// content to BP/<filename> relative to the filter working directory.
func pythonWriteFileScript(filename, content string) string {
	return fmt.Sprintf(
		"from pathlib import Path\nPath(\"BP\", %q).write_text(%q, encoding=\"utf-8\")\n",
		filename, content,
	)
}

func setupRemoteFilterTest(t *testing.T, filterJSON string, scripts map[string]string) (*RemoteFilter, string, string) {
	t.Helper()
	// Create TempDir before Chdir so LIFO cleanup restores cwd before removing it.
	projectDir := t.TempDir()
	t.Chdir(projectDir)

	dotRegolithPath := filepath.Join(projectDir, ".regolith")
	workDir, err := GetAbsoluteWorkingDirectory(dotRegolithPath)
	if err != nil {
		t.Fatalf("GetAbsoluteWorkingDirectory failed: %v", err)
	}
	if rel, relErr := filepath.Rel(projectDir, workDir); relErr != nil || strings.HasPrefix(rel, "..") {
		t.Cleanup(func() {
			_ = os.RemoveAll(workDir)
		})
	}
	if err := os.MkdirAll(filepath.Join(workDir, "BP"), 0755); err != nil {
		t.Fatalf("Failed to create working BP directory: %v", err)
	}

	filterDir := filepath.Join(dotRegolithPath, "cache", "filters", "test_remote")
	if err := os.MkdirAll(filterDir, 0755); err != nil {
		t.Fatalf("Failed to create remote filter directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filterDir, "filter.json"), []byte(filterJSON), 0644); err != nil {
		t.Fatalf("Failed to write filter.json: %v", err)
	}
	for name, content := range scripts {
		scriptPath := filepath.Join(filterDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(scriptPath), 0755); err != nil {
			t.Fatalf("Failed to create script parent dir: %v", err)
		}
		if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write script %s: %v", name, err)
		}
	}

	def := RemoteFilterDefinition{
		FilterDefinition: FilterDefinition{Id: "test_remote"},
		Url:              "example.com/test/filters",
		Version:          "1.0.0",
	}
	runner, err := def.CreateFilterRunner(map[string]any{}, "test_remote")
	if err != nil {
		t.Fatalf("CreateFilterRunner failed: %v", err)
	}
	rf, ok := runner.(*RemoteFilter)
	if !ok {
		t.Fatalf("Expected *RemoteFilter, got %T", runner)
	}
	return rf, dotRegolithPath, workDir
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Expected file %s to exist: %v", path, err)
	}
}

func assertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("Expected file %s not to exist", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Unexpected stat error for %s: %v", path, err)
	}
}

func TestRemoteFilterSubfilterModeCondition(t *testing.T) {
	const filterJSON = `{
		"version": "1.0.0",
		"filters": [
			{"runWith": "python", "script": "watch.py", "when": "mode == 'watch'"},
			{"runWith": "python", "script": "run.py", "when": "mode == 'run'"}
		]
	}`
	scripts := map[string]string{
		"watch.py": pythonWriteFileScript("watch.txt", "watch"),
		"run.py":   pythonWriteFileScript("run.txt", "run"),
	}

	t.Run("run mode", func(t *testing.T) {
		rf, dotRegolithPath, workDir := setupRemoteFilterTest(t, filterJSON, scripts)
		ctx := RunContext{
			Config:          &Config{Name: "test", Author: "test"},
			Profile:         "default",
			DotRegolithPath: dotRegolithPath,
		}
		interrupted, err := rf.Run(ctx)
		if err != nil {
			t.Fatalf("rf.Run failed: %v", err)
		}
		if interrupted {
			t.Fatal("Expected interrupted == false in run mode")
		}
		assertFileExists(t, filepath.Join(workDir, "BP", "run.txt"))
		assertFileNotExists(t, filepath.Join(workDir, "BP", "watch.txt"))
	})

	t.Run("watch mode", func(t *testing.T) {
		rf, dotRegolithPath, workDir := setupRemoteFilterTest(t, filterJSON, scripts)
		ctx := RunContext{
			Config:          &Config{Name: "test", Author: "test"},
			Profile:         "default",
			DotRegolithPath: dotRegolithPath,
			interruption:    make(chan string, 1),
		}
		interrupted, err := rf.Run(ctx)
		if err != nil {
			t.Fatalf("rf.Run failed: %v", err)
		}
		if interrupted {
			t.Fatal("Expected interrupted == false when no interruption was sent")
		}
		assertFileExists(t, filepath.Join(workDir, "BP", "watch.txt"))
		assertFileNotExists(t, filepath.Join(workDir, "BP", "run.txt"))
	})
}

func TestRemoteFilterSubfilterInterruptionPropagation(t *testing.T) {
	const filterJSON = `{
		"version": "1.0.0",
		"filters": [
			{"runWith": "python", "script": "first.py"},
			{"runWith": "python", "script": "second.py"}
		]
	}`
	scripts := map[string]string{
		"first.py":  pythonWriteFileScript("first.txt", "first"),
		"second.py": pythonWriteFileScript("second.txt", "second"),
	}

	rf, dotRegolithPath, workDir := setupRemoteFilterTest(t, filterJSON, scripts)
	interruption := make(chan string, 1)
	interruption <- "bp"
	ctx := RunContext{
		Config:          &Config{Name: "test", Author: "test"},
		Profile:         "default",
		DotRegolithPath: dotRegolithPath,
		interruption:    interruption,
	}

	interrupted, err := rf.Run(ctx)
	if err != nil {
		t.Fatalf("rf.Run failed: %v", err)
	}
	if !interrupted {
		t.Fatal("Expected interrupted == true when interruption channel was populated")
	}
	assertFileExists(t, filepath.Join(workDir, "BP", "first.txt"))
	assertFileNotExists(t, filepath.Join(workDir, "BP", "second.txt"))
}
