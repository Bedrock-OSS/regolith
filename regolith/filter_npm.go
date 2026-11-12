package regolith

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"golang.org/x/mod/semver"
)

// exactNpmVersionRegexp matches a full semantic version (major, minor and
// patch, with optional prerelease and build metadata). Anything else that
// npm accepts as a version ("latest", "^1.0.0", "1.x", "beta", ...) selects a
// version from the registry and has to be resolved before the download.
var exactNpmVersionRegexp = regexp.MustCompile(
	`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

// NpmFilterDefinition is the definition of a remote filter published to an
// npm registry instead of a Git repository. The package is downloaded with
// "npm pack" and extracted into the project's filter cache, where it's
// handled exactly like a Git remote filter: the runner, the subfilter
// dependency installation, the data folder and the version check in
// "regolith run" are all shared with RemoteFilterDefinition.
type NpmFilterDefinition struct {
	FilterDefinition
	// Package is the name of the npm package (for example "@scope/name").
	Package string `json:"npm,omitempty"`
	// Version is the version specification from the config file: an exact
	// version, a semver range or a dist-tag like "latest".
	Version string `json:"version,omitempty"`
	// NpmFilters can propagate some of the properties unique to other types
	// of filters (like Python's venvSlot).
	VenvSlot int `json:"venvSlot,omitempty"`
}

// NpmFilterDefinitionFromObject creates an NpmFilterDefinition from a filter
// definition object of the config file. The object must have the "npm" and
// "version" properties and must not have the "url" property.
func NpmFilterDefinitionFromObject(id string, obj map[string]any) (*NpmFilterDefinition, error) {
	result := &NpmFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
	npmPackage, ok := obj["npm"].(string)
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyTypeError, "npm", "string")
	}
	if npmPackage == "" {
		return nil, burrito.WrappedErrorf(
			"The \"npm\" property can't be an empty string.")
	}
	if _, ok := obj["url"]; ok {
		return nil, burrito.WrappedErrorf(npmFilterUrlConflictError, id)
	}
	result.Package = npmPackage
	versionObj, ok := obj["version"]
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "version")
	}
	version, ok := versionObj.(string)
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyTypeError, "version", "string")
	}
	if version == "HEAD" {
		return nil, burrito.WrappedErrorf(npmFilterHeadVersionError, id)
	}
	if version == "" {
		return nil, burrito.WrappedErrorf(
			"The \"version\" property can't be an empty string.")
	}
	result.Version = version
	venvSlot64, _ := obj["venvSlot"].(float64) // default venvSlot is 0.0
	result.VenvSlot = int(venvSlot64)
	return result, nil
}

// NpmFilterDefinitionFromTheRegistry creates an NpmFilterDefinition for the
// "regolith install npm:<package>" command. An empty version locks the
// definition to the exact version that "latest" resolves to at the time of
// the call; any other version specification is kept as written.
func NpmFilterDefinitionFromTheRegistry(
	npmPackage, name, version string,
) (*NpmFilterDefinition, error) {
	result := &NpmFilterDefinition{
		FilterDefinition: FilterDefinition{Id: name},
		Package:          npmPackage,
		Version:          version,
	}
	if version == "HEAD" {
		return nil, burrito.WrappedErrorf(npmFilterHeadVersionError, name)
	}
	if version == "" {
		resolved, err := resolveNpmVersion(npmPackage, "latest")
		if err != nil {
			return nil, burrito.PassError(err)
		}
		result.Version = resolved
	}
	return result, nil
}

// isExactNpmVersion returns whether version is a single semantic version, as
// opposed to a range or a dist-tag that has to be resolved on the registry.
func isExactNpmVersion(version string) bool {
	return exactNpmVersionRegexp.MatchString(version)
}

// remoteDefinition returns the RemoteFilterDefinition that handles the
// filter once it's in the project cache. Everything that happens after the
// download (version info, data folder, subfilter dependencies, checks and
// the runner) only needs the Id, the Version and the VenvSlot; the Url
// names the package in log messages and is never used for a download.
func (f *NpmFilterDefinition) remoteDefinition() *RemoteFilterDefinition {
	version := f.Version
	if !isExactNpmVersion(version) {
		// A range or a dist-tag doesn't name the version in the cache, so
		// the version check of "regolith run" can't compare them. The
		// check is skipped for them, the same way it is for "latest".
		version = "latest"
	}
	return &RemoteFilterDefinition{
		FilterDefinition: f.FilterDefinition,
		Url:              "npm:" + f.Package,
		Version:          version,
		VenvSlot:         f.VenvSlot,
	}
}

// GetDownloadPath returns the path location where the filter can be found.
func (f *NpmFilterDefinition) GetDownloadPath(dotRegolithPath string) string {
	return f.remoteDefinition().GetDownloadPath(dotRegolithPath)
}

func (f *NpmFilterDefinition) Uninstall(dotRegolithPath string) {
	f.remoteDefinition().Uninstall(dotRegolithPath)
}

// SaveVersionInfo saves the specified version string into the filter.json of
// the filter.
func (f *NpmFilterDefinition) SaveVersionInfo(version, dotRegolithPath string) error {
	return f.remoteDefinition().SaveVersionInfo(version, dotRegolithPath)
}

// InstalledVersion reads the version saved in the filter.json
func (f *NpmFilterDefinition) InstalledVersion(dotRegolithPath string) (string, error) {
	return f.remoteDefinition().InstalledVersion(dotRegolithPath)
}

// CopyFilterData copies the filter's data to the data folder.
func (f *NpmFilterDefinition) CopyFilterData(dataPath string, dotRegolithPath string) {
	f.remoteDefinition().CopyFilterData(dataPath, dotRegolithPath)
}

func (f *NpmFilterDefinition) InstallDependencies(_ *RemoteFilterDefinition, dotRegolithPath string) error {
	return f.remoteDefinition().InstallDependencies(nil, dotRegolithPath)
}

func (f *NpmFilterDefinition) Check(context RunContext) error {
	return f.remoteDefinition().Check(context)
}

func (f *NpmFilterDefinition) CreateFilterRunner(runConfiguration map[string]any, id string) (FilterRunner, error) {
	return f.remoteDefinition().CreateFilterRunner(runConfiguration, id)
}

// ResolveVersion returns the exact version of the package selected by the
// version specification of the definition. Exact versions are returned as
// they are, without contacting the registry.
func (f *NpmFilterDefinition) ResolveVersion() (string, error) {
	return resolveNpmVersion(f.Package, f.Version)
}

// Download downloads the package into the project cache. The tarball comes
// from npm's own cache when npm has it, as it would for "npm install": that
// cache is keyed by the registry that served the package and checked against
// the integrity hash of its packument, so two projects whose .npmrc files map
// the same package name to different registries never share a download.
// refreshFilters makes npm revalidate its cache against the registry.
func (f *NpmFilterDefinition) Download(
	isForced bool, dotRegolithPath string, refreshFilters bool,
) error {
	downloadPath := f.GetDownloadPath(dotRegolithPath)
	if _, err := os.Stat(downloadPath); err == nil {
		if !isForced {
			Logger.Warnf(
				"The download path of the \"%s\" already exists.This should "+
					"be the case only if the filter is installed.\n"+
					"    Skipped the download. You can force the it by "+
					"passing the \"-force\" flag.", f.Id)
			return nil
		} else {
			f.Uninstall(dotRegolithPath)
		}
	}

	Logger.Infof("Downloading filter %s...", f.Id)

	MeasureStart("Resolve npm package version")
	version, err := f.ResolveVersion()
	if err != nil {
		return burrito.PassError(err)
	}
	MeasureEnd()
	err = downloadNpmPackage(f.Package, version, downloadPath, refreshFilters)
	if err != nil {
		return burrito.PassError(err)
	}
	// Save the version of the filter we downloaded
	MeasureStart("Save version info")
	err = f.SaveVersionInfo(version, dotRegolithPath)
	if err != nil {
		return burrito.PassError(err)
	}
	MeasureEnd()
	// Remove 'test' folder, which we never want to use (saves space on disk)
	testFolder := path.Join(downloadPath, "test")
	if _, err := os.Stat(testFolder); err == nil {
		os.RemoveAll(testFolder)
	}
	Logger.Infof("Filter \"%s\" downloaded successfully.", f.Id)
	return nil
}

func (f *NpmFilterDefinition) Update(force bool, dotRegolithPath, dataPath string, refreshFilters bool) error {
	installedVersion, err := f.InstalledVersion(dotRegolithPath)
	if err != nil && force {
		Logger.Warnf("Unable to get installed version of filter %q.", f.Id)
	}
	MeasureStart("Resolve npm package version")
	version, err := f.ResolveVersion()
	if err != nil {
		return burrito.PassError(err)
	}
	MeasureEnd()
	if installedVersion != version || force {
		Logger.Infof(
			"Updating filter %q to new version: %q->%q.",
			f.Id, installedVersion, version)
		err = f.Download(true, dotRegolithPath, refreshFilters)
		if err != nil {
			return burrito.PassError(err)
		}
		// Copy the data of the remote filter to the data path
		f.CopyFilterData(dataPath, dotRegolithPath)
		err = f.InstallDependencies(nil, dotRegolithPath)
		if err != nil {
			return burrito.PassError(err)
		}
		Logger.Infof("Filter %q updated successfully.", f.Id)
	} else {
		Logger.Infof(
			"Filter %q is up to date. Installed version: %q.",
			f.Id, installedVersion)
	}
	return nil
}

// npmCommand returns a command that runs npm with the given arguments in the
// current working directory (the project root), so that npm applies the
// project's .npmrc on top of the user's. Regolith doesn't handle registry
// credentials itself; they belong to npm's configuration.
func npmCommand(args ...string) (*exec.Cmd, error) {
	npmRunner, err := getRunner("npm", "npm")
	if err != nil {
		return nil, burrito.WrapError(err, getRunnerError)
	}
	npmPath, err := exec.LookPath(npmRunner)
	if err != nil {
		return nil, burrito.WrapErrorf(err, npmNotInstalledError, npmRunner)
	}
	cmd, err := batchFileCommand(npmPath, args...)
	if err != nil {
		return nil, burrito.PassError(err)
	}
	cmd.Dir, err = os.Getwd()
	if err != nil {
		return nil, burrito.WrapError(err, osGetwdError)
	}
	return cmd, nil
}

// runNpmCommand runs npm with the given arguments and returns its standard
// output. On failure, the error carries npm's standard error output and a
// hint about its most common causes.
func runNpmCommand(args ...string) ([]byte, error) {
	cmd, err := npmCommand(args...)
	if err != nil {
		return nil, burrito.PassError(err)
	}
	Logger.Debugf("Exec: npm %s", strings.Join(args, " "))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return nil, burrito.WrapErrorf(
			err, npmCommandError,
			"npm "+strings.Join(args, " "),
			strings.TrimSpace(stderr.String()),
			npmErrorHint(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// npmErrorHint returns a hint about the cause of a failed npm command based
// on the error codes in its standard error output.
func npmErrorHint(stderr string) string {
	switch {
	case strings.Contains(stderr, "E404"), strings.Contains(stderr, "ETARGET"):
		return "The package or the version doesn't exist on the registry " +
			"that npm uses for it.\nCheck the package name and the " +
			"registry configured for its scope in the .npmrc file."
	case strings.Contains(stderr, "E401"), strings.Contains(stderr, "E403"),
		strings.Contains(stderr, "ENEEDAUTH"):
		return "The registry refused the request.\nCheck the " +
			"authentication token configured for the registry in the " +
			".npmrc file (of the project or of the user)."
	case strings.Contains(stderr, "ENOTFOUND"),
		strings.Contains(stderr, "ECONNREFUSED"),
		strings.Contains(stderr, "ETIMEDOUT"),
		strings.Contains(stderr, "EAI_AGAIN"):
		return "npm couldn't reach the registry.\nCheck the registry URL " +
			"in the .npmrc file and the network connection."
	}
	return "npm failed to run the command. See its output above."
}

// resolveNpmVersion returns the highest version of the package that
// satisfies the version specification (an exact version, a semver range or a
// dist-tag). Exact versions are returned as they are.
func resolveNpmVersion(npmPackage, version string) (string, error) {
	if isExactNpmVersion(version) {
		return version, nil
	}
	spec := npmPackage + "@" + version
	output, err := runNpmCommand("view", spec, "version", "--json")
	if err != nil {
		return "", burrito.WrapErrorf(err, npmResolveVersionError, spec)
	}
	resolved, err := highestNpmViewVersion(output)
	if err != nil {
		return "", burrito.WrapErrorf(err, npmResolveVersionError, spec)
	}
	if resolved == "" {
		return "", burrito.WrappedErrorf(
			npmNoMatchingVersionError, npmPackage, version)
	}
	Logger.Debugf("Resolved %s to version %s", spec, resolved)
	return resolved, nil
}

// highestNpmViewVersion parses the output of "npm view <spec> version --json"
// and returns the highest version in it. The output is a JSON string when
// one version matches the specification, a JSON array of strings when
// several do, and empty when none does.
func highestNpmViewVersion(output []byte) (string, error) {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return "", nil
	}
	var single string
	if err := json.Unmarshal(trimmed, &single); err == nil {
		return single, nil
	}
	var versions []string
	if err := json.Unmarshal(trimmed, &versions); err != nil {
		return "", burrito.WrapErrorf(
			err, "Failed to parse the output of \"npm view\".\nOutput: %s",
			string(trimmed))
	}
	highest := ""
	for _, v := range versions {
		if highest == "" || semver.Compare("v"+v, "v"+highest) > 0 {
			highest = v
		}
	}
	return highest, nil
}

// downloadNpmPackage downloads the package of the given name and exact
// version with "npm pack" and extracts it to destPath. The tarball and the
// extraction go to a temporary sibling directory that is renamed to destPath
// only once the package is complete and verified, so an interrupted download
// never leaves a partial filter behind. preferOnline makes npm revalidate
// its cache against the registry even when it holds the package.
func downloadNpmPackage(npmPackage, version, destPath string, preferOnline bool) error {
	spec := npmPackage + "@" + version
	parentPath := filepath.Dir(destPath)
	err := os.MkdirAll(parentPath, 0755)
	if err != nil {
		return burrito.WrapErrorf(err, osMkdirError, parentPath)
	}
	tmpPath, err := os.MkdirTemp(parentPath, filepath.Base(destPath)+"-download-*")
	if err != nil {
		return burrito.WrapErrorf(err, osMkdirError, parentPath)
	}
	// If Regolith is killed before this deferred call runs, the directory
	// stays behind. Nothing lists the contents of cache/filters, and
	// "regolith clean" removes it.
	defer os.RemoveAll(tmpPath)
	packPath := filepath.Join(tmpPath, "pack")
	err = os.MkdirAll(packPath, 0755)
	if err != nil {
		return burrito.WrapErrorf(err, osMkdirError, packPath)
	}
	// Download the tarball. npm uses the .npmrc files for the registry and
	// its credentials, and its own cache for the tarball. The package's own
	// scripts never run here; they belong to the publisher's machine.
	args := []string{
		"pack", spec, "--json", "--ignore-scripts",
		"--pack-destination", packPath}
	if preferOnline {
		args = append(args, "--prefer-online")
	}
	MeasureStart("npm pack %s", spec)
	output, err := runNpmCommand(args...)
	if err != nil {
		return burrito.WrapErrorf(err, npmPackError, spec)
	}
	MeasureEnd()
	tarballPath, err := npmPackTarballPath(output, packPath)
	if err != nil {
		return burrito.WrapErrorf(err, npmPackError, spec)
	}
	// Extract the tarball
	MeasureStart("Extract %s", filepath.Base(tarballPath))
	extractPath := filepath.Join(tmpPath, "package")
	err = extractNpmTarball(tarballPath, extractPath)
	if err != nil {
		return burrito.WrapErrorf(err, npmPackError, spec)
	}
	MeasureEnd()
	if _, err := os.Stat(filepath.Join(extractPath, "filter.json")); err != nil {
		return burrito.WrappedErrorf(npmFilterMissingFilterJsonError, spec)
	}
	// Move the package into place
	err = os.RemoveAll(destPath)
	if err != nil {
		return burrito.WrapErrorf(err, osRemoveError, destPath)
	}
	err = os.Rename(extractPath, destPath)
	if err != nil {
		return burrito.WrapErrorf(err, osRenameError, extractPath, destPath)
	}
	return nil
}

// npmPackTarballPath returns the path of the tarball created by "npm pack
// --json --pack-destination <packPath>" from the command's output. When the
// output doesn't name a file that exists, it falls back to the only .tgz
// file in the destination directory.
func npmPackTarballPath(output []byte, packPath string) (string, error) {
	var packed []struct {
		Filename string `json:"filename"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(output), &packed); err == nil &&
		len(packed) == 1 && packed[0].Filename != "" {
		// Some npm versions report the file name of a scoped package as
		// "@scope/name-1.0.0.tgz" while writing "scope-name-1.0.0.tgz", so
		// the reported name is only trusted when the file is there.
		reported := filepath.Join(packPath, filepath.Base(packed[0].Filename))
		if _, err := os.Stat(reported); err == nil {
			return reported, nil
		}
	}
	tarballs, err := filepath.Glob(filepath.Join(packPath, "*.tgz"))
	if err != nil || len(tarballs) != 1 {
		return "", burrito.WrappedErrorf(
			"Unable to find the tarball created by \"npm pack\".\n"+
				"Output: %s", strings.TrimSpace(string(output)))
	}
	return tarballs[0], nil
}

// extractNpmTarball extracts a package tarball created by "npm pack" into
// destPath. The entries of such a tarball share a single top-level directory
// (usually "package"), which is stripped. Entries that aren't regular files
// or directories (symbolic links, hard links, devices) are skipped, and any
// entry that would land outside destPath is an error.
func extractNpmTarball(tarballPath, destPath string) error {
	file, err := os.Open(tarballPath)
	if err != nil {
		return burrito.WrapErrorf(err, osOpenError, tarballPath)
	}
	defer file.Close()
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return burrito.WrapErrorf(
			err, "Failed to read the tarball as a gzip file.\nPath: %s",
			tarballPath)
	}
	defer gzReader.Close()
	destPath, err = filepath.Abs(destPath)
	if err != nil {
		return burrito.WrapErrorf(err, filepathAbsError, destPath)
	}
	err = os.MkdirAll(destPath, 0755)
	if err != nil {
		return burrito.WrapErrorf(err, osMkdirError, destPath)
	}
	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return burrito.WrapErrorf(
				err, "Failed to read an entry of the tarball.\nPath: %s",
				tarballPath)
		}
		// Strip the top-level directory
		name := strings.TrimPrefix(header.Name, "./")
		i := strings.Index(name, "/")
		if i < 0 {
			// The top-level directory itself (or a stray file in it).
			continue
		}
		name = strings.TrimRight(name[i+1:], "/")
		if name == "" {
			continue
		}
		target := filepath.Join(destPath, filepath.FromSlash(name))
		relTarget, err := filepath.Rel(destPath, target)
		if err != nil || relTarget == ".." ||
			strings.HasPrefix(relTarget, ".."+string(filepath.Separator)) ||
			filepath.IsAbs(relTarget) {
			return burrito.WrappedErrorf(
				npmTarballEntryEscapesError, tarballPath, header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return burrito.WrapErrorf(err, osMkdirError, target)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return burrito.WrapErrorf(err, osMkdirError, filepath.Dir(target))
			}
			// Keep the permission bits of the entry (an executable
			// subfilter needs them) but make sure the owner can always
			// read and write the file.
			mode := header.FileInfo().Mode().Perm()&fs.ModePerm | 0600
			outFile, err := os.OpenFile(
				target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return burrito.WrapErrorf(err, osCreateError, target)
			}
			_, err = io.Copy(outFile, tarReader)
			outFile.Close()
			if err != nil {
				return burrito.WrapErrorf(err, fileWriteError, target)
			}
		default:
			Logger.Debugf(
				"Skipped the %q entry of the tarball: it's not a regular "+
					"file or a directory.", header.Name)
		}
	}
	return nil
}
