package regolith

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"github.com/otiai10/copy"
	"golang.org/x/mod/semver"
)

// GetExportPaths returns file paths for exporting behavior pack and
// resource pack based on exportTarget (a structure with data related to
// export settings) and the name of the project.
func GetExportPaths(
	exportTarget ExportTarget, ctx RunContext,
) (bpPath string, rpPath string, err error) {
	bpName, rpName, err := GetExportNames(exportTarget, ctx)
	if err != nil {
		return "", "", burrito.WrapError(
			err, "Failed to get the export names.")
	}
	vFormatVersion := "v" + ctx.Config.FormatVersion

	if semver.Compare(vFormatVersion, "v1.4.0") < 0 {
		bpPath, rpPath, err = getExportPathsV1_2_0(
			exportTarget, bpName, rpName)
	} else if semver.Compare(vFormatVersion, "v"+latestCompatibleVersion) <= 0 {
		bpPath, rpPath, err = getExportPathsV1_4_0(
			exportTarget, bpName, rpName)
	} else {
		err = burrito.WrappedErrorf(
			incompatibleFormatVersionError,
			ctx.Config.FormatVersion, latestCompatibleVersion)
	}
	return
}

// GetPackExportPath returns the export destination for a single pack. For
// formatVersion < 1.9.0 (single pack only) it delegates to the legacy
// GetExportPaths so behavior is unchanged. For >=1.9.0 it resolves a
// destination per pack using the 1.4.0-style com.mojang lookup.
func GetPackExportPath(
	exportTarget ExportTarget, ctx RunContext, pack Pack, packType string,
) (string, error) {
	if semver.Compare("v"+ctx.Config.FormatVersion, "v1.9.0") < 0 {
		bpPath, rpPath, err := GetExportPaths(exportTarget, ctx)
		if err != nil {
			return "", burrito.PassError(err)
		}
		if packType == "bp" {
			return bpPath, nil
		}
		return rpPath, nil
	}
	name, err := GetExportName(exportTarget, ctx, pack, packType)
	if err != nil {
		return "", burrito.PassError(err)
	}
	switch exportTarget.Target {
	case "development":
		comMojang, err := FindMojangDir(exportTarget.Build, PacksPath)
		if err != nil {
			return "", burrito.PassError(err)
		}
		subDir := "development_behavior_packs"
		if packType == "rp" {
			subDir = "development_resource_packs"
		}
		return comMojang + "/" + subDir + "/" + name, nil
	case "world":
		return getWorldPackExportPath(exportTarget, packType, name)
	case "exact":
		return getExactPackExportPath(exportTarget, pack, packType)
	case "local":
		return "build/" + name + "/", nil
	case "none":
		return "", nil
	default:
		return "", burrito.WrappedErrorf(
			"Export target %q is not valid", exportTarget.Target)
	}
}

func getWorldPackExportPath(
	exportTarget ExportTarget, packType, name string,
) (string, error) {
	subDir := "behavior_packs"
	if packType == "rp" {
		subDir = "resource_packs"
	}
	if exportTarget.WorldPath != "" {
		if exportTarget.WorldName != "" {
			return "", burrito.WrappedError(
				"Using both \"worldName\" and \"worldPath\" is not allowed.")
		}
		wPath, err := ResolvePath(exportTarget.WorldPath)
		if err != nil {
			return "", burrito.WrapError(err, "Failed to resolve world path.")
		}
		return filepath.Join(wPath, subDir, name), nil
	}
	if exportTarget.WorldName != "" {
		dir, err := FindMojangDir(exportTarget.Build, WorldPath)
		if err != nil {
			return "", burrito.WrapError(
				err, "Failed to find \"com.mojang\" directory.")
		}
		worlds, err := ListWorlds(dir)
		if err != nil {
			return "", burrito.WrapError(err, "Failed to list worlds.")
		}
		for _, world := range worlds {
			if world.Name == exportTarget.WorldName {
				return filepath.Join(world.Path, subDir, name), nil
			}
		}
		return "", burrito.WrappedErrorf(
			"Failed to find the world.\nWorld name: %s", exportTarget.WorldName)
	}
	return "", burrito.WrappedError(
		"The \"world\" export target requires either a \"worldName\" or " +
			"\"worldPath\" property")
}

func getExactPackExportPath(
	exportTarget ExportTarget, pack Pack, packType string,
) (string, error) {
	pathsMap := exportTarget.BpPaths
	singlePath := exportTarget.BpPath
	primaryName := "BP"
	pluralKey := "bpPaths"
	if packType == "rp" {
		pathsMap = exportTarget.RpPaths
		singlePath = exportTarget.RpPath
		primaryName = "RP"
		pluralKey = "rpPaths"
	}
	if raw, ok := pathsMap[pack.Name]; ok {
		return ResolvePath(raw)
	}
	if pack.Name == primaryName && singlePath != "" {
		return ResolvePath(singlePath)
	}
	return "", burrito.WrappedErrorf(exactPathMissingError, pack.Name, pluralKey)
}

func FindMojangDir(build string, pathType ComMojangPathType) (string, error) {
	switch build {
	case "standard":
		return FindStandardMojangDir(pathType)
	case "preview":
		return FindPreviewDir(pathType)
	case "education":
		return FindEducationDir()
		// WARNING: If for some reason we will expand this in the future to
		// match a new format version, we need to split this into versioned
		// functions.
	default:
		return "", burrito.WrappedErrorf(
			invalidExportPathError,
			// current value; valid values
			build, "standard, preview, education")
	}
}

// getExportPathsV1_2_0 handles GetExportPaths for Regolith format versions
// below 1.4.0.
func getExportPathsV1_2_0(
	exportTarget ExportTarget, bpName string, rpName string,
) (bpPath string, rpPath string, err error) {
	switch exportTarget.Target {
	case "development":
		comMojang, err := FindStandardMojangDir(PacksPath)
		if err != nil {
			return "", "", burrito.WrapError(
				err, findMojangDirError)
		}
		return GetDevelopmentExportPaths(bpName, rpName, comMojang)
	case "preview":
		comMojang, err := FindPreviewDir(PacksPath)
		if err != nil {
			return "", "", burrito.WrapError(
				err, findPreviewDirError)
		}
		return GetDevelopmentExportPaths(bpName, rpName, comMojang)
	case "exact":
		return GetExactExportPaths(exportTarget)
	case "world":
		return GetWorldExportPaths(
			exportTarget.WorldPath,
			exportTarget.WorldName,
			"standard",
			bpName, rpName)
	case "local":
		bpPath = "build/" + bpName + "/"
		rpPath = "build/" + rpName + "/"
	case "none":
		bpPath = ""
		rpPath = ""
	default:
		err = burrito.WrappedErrorf(
			"Export target %q is not valid", exportTarget.Target)
	}
	return
}

// getExportPathsV1_4_0 handles GetExportPaths for Regolith format version
// 1.4.0.
func getExportPathsV1_4_0(
	exportTarget ExportTarget, bpName string, rpName string,
) (bpPath string, rpPath string, err error) {
	switch exportTarget.Target {
	case "development":
		comMojang, err := FindMojangDir(exportTarget.Build, PacksPath)
		if err != nil {
			return "", "", burrito.PassError(err)
		}
		return GetDevelopmentExportPaths(bpName, rpName, comMojang)
	case "world":
		return GetWorldExportPaths(
			exportTarget.WorldPath,
			exportTarget.WorldName,
			exportTarget.Build,
			bpName, rpName)
	case "exact":
		return GetExactExportPaths(exportTarget)
	case "local":
		bpPath = "build/" + bpName + "/"
		rpPath = "build/" + rpName + "/"
	case "none":
		bpPath = ""
		rpPath = ""
	default:
		err = burrito.WrappedErrorf(
			"Export target %q is not valid", exportTarget.Target)
	}
	return
}

func GetDevelopmentExportPaths(bpName, rpName, comMojang string) (bpPath string, rpPath string, err error) {
	bpPath = comMojang + "/development_behavior_packs/" + bpName
	rpPath = comMojang + "/development_resource_packs/" + rpName
	return
}

func GetExactExportPaths(exportTarget ExportTarget) (bpPath string, rpPath string, err error) {
	bpPath, err = ResolvePath(exportTarget.BpPath)
	if err != nil {
		return "", "", burrito.WrapError(
			err, "Failed to resolve behavior pack path.")
	}
	rpPath, err = ResolvePath(exportTarget.RpPath)
	if err != nil {
		return "", "", burrito.WrapError(
			err, "Failed to resolve resource pack path.")
	}
	return
}

func GetWorldExportPaths(
	worldPath, worldName, build, bpName, rpName string,
) (bpPath string, rpPath string, err error) {
	if worldPath != "" {
		if worldName != "" {
			return "", "", burrito.WrappedError(
				"Using both \"worldName\" and \"worldPath\" is not" +
					" allowed.")
		}
		wPath, err := ResolvePath(worldPath)
		if err != nil {
			return "", "", burrito.WrapError(
				err, "Failed to resolve world path.")
		}
		bpPath = filepath.Join(
			wPath, "behavior_packs", bpName)
		rpPath = filepath.Join(
			wPath, "resource_packs", rpName)
	} else if worldName != "" {
		dir, err := FindMojangDir(build, WorldPath)
		if err != nil {
			return "", "", burrito.WrapError(
				err, "Failed to find \"com.mojang\" directory.")
		}
		worlds, err := ListWorlds(dir)
		if err != nil {
			return "", "", burrito.WrapError(err, "Failed to list worlds.")
		}
		for _, world := range worlds {
			if world.Name != worldName {
				continue
			}
			bpPath = filepath.Join(
				world.Path, "behavior_packs", bpName)
			rpPath = filepath.Join(
				world.Path, "resource_packs", rpName)
			return bpPath, rpPath, nil
		}
		return "", "", burrito.WrappedErrorf(
			"Failed to find the world.\n"+
				"World name: %s", worldName)
	} else {
		err = burrito.WrappedError(
			"The \"world\" export target requires either a " +
				"\"worldName\" or \"worldPath\" property")
	}
	return
}

// GetExportNames returns the names for the behavior pack and resource pack
// based on the evaluated values of the "bpName" and "rpName" from the
// exportTarget object.
func GetExportNames(exportTarget ExportTarget, ctx RunContext) (bpName string, rpName string, err error) {
	if exportTarget.BpName != "" {
		bpName, err = EvalString(exportTarget.BpName, ctx)
		if err != nil {
			return "", "", burrito.WrapError(
				err, "Failed to evaluate behavior pack name.")
		}
	} else {
		bpName = ctx.Config.Name + "_bp"
	}
	if exportTarget.RpName != "" {
		rpName, err = EvalString(exportTarget.RpName, ctx)
		if err != nil {
			return "", "", burrito.WrapError(
				err, "Failed to evaluate resource pack name.")
		}
	} else {
		rpName = ctx.Config.Name + "_rp"
	}
	return
}

// GetExportName returns the export pack name for a single pack. Resolution
// order: per-pack override map (bpNames/rpNames), then the singular
// bpName/rpName for the primary pack, then the default
// "<projectName>_bp<index>" / "<projectName>_rp<index>".
func GetExportName(
	exportTarget ExportTarget, ctx RunContext, pack Pack, packType string,
) (string, error) {
	namesMap := exportTarget.BpNames
	singleName := exportTarget.BpName
	primaryName := "BP"
	if packType == "rp" {
		namesMap = exportTarget.RpNames
		singleName = exportTarget.RpName
		primaryName = "RP"
	}
	if expr, ok := namesMap[pack.Name]; ok && expr != "" {
		name, err := EvalString(expr, ctx)
		if err != nil {
			return "", burrito.WrapError(err, "Failed to evaluate pack name.")
		}
		return name, nil
	}
	if pack.Name == primaryName && singleName != "" {
		name, err := EvalString(singleName, ctx)
		if err != nil {
			return "", burrito.WrapError(err, "Failed to evaluate pack name.")
		}
		return name, nil
	}
	return ctx.Config.Name + "_" + strings.ToLower(packType) + packSuffix(pack.Name), nil
}

type resolvedPackExport struct {
	pack     Pack
	packType string // "bp" or "rp"
	destPath string
}

type resolvedExportTarget struct {
	target      ExportTarget
	packExports []resolvedPackExport
}

func normalizeExportPathForCollision(path string) (string, error) {
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", burrito.WrapErrorf(err, filepathAbsError, path)
	}
	if resolvedPath, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = resolvedPath
	}
	if runtime.GOOS == "windows" {
		absPath = strings.ToLower(absPath)
	}
	return absPath, nil
}

func pathContains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator)) &&
		!filepath.IsAbs(rel)
}

func checkExportPathCollision(seen map[string]string, path, label string) error {
	normalizedPath, err := normalizeExportPathForCollision(path)
	if err != nil {
		return burrito.PassError(err)
	}
	for seenPath, seenLabel := range seen {
		if normalizedPath == seenPath ||
			pathContains(normalizedPath, seenPath) ||
			pathContains(seenPath, normalizedPath) {
			return burrito.WrappedErrorf(
				"Export path collision detected.\n"+
					"First path: %s\n"+
					"Second path: %s\n"+
					"Overlapping path: %s",
				seenLabel, label, normalizedPath)
		}
	}
	seen[normalizedPath] = label
	return nil
}

// ExportProject copies files from the tmp pack folders (tmp/BP, tmp/BP1,
// tmp/RP, ...) into the project's export targets. Destinations are generated
// per pack with GetPackExportPath.
func ExportProject(ctx RunContext) error {
	MeasureStart("Export - GetExportPaths")
	profile, err := ctx.GetProfile()
	if err != nil {
		return burrito.WrapError(err, runContextGetProfileError)
	}
	// Resolve all non-"none" targets (and all their packs) before modifying any
	// export path, so failure stays atomic.
	var activeTargets []resolvedExportTarget
	seenExportPaths := make(map[string]string)
	for i, exportTarget := range profile.activeExportTargets() {
		var packExports []resolvedPackExport
		resolvePack := func(pack Pack, packType string) error {
			destPath, err := GetPackExportPath(exportTarget, ctx, pack, packType)
			if err != nil {
				return burrito.WrapError(err, getExportPathsError)
			}
			label := fmt.Sprintf(
				"export target %d (%s) %s pack %q: %s",
				i+1, exportTarget.Target, packType, pack.Name, destPath)
			if err := checkExportPathCollision(seenExportPaths, destPath, label); err != nil {
				return burrito.PassError(err)
			}
			packExports = append(packExports, resolvedPackExport{pack, packType, destPath})
			return nil
		}
		for _, pack := range ctx.Config.Packs.BehaviorPacks {
			if err := resolvePack(pack, "bp"); err != nil {
				return burrito.PassError(err)
			}
		}
		for _, pack := range ctx.Config.Packs.ResourcePacks {
			if err := resolvePack(pack, "rp"); err != nil {
				return burrito.PassError(err)
			}
		}
		activeTargets = append(activeTargets, resolvedExportTarget{
			target:      exportTarget,
			packExports: packExports,
		})
	}
	if len(activeTargets) == 0 {
		Logger.Debugf("All export targets are set to \"none\". Skipping export.")
		return nil
	}
	dotRegolithPath := ctx.DotRegolithPath
	isSinglePack := len(ctx.Config.Packs.BehaviorPacks) == 1 && len(ctx.Config.Packs.ResourcePacks) == 1
	useSymlink := ctx.SymlinkExport && len(activeTargets) == 1 && isSinglePack
	editedFiles := LoadEditedFiles(dotRegolithPath)
	if !useSymlink && !ctx.UnsafeMode {
		MeasureStart("Export - CheckDeletionSafety")
		for _, exportTarget := range activeTargets {
			for _, pe := range exportTarget.packExports {
				if err := editedFiles.CheckPackDeletionSafety(pe.packType, pe.destPath); err != nil {
					return burrito.WrapErrorf(
						err, checkDeletionSafetyError, pe.destPath, pe.destPath)
				}
			}
		}
	}

	for i, exportTarget := range activeTargets {
		if useSymlink && i == 0 {
			Logger.Debugf("Symlink export is enabled. Skipping RP and BP export.")
			continue
		}
		// Move is only safe when there is exactly one active target and symlink
		// export is off, since each tmp pack folder is the sole source.
		canMove := len(activeTargets) == 1 && !useSymlink
		if err := exportTargetPacks(exportTarget, ctx, canMove); err != nil {
			return burrito.PassError(err)
		}
	}
	// Export data once (not per target)
	MeasureStart("Export - ExportData")
	if err := exportProjectData(profile, ctx); err != nil {
		return burrito.PassError(err)
	}
	MeasureStart("Export - EditedFiles.UpdateFromPaths")
	for _, exportTarget := range activeTargets {
		for _, pe := range exportTarget.packExports {
			if err := editedFiles.UpdatePackFromPath(pe.packType, pe.destPath); err != nil {
				return burrito.WrapError(
					err,
					"Failed to create a list of files edited by this 'regolith run'")
			}
		}
	}
	if err := editedFiles.Dump(dotRegolithPath); err != nil {
		return burrito.WrapError(err, updatedFilesDumpError)
	}
	MeasureStart("Export - Remove Empty Export Paths")
	for i, exportTarget := range activeTargets {
		if useSymlink && i == 0 {
			continue
		}
		for _, pe := range exportTarget.packExports {
			pathEmpty, _ := IsDirEmpty(pe.destPath)
			if pathEmpty {
				if err := os.Remove(pe.destPath); err != nil {
					Logger.Warnf(
						"Failed to remove empty pack directory.\n"+
							"Path: %s\nError: %v", pe.destPath, err)
				}
			}
		}
	}
	MeasureEnd()
	return nil
}

// exportTargetPacks exports every pack of a resolved target from its tmp folder
// to its destination. Moving is only allowed for a single active target without
// symlink export, so the tmp source stays intact for additional targets.
func exportTargetPacks(rt resolvedExportTarget, ctx RunContext, allowMove bool) error {
	dotRegolithPath := ctx.DotRegolithPath
	absWorkingDir, err := GetAbsoluteWorkingDirectory(dotRegolithPath)
	if err != nil {
		return burrito.WrapError(err, getAbsoluteWorkingDirectoryError)
	}
	if ctx.DisableSizeTimeCheck {
		MeasureStart("Export - Clean")
		for _, pe := range rt.packExports {
			if err := removeJunctionSafe(pe.destPath); err != nil {
				return burrito.WrapErrorf(
					err, "Failed to clear %s pack from build path %q.\n"+
						"Are user permissions correct?", pe.packType, pe.destPath)
			}
		}
	}
	MeasureStart("Export - MoveOrCopy")
	var wg sync.WaitGroup
	errChan := make(chan error, len(rt.packExports))
	for _, pe := range rt.packExports {
		pe := pe
		source := filepath.Join(absWorkingDir, pe.pack.Name)
		wg.Go(func() {
			Logger.Infof("Exporting %s pack %q to \"%s\".", pe.packType, pe.pack.Name, pe.destPath)
			var e error
			if !ctx.DisableSizeTimeCheck {
				e = SyncDirectories(source, pe.destPath, rt.target.ReadOnly)
			} else if allowMove {
				e = MoveOrCopy(source, pe.destPath, rt.target.ReadOnly, true)
			} else {
				e = copyExportPath(source, pe.destPath, rt.target.ReadOnly)
			}
			if e != nil {
				errChan <- burrito.WrapErrorf(e, "Failed to export %s pack %q.", pe.packType, pe.pack.Name)
				return
			}
			errChan <- nil
		})
	}
	wg.Wait()
	close(errChan)
	for e := range errChan {
		if e != nil {
			return e
		}
	}
	return nil
}

func copyExportPath(source, destination string, makeReadOnly bool) error {
	copySource := source
	if resolvedSource, err := filepath.EvalSymlinks(source); err == nil {
		copySource = resolvedSource
	}
	copyOptions := copy.Options{
		PreserveTimes: false,
		Sync:          false,
	}
	if runtime.GOOS == "windows" {
		copyOptions.PermissionControl = copy.DoNothing
	}
	if err := copy.Copy(copySource, destination, copyOptions); err != nil {
		return burrito.WrapErrorf(err, osCopyError, source, destination)
	}
	if makeReadOnly {
		setPathReadOnly(destination)
	}
	return nil
}

// exportProjectData is a helper function for ExportProject. It exports the 'data'
// folder back to the project's source files for the filters that opted-in for
// that with exportProjectData option.
func exportProjectData(profile Profile, ctx RunContext) error {
	dataPath := ctx.Config.DataPath
	dotRegolithPath := ctx.DotRegolithPath
	// List the names of the filters that opt-in to the data export process
	var exportedFilterNames []string
	err := profile.ForeachFilter(ctx, func(filter FilterRunner) error {
		usingDataPath, err := filter.IsUsingDataExport(dotRegolithPath, ctx)
		if err != nil {
			return burrito.WrapErrorf(
				err,
				"Failed to check if filter is using data export.\n"+
					"Path: %s", filter.GetId())
		}
		if usingDataPath {
			// Make sure that the filter name isn't a path that tries to access
			// files outside of the data path.
			filterName := filter.GetId()
			for _, forbidden := range []string{"..", "/", "\\", ":"} {
				if strings.Contains(filterName, forbidden) {
					// Other cases should be handled by mkdirAll
					return burrito.WrappedErrorf(
						"Filter name %q contains %q which is not allowed.",
						filterName, forbidden)
				}
			}
			// Add the filter name to the list of paths to export
			exportedFilterNames = append(exportedFilterNames, filter.GetId())
		}
		return nil
	}, true)
	if err != nil {
		return burrito.WrapError(err, "Failed to walk the list of the filters.")
	}
	if len(exportedFilterNames) == 0 {
		return nil
	}
	// The root of the data path cannot be deleted because the
	// "regolith watch" function would stop watching the file changes
	// (due to Windows API limitation).
	_, err = os.ReadDir(dataPath)
	if err != nil {
		var err1 error = nil
		if os.IsNotExist(err) {
			err1 = os.MkdirAll(dataPath, 0755)
		}
		if err1 != nil {
			return burrito.WrapErrorf(err, osReadDirError, dataPath)
		}
	}
	// Create revertible operations object
	backupPath := filepath.Join(dotRegolithPath, ".dataBackup")
	revertibleOps, err := NewRevertibleFsOperations(backupPath)
	if err != nil {
		return burrito.WrapErrorf(err, newRevertibleFsOperationsError, backupPath)
	}
	// Export data
	absWorkingDir, err := GetAbsoluteWorkingDirectory(dotRegolithPath)
	if err != nil {
		return burrito.WrapError(err, getAbsoluteWorkingDirectoryError)
	}
	for _, exportedFilterName := range exportedFilterNames {
		// Clear export target
		targetPath := filepath.Join(dataPath, exportedFilterName)
		if _, err := os.Stat(targetPath); err == nil {
			err = revertibleOps.Delete(targetPath)
			if err != nil {
				handlerError := revertibleOps.Undo()
				mainError := burrito.WrapErrorf(err, updateSourceFilesError, targetPath)
				if handlerError != nil {
					return burrito.GroupErrors(mainError, burrito.WrapError(handlerError, fsUndoError))
				}
				if handlerError := revertibleOps.Close(); handlerError != nil {
					return burrito.GroupErrors(mainError, handlerError)
				}
				return mainError
			}
		} else if os.IsNotExist(err) {
			err = os.MkdirAll(targetPath, 0755)
			if err != nil {
				return burrito.WrapErrorf(err, osMkdirError, targetPath)
			}
		} else {
			return burrito.WrapErrorf(err, osStatErrorAny, targetPath)
		}
		sourcePath := filepath.Join(absWorkingDir, "data", exportedFilterName)
		// If source path doesn't exist, skip
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			continue
		}
		// Copy data
		err = revertibleOps.MoveOrCopyDir(sourcePath, targetPath)
		if err != nil {
			handlerError := revertibleOps.Undo()
			mainError := burrito.WrapErrorf(err, moveOrCopyError, sourcePath, targetPath)
			if handlerError != nil {
				return burrito.GroupErrors(mainError, burrito.WrapError(handlerError, fsUndoError))
			}
			if handlerError := revertibleOps.Close(); handlerError != nil {
				return burrito.GroupErrors(mainError, handlerError)
			}
			return mainError
		}
	}
	if err := revertibleOps.Close(); err != nil {
		return burrito.PassError(err)
	}
	return nil
}

// InplaceExportProject copies the files from every pack's tmp folder
// (tmp/BP, tmp/BP1, …, tmp/RP, … and tmp/data) back into the project's source
// files. It's used by the "regolith apply-filter" command. This operation is
// destructive and cannot be undone.
func InplaceExportProject(
	config *Config, dotRegolithPath string,
) (err error) {
	// Create revertible ops object
	backupPath := filepath.Join(dotRegolithPath, ".dataBackup")
	revertibleOps, err := NewRevertibleFsOperations(backupPath)
	if err != nil {
		return burrito.WrapErrorf(err, newRevertibleFsOperationsError, backupPath)
	}
	// Schedule Undo in case of a revertible ops error and schedule Close()
	defer func() {
		if err != nil { // Handle previous error
			Logger.Warnf("Reverting changes...")
			handlerError := revertibleOps.Undo()
			if handlerError != nil {
				err = burrito.GroupErrors(err, burrito.WrapError(handlerError, fsUndoError))
				return
			}
			handlerError = revertibleOps.Close()
			if handlerError != nil {
				err = burrito.GroupErrors(err, handlerError)
			}
		} else { // No previous error but Close() must be called
			err = revertibleOps.Close()
			if err != nil {
				err = burrito.PassError(err)
			}
		}
	}()
	// Delete every pack source and data before replacing them with files from
	// tmp.
	var deleteDirs []string
	for _, pack := range config.Packs.BehaviorPacks {
		if pack.Source != "" {
			deleteDirs = append(deleteDirs, pack.Source)
		}
	}
	for _, pack := range config.Packs.ResourcePacks {
		if pack.Source != "" {
			deleteDirs = append(deleteDirs, pack.Source)
		}
	}
	if config.DataPath != "" {
		deleteDirs = append(deleteDirs, config.DataPath)
	}
	for _, deleteDir := range deleteDirs {
		err = revertibleOps.Delete(deleteDir)
		if err != nil {
			err = burrito.WrapErrorf(err, updateSourceFilesError, deleteDir)
			return err // Overwritten by defer
		}
	}
	// Move files from tmp to each pack source and data.
	absWorkingDir, err := GetAbsoluteWorkingDirectory(dotRegolithPath)
	if err != nil {
		return burrito.WrapError(err, getAbsoluteWorkingDirectoryError)
	}
	moveFiles := [][2]string{}
	for _, pack := range config.Packs.BehaviorPacks {
		if pack.Source != "" {
			moveFiles = append(moveFiles,
				[2]string{filepath.Join(absWorkingDir, pack.Name), pack.Source})
		}
	}
	for _, pack := range config.Packs.ResourcePacks {
		if pack.Source != "" {
			moveFiles = append(moveFiles,
				[2]string{filepath.Join(absWorkingDir, pack.Name), pack.Source})
		}
	}
	if config.DataPath != "" {
		moveFiles = append(moveFiles,
			[2]string{filepath.Join(absWorkingDir, "data"), config.DataPath})
	}
	for _, moveFile := range moveFiles {
		source, target := moveFile[0], moveFile[1]
		err = revertibleOps.MoveOrCopyDir(source, target)
		if err != nil {
			err = burrito.WrapErrorf(err, moveOrCopyError, source, target)
			return err // Overwritten by defer
		}
	}
	return err // Can be altered by defer
}
