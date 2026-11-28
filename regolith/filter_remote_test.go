package regolith

import (
	"archive/zip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func setTestUserConfig(t *testing.T, cfg *UserConfig) {
	t.Helper()
	prev := cachedCombinedUserConfig
	cachedCombinedUserConfig = cfg
	t.Cleanup(func() {
		cachedCombinedUserConfig = prev
	})
}

func isolateTestCache(t *testing.T) string {
	t.Helper()
	userCache := t.TempDir()
	t.Setenv("LOCALAPPDATA", userCache)
	t.Setenv("XDG_CACHE_HOME", userCache)
	if runtime.GOOS == "darwin" {
		t.Setenv("HOME", userCache)
	}
	return userCache
}

func setOfflinePipEnv(t *testing.T) {
	t.Helper()
	t.Setenv("PIP_NO_INDEX", "1")
	t.Setenv("PIP_DISABLE_PIP_VERSION_CHECK", "1")
}

func fileURL(path string) string {
	slash := filepath.ToSlash(path)
	if !strings.HasPrefix(slash, "/") {
		slash = "/" + slash
	}
	return "file://" + slash
}

func runGitTestCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func createTestBareFilterRepo(t *testing.T, filterId, version string, files map[string]string) string {
	t.Helper()
	if !hasGit() {
		t.Skip("git is not installed")
	}
	workDir := filepath.Join(t.TempDir(), "work")
	for relPath, content := range files {
		fullPath := filepath.Join(workDir, filterId, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	runGitTestCmd(t, workDir, "init")
	runGitTestCmd(t, workDir, "add", ".")
	runGitTestCmd(t, workDir, "-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false", "commit", "-m", "init")
	runGitTestCmd(t, workDir, "-c", "tag.gpgsign=false", "tag", filterId+"-"+version)

	bareDir := filepath.Join(t.TempDir(), "repo.git")
	runGitTestCmd(t, filepath.Dir(bareDir), "clone", "--bare", workDir, bareDir)
	return bareDir
}

func TestRemoteFilterUpdate_CleansUpOnFailedInstallAndRecovers(t *testing.T) {
	isolateTestCache(t)
	setOfflinePipEnv(t)

	const filterId = "py-remote"
	const repoURL = "example.invalid/test-filters"
	bareDir := createTestBareFilterRepo(t, filterId, "1.0.0", map[string]string{
		"filter.json":      `{"filters": [{"runWith": "python", "script": "main.py"}]}`,
		"main.py":          "print('hello')\n",
		"requirements.txt": "",
		"data/default.txt": "default data\n",
	})

	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "url."+fileURL(bareDir)+".insteadOf")
	t.Setenv("GIT_CONFIG_VALUE_0", "https://"+repoURL)

	projectDir := t.TempDir()
	dotRegolithPath := filepath.Join(projectDir, ".regolith")
	dataPath := filepath.Join(projectDir, "packs", "data")
	def := &RemoteFilterDefinition{
		FilterDefinition: FilterDefinition{Id: filterId},
		Url:              repoURL,
		Version:          "1.0.0",
	}

	missingPython := filepath.Join(t.TempDir(), "nonexistent-python"+exeSuffix)
	badCfg := NewUserConfig()
	badCfg.fillDefaults()
	badCfg.PythonRunner = &missingPython
	setTestUserConfig(t, badCfg)

	err := def.Update(false, dotRegolithPath, dataPath, false)
	if err == nil {
		t.Fatal("expected Update to fail when Python is missing")
	}
	if !strings.Contains(err.Error(), "Python not found") {
		t.Fatalf("expected Python not found error, got: %v", err)
	}
	if _, statErr := os.Stat(def.GetDownloadPath(dotRegolithPath)); !os.IsNotExist(statErr) {
		t.Fatalf("expected failed install to remove cached filter dir %s, statErr=%v", def.GetDownloadPath(dotRegolithPath), statErr)
	}
	if _, verErr := def.InstalledVersion(dotRegolithPath); verErr == nil {
		t.Fatal("expected InstalledVersion to fail after failed install cleanup")
	}
	venvPath := filepath.Join(dotRegolithPath, "cache", "venvs", "0")
	if _, venvErr := os.Stat(venvPath); !os.IsNotExist(venvErr) {
		t.Fatalf("expected no broken venv at %s, statErr=%v", venvPath, venvErr)
	}
	if _, dataErr := os.Stat(filepath.Join(dataPath, filterId)); !os.IsNotExist(dataErr) {
		t.Fatalf("expected filter data not to be copied on failed install, statErr=%v", dataErr)
	}

	// Restore valid Python and retry without force: should install cleanly.
	goodCfg := NewUserConfig()
	goodCfg.fillDefaults()
	cachedCombinedUserConfig = goodCfg
	if _, err := findPython(); err != nil {
		t.Skipf("python not available for recovery check: %v", err)
	}

	if err := def.Update(false, dotRegolithPath, dataPath, false); err != nil {
		t.Fatalf("expected recovery Update(false) to succeed, got: %v", err)
	}
	gotVer, err := def.InstalledVersion(dotRegolithPath)
	if err != nil {
		t.Fatalf("InstalledVersion failed after recovery: %v", err)
	}
	if gotVer != "1.0.0" {
		t.Fatalf("expected installed version %q, got %q", "1.0.0", gotVer)
	}
	venvPython := filepath.Join(venvPath, venvScriptsPath, "python"+exeSuffix)
	if _, err := os.Stat(venvPython); err != nil {
		t.Fatalf("expected venv python at %s: %v", venvPython, err)
	}
	if _, err := os.Stat(filepath.Join(dataPath, filterId, "default.txt")); err != nil {
		t.Fatalf("expected filter data to be copied after recovery: %v", err)
	}
}

func TestRemoteFilterDownload_SkipsWhenAlreadyCached(t *testing.T) {
	dotRegolithPath := filepath.Join(t.TempDir(), ".regolith")
	def := &RemoteFilterDefinition{
		FilterDefinition: FilterDefinition{Id: "cached-filter"},
		Url:              "example.invalid/unused",
		Version:          "1.0.0",
	}
	downloadPath := def.GetDownloadPath(dotRegolithPath)
	if err := os.MkdirAll(downloadPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(downloadPath, "filter.json"), []byte(`{"filters": [], "version": "0.9.0"}`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := def.Download(false, dotRegolithPath, false); err != nil {
		t.Fatalf("Download(false) returned unexpected error: %v", err)
	}
	gotVer, err := def.InstalledVersion(dotRegolithPath)
	if err != nil {
		t.Fatalf("InstalledVersion failed: %v", err)
	}
	if gotVer != "0.9.0" {
		t.Fatalf("expected cached version %q to stay untouched, got %q", "0.9.0", gotVer)
	}
}
