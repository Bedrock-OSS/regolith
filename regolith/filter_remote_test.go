package regolith

import (
	"archive/zip"
	"os"
	"path/filepath"
	"slices"
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
