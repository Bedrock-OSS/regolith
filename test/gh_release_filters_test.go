package test

import (
	"archive/zip"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

const (
	// ghReleaseFilterUrl is the repository with the gh-release-filter. The
	// filter writes "Downloaded with git" when it's downloaded with git and
	// "Downloaded with github release" when it's downloaded from the ZIP file
	// of its GitHub release. Only the 1.0.0 version has a GitHub release.
	ghReleaseFilterUrl = "github.com/Bedrock-OSS/regolith-test-filters"
	ghReleaseFilterId  = "gh-release-filter"
	// ghReleaseFilterOutput is the path of the file created by the filter,
	// relative to the project root.
	ghReleaseFilterOutput = "build/regolith_test_project_bp/gh_release_filter.txt"
)

// requireGhCli skips the test when GitHub CLI isn't installed or isn't
// logged in. On CI it fails instead, so the tests can't be skipped silently.
func requireGhCli(t *testing.T) {
	t.Helper()
	reason := ""
	if _, err := exec.LookPath("gh"); err != nil {
		reason = "GitHub CLI (gh) is not installed"
	} else if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		reason = "GitHub CLI (gh) is not logged in (on CI set GH_TOKEN)"
	}
	if reason == "" {
		return
	}
	if os.Getenv("CI") != "" {
		t.Fatal(reason)
	}
	t.Skip(reason)
}

// ghReleaseCacheZip returns the path where Regolith caches the ZIP file of
// the given version of the gh-release-filter.
func ghReleaseCacheZip(version string, t *testing.T) string {
	t.Helper()
	userCache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal("Failed to get user cache dir:", err)
	}
	return filepath.Join(
		userCache, "regolith", "gh-release-cache", "Bedrock-OSS",
		"regolith-test-filters",
		ghReleaseFilterId+"-"+version+".zip")
}

// installGhReleaseFilter forcefully installs the given version of the
// gh-release-filter.
func installGhReleaseFilter(version string, refreshFilters bool, t *testing.T) {
	t.Helper()
	err := regolith.Install(
		[]string{ghReleaseFilterUrl + "/" + ghReleaseFilterId + "==" + version},
		true,           // Force
		false,          // Refresh resolvers
		refreshFilters, // Refresh filters
		[]string{},     // Profiles
		true,           // Debug
		"",             // Env
	)
	if err != nil {
		t.Fatal("'regolith install' failed:", err)
	}
}

// runAndExpectGhReleaseFilterOutput runs the project and checks the content
// of the file created by the gh-release-filter.
func runAndExpectGhReleaseFilterOutput(expected string, t *testing.T) {
	t.Helper()
	err := regolith.Run("default", []string{}, true, "", false, false, true)
	if err != nil {
		t.Fatal("'regolith run' failed:", err)
	}
	content, err := os.ReadFile(ghReleaseFilterOutput)
	if err != nil {
		t.Fatal("Failed to read the output of the filter:", err)
	}
	if string(content) != expected {
		t.Fatalf("Unexpected filter output.\nExpected: %q\nActual: %q",
			expected, string(content))
	}
}

// fakeReleaseFiles returns the files of a copy of the filter that writes
// "Downloaded with <source>". The files are placed in the 'folder' inside of
// the ZIP file ("" for the root).
func fakeReleaseFiles(source, folder string) map[string]string {
	return map[string]string{
		folder + "filter.json": `{"filters": [{"runWith": "python", "script": "./main.py"}]}`,
		folder + "main.py": "from pathlib import Path\n" +
			"Path('BP/gh_release_filter.txt').write_text('Downloaded with " +
			source + "')\n",
	}
}

// writeFakeReleaseZip replaces the cached release ZIP with given files.
func writeFakeReleaseZip(zipPath string, files map[string]string, t *testing.T) {
	t.Helper()
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal("Failed to create the ZIP file:", err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	for name, content := range files {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatal("Failed to create a ZIP entry:", err)
		}
		entry.Write([]byte(content))
	}
	if err := w.Close(); err != nil {
		t.Fatal("Failed to write the ZIP file:", err)
	}
}

// TestGhReleaseFilter tests downloading remote filters from GitHub releases
// using GitHub CLI: downloading the release ZIP, reusing it from the cache,
// refreshing the cache, recovering from a broken cached file and falling
// back to git when the version has no release.
func TestGhReleaseFilter(t *testing.T) {
	requireGhCli(t)
	// Switch to current working directory at the end of the test
	defer os.Chdir(getWdOrFatal(t))
	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory("TestGhReleaseFilter", t)

	t.Log("Copying the project files into the testing directory...")
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(ghReleaseFilterPath, workingDir, t)
	os.Chdir(workingDir)

	cachedZip := ghReleaseCacheZip("1.0.0", t)
	if err := os.RemoveAll(cachedZip); err != nil {
		t.Fatal("Failed to remove the cached release file:", err)
	}

	// THE TEST
	t.Log("Installing a version with a GitHub release...")
	err := regolith.InstallAll(false, false, true, false, "")
	if err != nil {
		t.Fatal("'regolith install-all' failed:", err)
	}
	if _, err := os.Stat(cachedZip); err != nil {
		t.Fatal("The release file was not saved in the cache:", err)
	}
	runAndExpectGhReleaseFilterOutput("Downloaded with github release", t)

	t.Log("Installing the same version again should use the cached file...")
	writeFakeReleaseZip(cachedZip, fakeReleaseFiles("cache", ""), t)
	installGhReleaseFilter("1.0.0", false, t)
	runAndExpectGhReleaseFilterOutput("Downloaded with cache", t)

	t.Log("Refreshing filters should download the release file again...")
	installGhReleaseFilter("1.0.0", true, t)
	runAndExpectGhReleaseFilterOutput("Downloaded with github release", t)

	t.Log("A broken cached file should fall back to git and be removed...")
	if err := os.WriteFile(cachedZip, []byte("not a zip"), 0644); err != nil {
		t.Fatal("Failed to break the cached file:", err)
	}
	installGhReleaseFilter("1.0.0", false, t)
	runAndExpectGhReleaseFilterOutput("Downloaded with git", t)
	if _, err := os.Stat(cachedZip); !os.IsNotExist(err) {
		t.Fatal("The broken cached release file was not removed:", err)
	}

	t.Log("A ZIP without filter.json in its root should fall back to git...")
	writeFakeReleaseZip(cachedZip, fakeReleaseFiles("cache", "gh-release-filter/"), t)
	installGhReleaseFilter("1.0.0", false, t)
	runAndExpectGhReleaseFilterOutput("Downloaded with git", t)
	if _, err := os.Stat(cachedZip); !os.IsNotExist(err) {
		t.Fatal("The cached release file without filter.json was not removed:", err)
	}

	t.Log("A version without a GitHub release should fall back to git...")
	installGhReleaseFilter("1.0.1", false, t)
	runAndExpectGhReleaseFilterOutput("Downloaded with git", t)
}
