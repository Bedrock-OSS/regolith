package regolith

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"golang.org/x/mod/semver"
)

const latestCompatibleVersion = "1.9.0"

const StandardLibraryUrl = "github.com/Bedrock-OSS/regolith-filters"
const ConfigFilePath = "config.json"
const GitIgnore = "/build\n/.regolith"

// Config represents the full configuration file of Regolith, as saved in
// "config.json".
type Config struct {
	Name            string `json:"name,omitempty"`
	Author          string `json:"author,omitempty"`
	Packs           Packs  `json:"packs,omitzero"`
	RegolithProject `json:"regolith,omitzero"`
}

// ExportTarget is a part of "config.json" that contains export information
// for a profile, which denotes where compiled files will go.
// When editing, adjust ExportTargetFromObject function as well.
type ExportTarget struct {
	Target    string            `json:"target,omitempty"`    // The mode of exporting. "develop" or "exact"
	RpPath    string            `json:"rpPath,omitempty"`    // Relative or absolute path to resource pack for "exact" export target
	BpPath    string            `json:"bpPath,omitempty"`    // Relative or absolute path to behavior pack for "exact" export target
	RpName    string            `json:"rpName,omitempty"`
	BpName    string            `json:"bpName,omitempty"`
	WorldName string            `json:"worldName,omitempty"`
	WorldPath string            `json:"worldPath,omitempty"`
	ReadOnly  bool              `json:"readOnly"`            // Whether the exported files should be read-only
	Build     string            `json:"build,omitempty"`     // The type of Minecraft build for the 'develop'
	BpNames   map[string]string `json:"bpNames,omitempty"`   // per-pack name overrides (>=1.9.0)
	RpNames   map[string]string `json:"rpNames,omitempty"`
	BpPaths   map[string]string `json:"bpPaths,omitempty"`   // per-pack "exact" paths (>=1.9.0)
	RpPaths   map[string]string `json:"rpPaths,omitempty"`
}

// ExportTargets is the config representation of a profile's "export" value.
// It accepts both the single-object form and the multi-target array
// form. When marshaling, a single target is written as an object to keep newly
// generated configs backward compatible with older Regolith versions.
type ExportTargets []ExportTarget

// IsZero lets json:",omitzero" omit an unset target list.
func (et ExportTargets) IsZero() bool {
	return len(et) == 0
}

func (et ExportTargets) MarshalJSON() ([]byte, error) {
	if len(et) == 1 {
		return json.Marshal(et[0])
	}
	return json.Marshal([]ExportTarget(et))
}

func (et *ExportTargets) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	targets, err := ExportTargetsFromObject(raw)
	if err != nil {
		return err
	}
	*et = targets
	return nil
}

// Pack is a single source pack mapped into the tmp working directory under a
// folder named Name ("BP", "BP1", "RP", ...). Source may be empty when the
// pack has no source on disk and is produced by filters.
type Pack struct {
	Name   string
	Source string
}

// Packs is a part of "config.json" that points to the source behavior and
// resource packs. The first behavior pack is always named "BP" and the first
// resource pack "RP" (the primary packs the filters operate on).
type Packs struct {
	BehaviorPacks []Pack
	ResourcePacks []Pack
}

// IsZero lets json:",omitzero" omit an unset Packs value.
func (p Packs) IsZero() bool {
	return len(p.BehaviorPacks) == 0 && len(p.ResourcePacks) == 0
}

// packSuffix returns the numeric suffix of a pack name ("" for "BP", "1" for
// "BP1", "12" for "BP12").
func packSuffix(name string) string {
	i := 0
	for i < len(name) && (name[i] < '0' || name[i] > '9') {
		i++
	}
	return name[i:]
}

// packIndex returns the numeric index of a pack name (0 for the bare "BP"/"RP").
func packIndex(name string) int {
	suffix := packSuffix(name)
	if suffix == "" {
		return 0
	}
	n, err := strconv.Atoi(suffix)
	if err != nil {
		return 0
	}
	return n
}

// sortPacks orders packs by numeric index so the primary ("BP"/"RP") comes
// first, then "BP1", "BP2", ...
func sortPacks(packs []Pack) {
	sort.SliceStable(packs, func(i, j int) bool {
		return packIndex(packs[i].Name) < packIndex(packs[j].Name)
	})
}

func hasPackNamed(packs []Pack, name string) bool {
	for _, pack := range packs {
		if pack.Name == name {
			return true
		}
	}
	return false
}

// SortPacksForTest exposes sortPacks for external tests.
func SortPacksForTest(packs []Pack) { sortPacks(packs) }

var behaviorPackKeyRe = regexp.MustCompile(`^BP([1-9][0-9]*)?$`)
var resourcePackKeyRe = regexp.MustCompile(`^RP([1-9][0-9]*)?$`)

// MarshalJSON writes the single-pack default (one "BP" and/or one "RP") as the
// legacy string form for backward compatibility, and the multi-pack form as
// name->path maps.
func (p Packs) MarshalJSON() ([]byte, error) {
	obj := map[string]any{}
	marshalPackList(obj, p.BehaviorPacks, "BP", "behaviorPack", "behaviorPacks")
	marshalPackList(obj, p.ResourcePacks, "RP", "resourcePack", "resourcePacks")
	return json.Marshal(obj)
}

func marshalPackList(
	obj map[string]any, packs []Pack, primaryName, singularKey, pluralKey string,
) {
	isSingleDefault := len(packs) == 1 && packs[0].Name == primaryName
	if isSingleDefault {
		if packs[0].Source != "" {
			obj[singularKey] = packs[0].Source
		}
		return
	}
	if len(packs) == 0 {
		return
	}
	m := make(map[string]string, len(packs))
	for _, pack := range packs {
		m[pack.Name] = pack.Source
	}
	obj[pluralKey] = m
}

// RegolithProject is a part of "config.json" with the regolith namespace
// within the Minecraft Project Schema
type RegolithProject struct {
	Profiles          map[string]Profile         `json:"profiles,omitempty"`
	FilterDefinitions map[string]FilterInstaller `json:"filterDefinitions"`
	DataPath          string                     `json:"dataPath,omitempty"`
	WatchPaths        []string                   `json:"watchPaths,omitempty"`
	FormatVersion     string                     `json:"formatVersion,omitempty"`
}

// ConfigFromObject creates a "Config" object from map[string]interface{}
func ConfigFromObject(obj map[string]any) (*Config, error) {
	result := &Config{}
	// Name
	name, ok := obj["name"].(string)
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPathMissingError, "name")
	}
	result.Name = name
	// Author
	author, ok := obj["author"].(string)
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPathMissingError, "author")
	}
	result.Author = author
	// Read formatVersion early so packs parsing can gate the multi-pack map
	// form. Full validation of formatVersion still happens in
	// RegolithProjectFromObject below.
	formatVersion := "1.2.0"
	if regolithObj, ok := obj["regolith"].(map[string]any); ok {
		if fv, ok := regolithObj["formatVersion"].(string); ok {
			formatVersion = fv
		}
	}
	// Packs
	if packs, ok := obj["packs"]; ok {
		packs, ok := packs.(map[string]any)
		if !ok {
			return nil, burrito.WrappedErrorf(jsonPathTypeError, "packs", "object")
		}
		parsedPacks, err := PacksFromObject(packs, formatVersion)
		if err != nil {
			return nil, burrito.WrapErrorf(err, jsonPropertyParseError, "packs")
		}
		result.Packs = parsedPacks
	} else {
		return nil, burrito.WrappedErrorf(jsonPathMissingError, "packs")
	}
	// Regolith
	if regolith, ok := obj["regolith"]; ok {
		regolith, ok := regolith.(map[string]any)
		if !ok {
			return nil, burrito.WrappedErrorf(
				jsonPathTypeError, "regolith", "object")
		}
		regolithProject, err := RegolithProjectFromObject(regolith)
		if err != nil {
			return nil, burrito.WrapErrorf(err, jsonPropertyParseError, "regolith")
		}
		result.RegolithProject = regolithProject
	} else {
		return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "regolith")
	}
	return result, nil
}

// PacksFromObject creates a "Packs" object from map[string]interface{}. The
// formatVersion gates the multi-pack map form: it is only allowed for
// formatVersion >= 1.9.0. Older versions accept only the singular string form.
func PacksFromObject(obj map[string]any, formatVersion string) (Packs, error) {
	result := Packs{}
	multiPackSupported := semver.Compare("v"+formatVersion, "v1.9.0") >= 0

	behaviorPacks, err := parsePackField(
		obj, "behaviorPack", "behaviorPacks", "BP",
		behaviorPackKeyRe, multiPackSupported)
	if err != nil {
		return result, burrito.PassError(err)
	}
	result.BehaviorPacks = behaviorPacks

	resourcePacks, err := parsePackField(
		obj, "resourcePack", "resourcePacks", "RP",
		resourcePackKeyRe, multiPackSupported)
	if err != nil {
		return result, burrito.PassError(err)
	}
	result.ResourcePacks = resourcePacks
	return result, nil
}

// parsePackField parses one pack type. It accepts the singular string form
// (single primary pack) or the plural map form (multiple packs, >=1.9.0). The
// primary pack (primaryName) is always present in the returned slice.
func parsePackField(
	obj map[string]any,
	singularKey, pluralKey, primaryName string,
	keyRe *regexp.Regexp,
	multiPackSupported bool,
) ([]Pack, error) {
	_, hasPlural := obj[pluralKey]
	_, hasSingular := obj[singularKey]

	if hasPlural {
		if !multiPackSupported {
			return nil, burrito.WrappedErrorf(multiPackVersionError, pluralKey)
		}
		if hasSingular {
			return nil, burrito.WrappedErrorf(
				packsMixedFormError, singularKey, pluralKey)
		}
		packMap, ok := obj[pluralKey].(map[string]any)
		if !ok {
			return nil, burrito.WrappedErrorf(jsonPropertyTypeError, pluralKey, "object")
		}
		packs := make([]Pack, 0, len(packMap))
		for name, src := range packMap {
			if !keyRe.MatchString(name) {
				return nil, burrito.WrappedErrorf(packKeyInvalidError, name, pluralKey)
			}
			srcStr, ok := src.(string)
			if !ok {
				return nil, burrito.WrappedErrorf(
					jsonPathTypeError, pluralKey+"->"+name, "string")
			}
			packs = append(packs, Pack{Name: name, Source: srcStr})
		}
		if !hasPackNamed(packs, primaryName) {
			packs = append(packs, Pack{Name: primaryName, Source: ""})
		}
		sortPacks(packs)
		return packs, nil
	}

	// Singular string form (legacy and >=1.9.0 sugar). The primary pack always
	// exists, even when the source is empty/missing.
	src, _ := obj[singularKey].(string)
	return []Pack{{Name: primaryName, Source: src}}, nil
}

// RegolithProjectFromObject creates a "RegolithProject" object from
// map[string]interface{}
func RegolithProjectFromObject(
	obj map[string]any,
) (RegolithProject, error) {
	result := RegolithProject{
		Profiles:          make(map[string]Profile),
		FilterDefinitions: make(map[string]FilterInstaller),
	}
	// FormatVersion
	if version, ok := obj["formatVersion"]; !ok {
		Logger.Warn("Format version is missing. Defaulting to 1.2.0")
		result.FormatVersion = "1.2.0"
	} else {
		formatVersion, ok := version.(string)
		if !ok {
			return result, burrito.WrappedErrorf(
				jsonPropertyTypeError, "formatVersion", "string")
		}
		result.FormatVersion = formatVersion
		vFormatVersion := "v" + formatVersion
		if !semver.IsValid("v" + formatVersion) {
			return result, burrito.WrappedErrorf(
				"Invalid value of formatVersion. The formatVersion must "+
					"be a semver version:\n"+
					"Current value: %s", formatVersion)
		}
		if semver.Compare(vFormatVersion, "v"+latestCompatibleVersion) > 0 {
			return result, burrito.WrappedErrorf(
				incompatibleFormatVersionError,
				formatVersion, latestCompatibleVersion)
		}
	}

	// DataPath
	if _, ok := obj["dataPath"]; !ok {
		return result, burrito.WrappedErrorf(jsonPropertyMissingError, "dataPath")
	}
	dataPath, ok := obj["dataPath"].(string)
	if !ok {
		return result, burrito.WrappedErrorf(
			jsonPropertyTypeError, "dataPath", "string")
	}
	result.DataPath = dataPath
	// WatchPaths
	if watchPaths, ok := obj["watchPaths"].([]any); ok {
		for i, path := range watchPaths {
			if path, ok := path.(string); ok {
				result.WatchPaths = append(result.WatchPaths, path)
			} else {
				return result, burrito.WrappedErrorf(
					jsonPathTypeError, fmt.Sprintf("watchPaths->%d", i), "string")
			}
		}
	}
	// Filter definitions
	filterDefinitions, ok := obj["filterDefinitions"].(map[string]any)
	if ok { // filter definitions are optional
		for filterDefinitionName, filterDefinition := range filterDefinitions {
			filterDefinitionMap, ok := filterDefinition.(map[string]any)
			if !ok {
				return result, burrito.WrappedErrorf(
					jsonPropertyTypeError, "filterDefinitions",
					"object")
			}
			filterInstaller, err := FilterInstallerFromObject(
				filterDefinitionName, filterDefinitionName, filterDefinitionMap)
			if err != nil {
				return result, burrito.WrapErrorf(
					err, jsonPropertyParseError, "filterDefinitions")
			}
			result.FilterDefinitions[filterDefinitionName] = filterInstaller
		}
	}
	// Profiles
	profiles, ok := obj["profiles"].(map[string]any)
	if !ok {
		return result, burrito.WrappedErrorf(jsonPropertyMissingError, "profiles")
	}
	for profileName, profile := range profiles {
		profileMap, ok := profile.(map[string]any)
		if !ok {
			return result, burrito.WrappedErrorf(
				jsonPropertyTypeError,
				"profiles->"+profileName, "object")
		}
		profileValue, err := ProfileFromObject(
			profileMap, result.FilterDefinitions)
		if err != nil {
			return result, burrito.WrapErrorf(
				err, jsonPropertyParseError, "profiles->"+profileName)
		}
		result.Profiles[profileName] = profileValue
	}
	return result, nil
}

// ExportTargetsFromObject parses the "export" value which can be either a
// single object (backward compatible) or an array of objects.
func ExportTargetsFromObject(exportValue any) (ExportTargets, error) {
	switch v := exportValue.(type) {
	case map[string]any:
		et, err := ExportTargetFromObject(v)
		if err != nil {
			return nil, burrito.WrapErrorf(err, jsonPropertyParseError, "export")
		}
		return ExportTargets{et}, nil
	case []any:
		if len(v) == 0 {
			return nil, burrito.WrappedErrorf(
				"The \"export\" array must contain at least one entry")
		}
		targets := make(ExportTargets, 0, len(v))
		for i, item := range v {
			obj, ok := item.(map[string]any)
			if !ok {
				return nil, burrito.WrappedErrorf(
					jsonPathTypeError, fmt.Sprintf("export->%d", i), "object")
			}
			et, err := ExportTargetFromObject(obj)
			if err != nil {
				return nil, burrito.WrapErrorf(
					err, jsonPropertyParseError, fmt.Sprintf("export->%d", i))
			}
			targets = append(targets, et)
		}
		return targets, nil
	default:
		return nil, burrito.WrappedErrorf(
			jsonPropertyTypeError, "export", "object or array")
	}
}

// ExportTargetFromObject creates a "ExportTarget" object from
// map[string]interface{}
func ExportTargetFromObject(obj map[string]any) (ExportTarget, error) {
	result := ExportTarget{}
	// Target
	targetObj, ok := obj["target"]
	if !ok {
		return result, burrito.WrappedErrorf(jsonPropertyMissingError, "target")
	}
	target, ok := targetObj.(string)
	if !ok {
		return result, burrito.WrappedErrorf(
			jsonPropertyTypeError, "target", "string")
	}
	result.Target = target
	// RpPath - can be empty
	rpPath, _ := obj["rpPath"].(string)
	result.RpPath = rpPath
	// BpPath - can be empty
	bpPath, _ := obj["bpPath"].(string)
	result.BpPath = bpPath
	// RpName - can be empty
	rpName, _ := obj["rpName"].(string)
	result.RpName = rpName
	// BpName - can be empty
	bpName, _ := obj["bpName"].(string)
	result.BpName = bpName
	// WorldName - can be empty
	worldName, _ := obj["worldName"].(string)
	result.WorldName = worldName
	// WorldPath - can be empty
	worldPath, _ := obj["worldPath"].(string)
	result.WorldPath = worldPath
	// ReadOnly - can be empty
	readOnly, _ := obj["readOnly"].(bool)
	result.ReadOnly = readOnly
	// Build - can be empty
	build, _ := obj["build"].(string)
	result.Build = build
	// Per-pack overrides (used only for formatVersion >= 1.9.0 multi-pack).
	result.BpNames = stringMapFromObject(obj, "bpNames")
	result.RpNames = stringMapFromObject(obj, "rpNames")
	result.BpPaths = stringMapFromObject(obj, "bpPaths")
	result.RpPaths = stringMapFromObject(obj, "rpPaths")
	return result, nil
}

// stringMapFromObject reads a JSON object value into a map[string]string,
// returning nil when the key is absent or not an object.
func stringMapFromObject(obj map[string]any, key string) map[string]string {
	raw, ok := obj[key].(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string]string, len(raw))
	for k, v := range raw {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}
