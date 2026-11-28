package regolith

import (
	"archive/zip"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
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

func runTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	baseArgs := []string{
		"-c", "user.name=test",
		"-c", "user.email=test@example.com",
		"-c", "commit.gpgsign=false",
		"-c", "tag.gpgsign=false",
		"-c", "core.autocrlf=false",
	}
	cmd := exec.Command("git", append(baseArgs, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed in %s: %v\n%s", strings.Join(args, " "), dir, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func initTestFilterRepo(t *testing.T, filterId, filterJsonContent string) (bareRepo, workRepo, headSha string) {
	t.Helper()
	if !hasGit() {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	bareRepo = filepath.Join(root, "remote.git")
	workRepo = filepath.Join(root, "work")
	if err := os.MkdirAll(bareRepo, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workRepo, 0755); err != nil {
		t.Fatal(err)
	}

	runTestGit(t, bareRepo, "init", "--bare")
	runTestGit(t, bareRepo, "symbolic-ref", "HEAD", "refs/heads/main")

	runTestGit(t, workRepo, "init")
	runTestGit(t, workRepo, "checkout", "-b", "main")
	runTestGit(t, workRepo, "remote", "add", "origin", bareRepo)

	filterDir := filepath.Join(workRepo, filterId)
	if err := os.MkdirAll(filterDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filterDir, "filter.json"), []byte(filterJsonContent), 0644); err != nil {
		t.Fatal(err)
	}

	runTestGit(t, workRepo, "add", ".")
	runTestGit(t, workRepo, "commit", "-m", "initial commit")
	runTestGit(t, workRepo, "push", "-u", "origin", "main")
	headSha = runTestGit(t, workRepo, "rev-parse", "HEAD")
	return bareRepo, workRepo, headSha
}

func TestDownloadFilterRepository_ForcePushedCommitWithinCooldown(t *testing.T) {
	v1 := `{"filters":[],"version":"v1"}`
	bareRepo, workRepo, sha1 := initTestFilterRepo(t, "myfilter", v1)

	cache := filepath.Join(t.TempDir(), "cache")
	dst1 := filepath.Join(t.TempDir(), "dst1")
	if err := downloadFilterRepositoryFromURL(bareRepo, "myfilter", dst1, sha1, false, cache, time.Hour); err != nil {
		t.Fatalf("initial download failed: %v", err)
	}
	got1, err := os.ReadFile(filepath.Join(dst1, "filter.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got1) != v1 {
		t.Fatalf("got %q, want %q", string(got1), v1)
	}

	// Force-push an amended commit to remote within cooldown window.
	v2 := `{"filters":[],"version":"v2"}`
	if err := os.WriteFile(filepath.Join(workRepo, "myfilter", "filter.json"), []byte(v2), 0644); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, workRepo, "add", ".")
	runTestGit(t, workRepo, "commit", "--amend", "-m", "amended commit")
	runTestGit(t, workRepo, "push", "--force", "origin", "main")
	sha2 := runTestGit(t, workRepo, "rev-parse", "HEAD")
	if sha2 == sha1 {
		t.Fatalf("expected amended commit SHA to differ from %s", sha1)
	}

	dst2 := filepath.Join(t.TempDir(), "dst2")
	if err := downloadFilterRepositoryFromURL(bareRepo, "myfilter", dst2, sha2, false, cache, time.Hour); err != nil {
		t.Fatalf("recovery after force push failed: %v", err)
	}
	got2, err := os.ReadFile(filepath.Join(dst2, "filter.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got2) != v2 {
		t.Fatalf("got %q, want %q", string(got2), v2)
	}
}

func TestDownloadFilterRepository_MovedTag(t *testing.T) {
	v1 := `{"filters":[],"version":"tag-v1"}`
	bareRepo, workRepo, _ := initTestFilterRepo(t, "myfilter", v1)
	runTestGit(t, workRepo, "tag", "myfilter-1.0.0")
	runTestGit(t, workRepo, "push", "origin", "refs/tags/myfilter-1.0.0")

	cache := filepath.Join(t.TempDir(), "cache")
	dst1 := filepath.Join(t.TempDir(), "dst1")
	if err := downloadFilterRepositoryFromURL(bareRepo, "myfilter", dst1, "myfilter-1.0.0", false, cache, time.Hour); err != nil {
		t.Fatalf("initial tag download failed: %v", err)
	}
	got1, err := os.ReadFile(filepath.Join(dst1, "filter.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got1) != v1 {
		t.Fatalf("got %q, want %q", string(got1), v1)
	}

	// Move tag myfilter-1.0.0 to a new commit and force-push it.
	v2 := `{"filters":[],"version":"tag-v2"}`
	if err := os.WriteFile(filepath.Join(workRepo, "myfilter", "filter.json"), []byte(v2), 0644); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, workRepo, "add", ".")
	runTestGit(t, workRepo, "commit", "-m", "move tag commit")
	runTestGit(t, workRepo, "tag", "-f", "myfilter-1.0.0")
	runTestGit(t, workRepo, "push", "origin", "main")
	runTestGit(t, workRepo, "push", "--force", "origin", "refs/tags/myfilter-1.0.0")

	dst2 := filepath.Join(t.TempDir(), "dst2")
	if err := downloadFilterRepositoryFromURL(bareRepo, "myfilter", dst2, "myfilter-1.0.0", true, cache, time.Hour); err != nil {
		t.Fatalf("moved tag download failed: %v", err)
	}
	got2, err := os.ReadFile(filepath.Join(dst2, "filter.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got2) != v2 {
		t.Fatalf("got %q, want %q", string(got2), v2)
	}
}

func TestDownloadFilterRepository_CorruptedCache(t *testing.T) {
	v1 := `{"filters":[],"version":"healthy"}`
	bareRepo, _, sha1 := initTestFilterRepo(t, "myfilter", v1)

	cache := filepath.Join(t.TempDir(), "cache")
	dst1 := filepath.Join(t.TempDir(), "dst1")
	if err := downloadFilterRepositoryFromURL(bareRepo, "myfilter", dst1, sha1, false, cache, time.Hour); err != nil {
		t.Fatalf("initial download failed: %v", err)
	}

	// Corrupt the cached git directory.
	if err := os.WriteFile(filepath.Join(cache, ".git", "HEAD"), []byte("corrupted-head\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(cache, ".git", "config")); err != nil {
		t.Fatal(err)
	}

	dst2 := filepath.Join(t.TempDir(), "dst2")
	if err := downloadFilterRepositoryFromURL(bareRepo, "myfilter", dst2, sha1, false, cache, time.Hour); err != nil {
		t.Fatalf("recovery from corrupted cache failed: %v", err)
	}
	got2, err := os.ReadFile(filepath.Join(dst2, "filter.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got2) != v1 {
		t.Fatalf("got %q, want %q", string(got2), v1)
	}
}

func TestDownloadFilterRepository_NonexistentRef(t *testing.T) {
	v1 := `{"filters":[],"version":"1.0.0"}`
	bareRepo, _, sha1 := initTestFilterRepo(t, "myfilter", v1)
	const missingRef = "nonexistent-ref-12345"

	assertDescriptiveError := func(t *testing.T, err error) {
		t.Helper()
		if err == nil {
			t.Fatal("expected error for nonexistent ref, got nil")
		}
		msg := err.Error()
		for _, want := range []string{bareRepo, missingRef, "[git]", "regolith clean --filter-cache"} {
			if !strings.Contains(msg, want) {
				t.Errorf("error message %q does not contain %q", msg, want)
			}
		}
	}

	t.Run("fresh cache", func(t *testing.T) {
		cache := filepath.Join(t.TempDir(), "cache")
		dst := filepath.Join(t.TempDir(), "dst")
		err := downloadFilterRepositoryFromURL(bareRepo, "myfilter", dst, missingRef, false, cache, time.Hour)
		assertDescriptiveError(t, err)
	})

	t.Run("populated cache within cooldown", func(t *testing.T) {
		cache := filepath.Join(t.TempDir(), "cache")
		dst1 := filepath.Join(t.TempDir(), "dst1")
		if err := downloadFilterRepositoryFromURL(bareRepo, "myfilter", dst1, sha1, false, cache, time.Hour); err != nil {
			t.Fatalf("initial download failed: %v", err)
		}
		dst2 := filepath.Join(t.TempDir(), "dst2")
		err := downloadFilterRepositoryFromURL(bareRepo, "myfilter", dst2, missingRef, false, cache, time.Hour)
		assertDescriptiveError(t, err)
	})
}

func TestDownloadFilterRepository_FailedCloneCleansUpCache(t *testing.T) {
	if !hasGit() {
		t.Skip("git is not installed")
	}
	invalidRepo := filepath.Join(t.TempDir(), "does-not-exist.git")
	cache := filepath.Join(t.TempDir(), "cache")
	dst := filepath.Join(t.TempDir(), "dst")

	err := downloadFilterRepositoryFromURL(invalidRepo, "myfilter", dst, "main", false, cache, time.Hour)
	if err == nil {
		t.Fatal("expected clone error for invalid repository path")
	}
	msg := err.Error()
	for _, want := range []string{invalidRepo, "[git]"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message %q does not contain %q", msg, want)
		}
	}
	if _, statErr := os.Stat(cache); !os.IsNotExist(statErr) {
		t.Fatalf("expected failed clone cache directory %s to be removed, stat err = %v", cache, statErr)
	}
}

func TestRunGitProcess_CapturesOutput(t *testing.T) {
	if !hasGit() {
		t.Skip("git is not installed")
	}
	out, err := RunGitProcess([]string{"--version"}, t.TempDir())
	if err != nil {
		t.Fatalf("git --version failed: %v", err)
	}
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "[git]") || !strings.Contains(joined, "git version") {
		t.Fatalf("unexpected git --version output: %q", joined)
	}

	out, err = RunGitProcess([]string{"rev-parse", "--verify", "nonexistent"}, t.TempDir())
	if err == nil {
		t.Fatal("expected error from failing git command")
	}
	joined = strings.Join(out, "\n")
	if len(out) == 0 || !strings.Contains(joined, "[git]") {
		t.Fatalf("expected captured [git] error output, got %v", out)
	}
}
