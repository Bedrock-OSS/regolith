package test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

const (
	// npmFilterPath contains the files for testing filters installed from
	// npm registries: 'package' is the source of the package that the fake
	// registry serves, 'project' is a project that uses it with an exact
	// version, and 'project_after_run' is the expected state of the project
	// after 'regolith install-all' and 'regolith run'.
	npmFilterPath = "testdata/npm_filter"

	// npmFilterPackageName is the name of the package in the 'package'
	// directory. It's scoped, so the tests cover the URL encoding of the
	// scope in registry requests.
	npmFilterPackageName = "@regolith-test/npm-filter"

	// npmFilterId is the name of the filter in the project's config.json
	npmFilterId = "npm-filter"
)

// fakeNpmRegistry is an HTTP server that implements the two endpoints that
// "npm view" and "npm pack" use: the packument of a package and the tarballs
// of its versions. It records the requests it receives.
type fakeNpmRegistry struct {
	server   *httptest.Server
	name     string
	tarballs map[string][]byte // version -> tarball
	// requiredToken, when set, is the bearer token that every request must
	// carry; requests without it are answered with 401, as a private
	// registry answers them.
	requiredToken string
	mu            sync.Mutex
	requests      []registryRequest // in the order the requests arrived
}

// registryRequest is a request the fake registry received and the status
// it answered with (0 until the handler returns).
type registryRequest struct {
	req    *http.Request
	status int
}

// statusRecorder records the status code of a response.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// buildNpmTarball creates the tarball that "npm publish" would create from
// the package directory, with the version of package.json replaced.
func buildNpmTarball(t *testing.T, packageDir, version string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)
	err := filepath.WalkDir(packageDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(packageDir, path)
		if err != nil {
			return err
		}
		if relPath == "package.json" {
			var packageJson map[string]any
			if err := json.Unmarshal(content, &packageJson); err != nil {
				return err
			}
			packageJson["version"] = version
			content, _ = json.MarshalIndent(packageJson, "", "\t")
		}
		header := &tar.Header{
			Name:     "package/" + filepath.ToSlash(relPath),
			Mode:     0644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		_, err = tarWriter.Write(content)
		return err
	})
	if err != nil {
		t.Fatalf("Failed to build the package tarball: %v", err)
	}
	tarWriter.Close()
	gzWriter.Close()
	return buf.Bytes()
}

// startFakeNpmRegistry starts a registry that serves the package from the
// 'package' directory of npmFilterPath in the given versions.
func startFakeNpmRegistry(t *testing.T, versions ...string) *fakeNpmRegistry {
	t.Helper()
	packageDir := absOrFatal(filepath.Join(npmFilterPath, "package"), t)
	registry := &fakeNpmRegistry{
		name:     npmFilterPackageName,
		tarballs: map[string][]byte{},
	}
	for _, version := range versions {
		registry.tarballs[version] = buildNpmTarball(t, packageDir, version)
	}
	registry.server = httptest.NewServer(http.HandlerFunc(registry.handle))
	t.Cleanup(registry.server.Close)
	return registry
}

func (r *fakeNpmRegistry) tarballName(version string) string {
	baseName := r.name[strings.LastIndex(r.name, "/")+1:]
	return fmt.Sprintf("%s-%s.tgz", baseName, version)
}

func (r *fakeNpmRegistry) handle(rw http.ResponseWriter, req *http.Request) {
	w := &statusRecorder{ResponseWriter: rw, status: http.StatusOK}
	r.mu.Lock()
	r.requests = append(r.requests, registryRequest{req: req.Clone(req.Context())})
	index := len(r.requests) - 1
	requiredToken := r.requiredToken
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.requests[index].status = w.status
		r.mu.Unlock()
	}()
	if requiredToken != "" && req.Header.Get("Authorization") != "Bearer "+requiredToken {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
		return
	}
	// req.URL.Path has the "%2f" of the scope decoded
	if req.URL.Path == "/"+r.name {
		r.servePackument(w)
		return
	}
	if prefix := "/" + r.name + "/-/"; strings.HasPrefix(req.URL.Path, prefix) {
		for version, tarball := range r.tarballs {
			if req.URL.Path == prefix+r.tarballName(version) {
				// Tarballs are immutable and registries say so, but npm
				// still revalidates its cached copy with a conditional
				// request, which a registry answers with 304 and no body.
				sha1Sum := sha1.Sum(tarball)
				etag := `"` + hex.EncodeToString(sha1Sum[:]) + `"`
				w.Header().Set("Cache-Control", "public, max-age=31557600, immutable")
				w.Header().Set("ETag", etag)
				if req.Header.Get("If-None-Match") == etag {
					w.WriteHeader(http.StatusNotModified)
					return
				}
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Write(tarball)
				return
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error": "Not found"}`))
}

func (r *fakeNpmRegistry) servePackument(w http.ResponseWriter) {
	versions := map[string]any{}
	latest := ""
	for version, tarball := range r.tarballs {
		sha1Sum := sha1.Sum(tarball)
		sha512Sum := sha512.Sum512(tarball)
		versions[version] = map[string]any{
			"name":    r.name,
			"version": version,
			"dist": map[string]any{
				"tarball":   r.server.URL + "/" + r.name + "/-/" + r.tarballName(version),
				"shasum":    hex.EncodeToString(sha1Sum[:]),
				"integrity": "sha512-" + base64.StdEncoding.EncodeToString(sha512Sum[:]),
			},
		}
		// The versions of the tests are "1.x.0", so a string comparison
		// is enough to find the latest one.
		if version > latest {
			latest = version
		}
	}
	packument := map[string]any{
		"name":      r.name,
		"dist-tags": map[string]string{"latest": latest},
		"versions":  versions,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(packument)
}

// requestCount returns the number of requests the registry received.
func (r *fakeNpmRegistry) requestCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.requests)
}

// authorizations returns the Authorization headers of the requests the
// registry received, one per request.
func (r *fakeNpmRegistry) authorizations() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]string, 0, len(r.requests))
	for _, r := range r.requests {
		result = append(result, r.req.Header.Get("Authorization"))
	}
	return result
}

// setupNpmTestEnvironment points npm at the fake registry and both npm and
// Regolith at temporary directories, so the test touches neither the user's
// npm configuration nor the user's Regolith cache. It skips the test when
// npm isn't installed.
func setupNpmTestEnvironment(t *testing.T, registry *fakeNpmRegistry) {
	t.Helper()
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm is not installed; skipping the npm filter test")
	}
	tmpDir := t.TempDir()
	// npm
	t.Setenv("npm_config_registry", registry.server.URL)
	t.Setenv("npm_config_cache", filepath.Join(tmpDir, "npm-cache"))
	t.Setenv("npm_config_userconfig", filepath.Join(tmpDir, "npmrc"))
	t.Setenv("npm_config_globalconfig", filepath.Join(tmpDir, "npmrc-global"))
	t.Setenv("npm_config_update_notifier", "false")
	t.Setenv("npm_config_fund", "false")
	t.Setenv("npm_config_audit", "false")
	// Regolith's per-user caches (resolvers, Git filters)
	userCache := filepath.Join(tmpDir, "user-cache")
	t.Setenv("LOCALAPPDATA", userCache)
	t.Setenv("XDG_CACHE_HOME", userCache)
	if runtime.GOOS == "darwin" {
		t.Setenv("HOME", userCache)
	}
}

// tarballDownloadsSince returns the number of tarballs the registry served
// in full (a 304 to a revalidation doesn't count) after its first n
// requests.
func (r *fakeNpmRegistry) tarballDownloadsSince(n int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := 0
	for _, r := range r.requests[n:] {
		if strings.HasSuffix(r.req.URL.Path, ".tgz") && r.status == http.StatusOK {
			result++
		}
	}
	return result
}

// prepareNpmFilterProject copies the test project to a fresh testing
// directory, switches to it and returns its path. rootDir is the directory
// of the test package, which the relative test data paths are based on.
func prepareNpmFilterProject(t *testing.T, rootDir, testName string) string {
	t.Helper()
	os.Chdir(rootDir)
	tmpDir := prepareTestDirectory(testName, t)
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(filepath.Join(npmFilterPath, "project"), workingDir, t)
	os.Chdir(workingDir)
	return workingDir
}

// setNpmFilterVersion rewrites the version of the npm filter in the
// config.json of the working directory.
func setNpmFilterVersion(t *testing.T, version string) {
	t.Helper()
	configMap, err := regolith.LoadConfigAsMap()
	if err != nil {
		t.Fatal("Failed to load config.json:", err)
	}
	definition, err := regolith.FindByJSONPath[map[string]any](
		configMap, "regolith/filterDefinitions/"+npmFilterId)
	if err != nil {
		t.Fatal(err)
	}
	definition["version"] = version
	configJson, _ := json.MarshalIndent(configMap, "", "\t")
	if err := os.WriteFile(regolith.ConfigFilePath, configJson, 0644); err != nil {
		t.Fatal(err)
	}
}

// readInstalledNpmFilterVersion returns the version saved in the filter.json
// of the installed filter.
func readInstalledNpmFilterVersion(t *testing.T) string {
	t.Helper()
	filterJsonPath := filepath.Join(".regolith", "cache", "filters", npmFilterId, "filter.json")
	content, err := os.ReadFile(filterJsonPath)
	if err != nil {
		t.Fatalf("The filter isn't installed: %v", err)
	}
	var filterJson map[string]any
	if err := json.Unmarshal(content, &filterJson); err != nil {
		t.Fatal(err)
	}
	version, _ := filterJson["version"].(string)
	return version
}

// readNpmFilterMarker returns the marker file that the filter writes into
// the exported behavior pack.
func readNpmFilterMarker(t *testing.T) map[string]string {
	t.Helper()
	markerPath := filepath.Join("build", "npm_filter_test_project_bp", "npm_filter_marker.json")
	content, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("The filter didn't write its marker file: %v", err)
	}
	var marker map[string]string
	if err := json.Unmarshal(content, &marker); err != nil {
		t.Fatal(err)
	}
	return marker
}

// TestNpmFilterInstallAllAndRun installs a filter from an npm registry with
// 'regolith install-all', runs it with 'regolith run' and compares the
// project with the expected result.
func TestNpmFilterInstallAllAndRun(t *testing.T) {
	rootDir := getWdOrFatal(t)
	defer os.Chdir(rootDir)
	registry := startFakeNpmRegistry(t, "1.0.0")
	setupNpmTestEnvironment(t, registry)
	expectedPath := absOrFatal(filepath.Join(npmFilterPath, "project_after_run"), t)
	prepareNpmFilterProject(t, rootDir, "TestNpmFilterInstallAllAndRun")

	t.Log("Testing the 'regolith install-all' command...")
	err := regolith.InstallAll(false, false, true, false, "")
	if err != nil {
		t.Fatal("'regolith install-all' failed:", err)
	}
	t.Log("Testing the 'regolith run' command...")
	err = regolith.Run("default", []string{}, true, "", false, false, true)
	if err != nil {
		t.Fatal("'regolith run' failed:", err)
	}

	t.Log("Evaluating the test results...")
	// The subfilter's "npm i" writes a lock file whose format depends on
	// the npm version, and some versions create a node_modules directory
	// for a package without dependencies, so both are left out of the
	// comparison.
	filterDir := filepath.Join(".regolith", "cache", "filters", npmFilterId)
	os.Remove(filepath.Join(filterDir, "package-lock.json"))
	os.RemoveAll(filepath.Join(filterDir, "node_modules"))
	comparePaths(expectedPath, ".", t)
	if registry.requestCount() == 0 {
		t.Fatal("npm didn't contact the registry")
	}
	for _, authorization := range registry.authorizations() {
		if authorization != "" {
			t.Fatal("npm sent credentials to a registry that has none configured")
		}
	}
}

// TestNpmFilterVersionResolution installs the filter with a dist-tag and
// with a range, which resolve to the highest published version, and checks
// that 'regolith run' accepts the installed version for them.
func TestNpmFilterVersionResolution(t *testing.T) {
	rootDir := getWdOrFatal(t)
	defer os.Chdir(rootDir)
	registry := startFakeNpmRegistry(t, "1.0.0", "1.1.0")
	setupNpmTestEnvironment(t, registry)
	for _, spec := range []string{"latest", "^1.0.0", "1.x"} {
		t.Run(spec, func(t *testing.T) {
			prepareNpmFilterProject(t, rootDir, "TestNpmFilterVersionResolution")
			setNpmFilterVersion(t, spec)
			err := regolith.InstallAll(false, false, true, false, "")
			if err != nil {
				t.Fatal("'regolith install-all' failed:", err)
			}
			if version := readInstalledNpmFilterVersion(t); version != "1.1.0" {
				t.Fatalf("Expected %q to install version 1.1.0, got %q", spec, version)
			}
			// A second install-all is a no-op: the installed version is
			// the one the specification resolves to.
			requests := registry.requestCount()
			err = regolith.InstallAll(false, false, true, false, "")
			if err != nil {
				t.Fatal("second 'regolith install-all' failed:", err)
			}
			// Resolving the specification contacts the registry, but
			// nothing is downloaded.
			if registry.tarballDownloadsSince(requests) != 0 {
				t.Fatal("The second install-all downloaded the package again")
			}
			err = regolith.Run("default", []string{}, true, "", false, false, true)
			if err != nil {
				t.Fatal("'regolith run' failed:", err)
			}
			if marker := readNpmFilterMarker(t); marker["version"] != "1.1.0" {
				t.Fatalf("Expected the filter of version 1.1.0 to run, got %q", marker["version"])
			}
		})
	}
	// An exact version that was published is installed as it is
	t.Run("exact", func(t *testing.T) {
		prepareNpmFilterProject(t, rootDir, "TestNpmFilterVersionResolution")
		err := regolith.InstallAll(false, false, true, false, "")
		if err != nil {
			t.Fatal("'regolith install-all' failed:", err)
		}
		if version := readInstalledNpmFilterVersion(t); version != "1.0.0" {
			t.Fatalf("Expected version 1.0.0, got %q", version)
		}
	})
	// Nothing satisfies a range above the published versions
	t.Run("no match", func(t *testing.T) {
		prepareNpmFilterProject(t, rootDir, "TestNpmFilterVersionResolution")
		setNpmFilterVersion(t, "^2.0.0")
		err := regolith.InstallAll(false, false, true, false, "")
		if err == nil {
			t.Fatal("'regolith install-all' should fail when no version matches")
		}
		if !strings.Contains(err.Error(), npmFilterPackageName) {
			t.Fatalf("The error should name the package:\n%v", err)
		}
	})
	// A version that doesn't exist on the registry
	t.Run("missing version", func(t *testing.T) {
		prepareNpmFilterProject(t, rootDir, "TestNpmFilterVersionResolution")
		setNpmFilterVersion(t, "9.9.9")
		err := regolith.InstallAll(false, false, true, false, "")
		if err == nil {
			t.Fatal("'regolith install-all' should fail for a missing version")
		}
		if !strings.Contains(err.Error(), npmFilterPackageName+"@9.9.9") {
			t.Fatalf("The error should name the package and version:\n%v", err)
		}
	})
}

// TestNpmFilterUpdate installs an exact version, then runs
// 'regolith install-all --update', which moves the config to the latest
// published version.
func TestNpmFilterUpdate(t *testing.T) {
	rootDir := getWdOrFatal(t)
	defer os.Chdir(rootDir)
	registry := startFakeNpmRegistry(t, "1.0.0", "1.1.0")
	setupNpmTestEnvironment(t, registry)
	prepareNpmFilterProject(t, rootDir, "TestNpmFilterUpdate")

	err := regolith.InstallAll(false, false, true, false, "")
	if err != nil {
		t.Fatal("'regolith install-all' failed:", err)
	}
	if version := readInstalledNpmFilterVersion(t); version != "1.0.0" {
		t.Fatalf("Expected version 1.0.0, got %q", version)
	}
	err = regolith.InstallAll(false, true, true, false, "")
	if err != nil {
		t.Fatal("'regolith install-all --update' failed:", err)
	}
	if version := readInstalledNpmFilterVersion(t); version != "1.1.0" {
		t.Fatalf("Expected version 1.1.0 after the update, got %q", version)
	}
	configMap, err := regolith.LoadConfigAsMap()
	if err != nil {
		t.Fatal(err)
	}
	definition, err := regolith.FindByJSONPath[map[string]any](
		configMap, "regolith/filterDefinitions/"+npmFilterId)
	if err != nil {
		t.Fatal(err)
	}
	if definition["npm"] != npmFilterPackageName || definition["version"] != "1.1.0" {
		t.Fatalf("The update should lock config.json to the new version, got %v", definition)
	}
	if _, ok := definition["url"]; ok {
		t.Fatal("The update must not add a \"url\" to the npm filter definition")
	}
}

// TestNpmFilterInstallCommand installs a filter with
// 'regolith install npm:<package>' and checks the config.json it writes.
func TestNpmFilterInstallCommand(t *testing.T) {
	defer os.Chdir(getWdOrFatal(t))
	registry := startFakeNpmRegistry(t, "1.0.0", "1.1.0")
	setupNpmTestEnvironment(t, registry)
	tmpDir := prepareTestDirectory("TestNpmFilterInstallCommand", t)
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(freshProjectPath, workingDir, t)
	os.Chdir(workingDir)

	// Without a version, the latest published version is locked in
	err := regolith.Install(
		[]string{"npm:" + npmFilterPackageName}, false, false, false,
		[]string{"default"}, true, "")
	if err != nil {
		t.Fatal("'regolith install' failed:", err)
	}
	configMap, err := regolith.LoadConfigAsMap()
	if err != nil {
		t.Fatal(err)
	}
	definition, err := regolith.FindByJSONPath[map[string]any](
		configMap, "regolith/filterDefinitions/"+npmFilterId)
	if err != nil {
		t.Fatal(err)
	}
	if definition["npm"] != npmFilterPackageName || definition["version"] != "1.1.0" {
		t.Fatalf("Unexpected filter definition: %v", definition)
	}
	if version := readInstalledNpmFilterVersion(t); version != "1.1.0" {
		t.Fatalf("Expected version 1.1.0, got %q", version)
	}
	// The filter was added to the profile
	filters, err := regolith.FindByJSONPath[[]any](
		configMap, "regolith/profiles/default/filters")
	if err != nil {
		t.Fatal(err)
	}
	if len(filters) != 1 || filters[0].(map[string]any)["filter"] != npmFilterId {
		t.Fatalf("The filter should be added to the profile, got %v", filters)
	}
	// Installing it again is refused without --force
	err = regolith.Install(
		[]string{"npm:" + npmFilterPackageName}, false, false, false, nil, true, "")
	if err == nil {
		t.Fatal("'regolith install' should refuse to install an installed filter")
	}
	// With --force and a version, the version is kept as written
	err = regolith.Install(
		[]string{"npm:" + npmFilterPackageName + "==1.0.0"}, true, false, false,
		nil, true, "")
	if err != nil {
		t.Fatal("'regolith install --force' failed:", err)
	}
	if version := readInstalledNpmFilterVersion(t); version != "1.0.0" {
		t.Fatalf("Expected version 1.0.0, got %q", version)
	}
}

// TestNpmFilterAuth runs against a registry that demands a token. Without
// the token, the install fails with npm's 401 and Regolith's hint about the
// .npmrc files; with the token in the project's .npmrc, npm sends it on
// every request. Regolith has no part in the credentials: this proves that
// the project's .npmrc applies to the download.
func TestNpmFilterAuth(t *testing.T) {
	rootDir := getWdOrFatal(t)
	defer os.Chdir(rootDir)
	registry := startFakeNpmRegistry(t, "1.0.0")
	const token = "dummy-token-for-the-test"
	registry.requiredToken = token
	setupNpmTestEnvironment(t, registry)
	prepareNpmFilterProject(t, rootDir, "TestNpmFilterAuth")

	// Without the token
	err := regolith.InstallAll(false, false, true, false, "")
	if err == nil {
		t.Fatal("'regolith install-all' should fail without the registry's token")
	}
	if !strings.Contains(err.Error(), "E401") ||
		!strings.Contains(err.Error(), "The registry refused the request") ||
		!strings.Contains(err.Error(), ".npmrc") {
		t.Fatalf("The error should carry npm's E401 and the hint about .npmrc:\n%v", err)
	}
	if _, err := os.Stat(filepath.Join(".regolith", "cache", "filters", npmFilterId)); err == nil {
		t.Fatal("A failed download must not leave the filter behind")
	}
	// With the token in the project's .npmrc
	registryHost := strings.TrimPrefix(registry.server.URL, "http:")
	npmrc := fmt.Sprintf("registry=%s\n%s/:_authToken=%s\n",
		registry.server.URL, registryHost, token)
	if err := os.WriteFile(".npmrc", []byte(npmrc), 0644); err != nil {
		t.Fatal(err)
	}
	requests := registry.requestCount()
	err = regolith.InstallAll(false, false, true, false, "")
	if err != nil {
		t.Fatal("'regolith install-all' failed:", err)
	}
	authorizations := registry.authorizations()[requests:]
	if len(authorizations) == 0 {
		t.Fatal("npm didn't contact the registry")
	}
	for i, authorization := range authorizations {
		if authorization != "Bearer "+token {
			t.Fatalf("Request %d didn't carry the token of .npmrc: %q", i, authorization)
		}
	}
	if version := readInstalledNpmFilterVersion(t); version != "1.0.0" {
		t.Fatalf("Expected version 1.0.0, got %q", version)
	}
}

// TestNpmFilterNpmCache checks that the download goes through npm's own
// cache: a reinstall of an exact version downloads no tarball (npm
// revalidates its copy and the registry answers 304), --force-filter-refresh
// goes back to the registry, and npm's offline mode reinstalls the filter
// with the registry stopped. The default mode also falls back to the cache
// when the registry is unreachable, but only after npm's connection
// retries, which take minutes; the test doesn't wait for them.
func TestNpmFilterNpmCache(t *testing.T) {
	rootDir := getWdOrFatal(t)
	defer os.Chdir(rootDir)
	registry := startFakeNpmRegistry(t, "1.0.0")
	setupNpmTestEnvironment(t, registry)
	prepareNpmFilterProject(t, rootDir, "TestNpmFilterNpmCache")
	filterDir := filepath.Join(".regolith", "cache", "filters", npmFilterId)

	err := regolith.InstallAll(false, false, true, false, "")
	if err != nil {
		t.Fatal("'regolith install-all' failed:", err)
	}
	if registry.tarballDownloadsSince(0) != 1 {
		t.Fatalf("Expected one tarball download, got %d", registry.tarballDownloadsSince(0))
	}
	// A reinstall of an exact version takes the tarball from npm's cache
	requests := registry.requestCount()
	if err := os.RemoveAll(filterDir); err != nil {
		t.Fatal(err)
	}
	err = regolith.InstallAll(false, false, true, false, "")
	if err != nil {
		t.Fatal("'regolith install-all' failed to reinstall the filter:", err)
	}
	if registry.tarballDownloadsSince(requests) != 0 {
		t.Fatal("The reinstall downloaded the tarball again")
	}
	if version := readInstalledNpmFilterVersion(t); version != "1.0.0" {
		t.Fatalf("Expected version 1.0.0, got %q", version)
	}
	// A forced refresh asks the registry again even when npm's cache
	// holds the package
	requests = registry.requestCount()
	err = regolith.InstallAll(true, false, true, true, "")
	if err != nil {
		t.Fatal("'regolith install-all --force --force-filter-refresh' failed:", err)
	}
	if registry.requestCount() == requests {
		t.Fatal("The forced refresh didn't contact the registry")
	}
	// With the registry stopped, npm's offline mode reinstalls the filter
	// from npm's cache
	registry.server.Close()
	if err := os.RemoveAll(filterDir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("npm_config_offline", "true")
	err = regolith.InstallAll(false, false, true, false, "")
	if err != nil {
		t.Fatal("'regolith install-all' failed to reinstall offline from npm's cache:", err)
	}
	if version := readInstalledNpmFilterVersion(t); version != "1.0.0" {
		t.Fatalf("Expected version 1.0.0, got %q", version)
	}
}

// TestNpmFilterConfigParsing checks that a config.json with an npm filter
// definition parses into an NpmFilterDefinition.
func TestNpmFilterConfigParsing(t *testing.T) {
	rootDir := getWdOrFatal(t)
	defer os.Chdir(rootDir)
	prepareNpmFilterProject(t, rootDir, "TestNpmFilterConfigParsing")
	configMap, err := regolith.LoadConfigAsMap()
	if err != nil {
		t.Fatal("Failed to load config.json:", err)
	}
	config, err := regolith.ConfigFromObject(configMap)
	if err != nil {
		t.Fatal("Failed to parse config:", err)
	}
	filterDef, ok := config.FilterDefinitions[npmFilterId]
	if !ok {
		t.Fatalf("Filter %q not found in the parsed config", npmFilterId)
	}
	npmDef, ok := filterDef.(*regolith.NpmFilterDefinition)
	if !ok {
		t.Fatalf("Expected *NpmFilterDefinition, got %T", filterDef)
	}
	if npmDef.Package != npmFilterPackageName || npmDef.Version != "1.0.0" {
		t.Fatalf("Unexpected definition: %+v", npmDef)
	}
}
