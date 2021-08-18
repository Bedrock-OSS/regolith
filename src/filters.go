package src

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/fatih/color"
	"github.com/otiai10/copy"
)

type filterFunc func(filter Filter, absoluteLocation string)
type checkFunc func()

type filterDefinition struct {
	filter filterFunc
	check  checkFunc
}

var FilterTypes = map[string]filterDefinition{}

func RegisterFilters() {
	RegisterPythonFilter(FilterTypes)
	RegisterNodeJSFilter(FilterTypes)
}

func Setup() error {
	start := time.Now()
	// Setup Directories
	Logger.Debug("Cleaning .regolith/tmp")
	err := os.RemoveAll(".regolith/tmp")
	if err != nil {
		return err
	}

	err = os.MkdirAll(".regolith/tmp", 0777)
	if err != nil {
		return err
	}

	// Copy the contents of the `regolith` folder to `.regolith/tmp`
	Logger.Debug("Copying project files to .regolith/tmp")
	err = copy.Copy("regolith", ".regolith/tmp", copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		return err
	}
	Logger.Debug("Setup done in ", time.Since(start))
	return nil
}

func RunProfile(profileName string) {
	Logger.Info("Running profile: ", profileName)
	project := LoadConfig()
	// The first arg specifies the profile of the manifest used
	profile := project.Profiles[profileName]

	if profile.Unsafe {
		Logger.Info("Warning! Profile flagged as unsafe. Exercise caution!")
	}

	// Check whether filter, that the user wants to run meet the requirements
	checked := map[string]bool{}
	for _, filter := range profile.Filters {
		if filter.RunWith != "" {
			if c, ok := checked[filter.RunWith]; ok || !c {
				if f, ok := FilterTypes[filter.RunWith]; ok {
					checked[filter.RunWith] = true
					f.check()
				} else {
					Logger.Warnf("Filter type '%s' not supported", filter.RunWith)
				}
			}
		}
	}

	err := Setup()
	if err != nil {
		Logger.Fatal("Unable to setup profile")
	}

	//now, we go through the filters!
	for _, filter := range profile.Filters {
		path, _ := filepath.Abs(".")
		RunFilter(filter, path)
	}

	//copy contents of .regolith/tmp to build
	Logger.Info("Moving files to target directory")
	start := time.Now()
	os.RemoveAll("build")
	os.Rename(".regolith/tmp", "build")
	Logger.Debug("Done in ", time.Since(start))

	// copy the build to the target directory
	Logger.Info("Copying build to target directory")
	start = time.Now()
	ExportProject(profile, project.Name)
	Logger.Debug("Done in ", time.Since(start))

	//Done!
	Logger.Info(color.GreenString("Finished"))
}

func GetExportPaths(export_target ExportTarget, name string) (string, string) {
	Logger.Debug(export_target)

	if export_target.Target == "development" {
		com_mojang := FindMojangDir()
		return com_mojang + "/development_behavior_packs/" + name + "_bp", com_mojang + "/development_resource_packs/" + name + "_rp"
	}

	// Throw fatal error that export target isn't valid
	Logger.Fatalf("Export '%s' target not valid", export_target.Target)
	return "", ""
}

func ExportProject(profile Profile, name string) {
	var err error
	export_target := profile.ExportTarget
	bp_path, rp_path := GetExportPaths(export_target, name)

	Logger.Info("Exporting project to ", bp_path)
	Logger.Info("Exporting project to ", rp_path)

	err = copy.Copy("build/BP/", bp_path, copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		Logger.Fatal(color.RedString("Couldn't copy BP files to %s", bp_path), err)
	}

	err = copy.Copy("build/RP/", rp_path, copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		Logger.Fatal(color.RedString("Couldn't copy RP files to %s", rp_path), err)
	}
}

// Runs the filter by selecting the correct filter type and running it
func RunFilter(filter Filter, absoluteLocation string) {
	Logger.Infof("Running filter '%s'", filter.Name)
	start := time.Now()

	if filter.Url != "" {
		RunRemoteFilter(filter.Url, filter.Arguments)
	} else if filter.Filter != "" {
		RunStandardFilter(filter, filter.Arguments)
	} else {
		if f, ok := FilterTypes[filter.RunWith]; ok {
			f.filter(filter, absoluteLocation+string(os.PathSeparator)+filter.Location)
		} else {
			Logger.Warnf("Filter type '%s' not supported", filter.RunWith)
		}
		Logger.Info("Executed in ", time.Since(start))
	}
}

func RunStandardFilter(filter Filter, arguments []string) {
	Logger.Infof("RunStandardFilter '%s'", filter.Filter)
	RunRemoteFilter(FilterNameToUrl(filter.Filter), arguments)
}

func LoadFiltersFromPath(path string) Profile {
	path = path + "/filter.json"
	file, err := ioutil.ReadFile(path)

	if err != nil {
		log.Fatal(color.RedString("Couldn't find %s! Consider running 'regolith install'", path), err)
	}

	var result Profile
	err = json.Unmarshal(file, &result)
	if err != nil {
		log.Fatal(color.RedString("Couldn't load %s: ", path), err)
	}
	return result
}

func RunRemoteFilter(url string, arguments []string) {
	Logger.Infof("RunRemoteFilter '%s'", url)
	if !IsRemoteFilterCached(url) {
		Logger.Error("Filter is not downloaded! Please run 'regolith install'.")
	}

	path := UrlToPath(url)
	absolutePath, _ := filepath.Abs(path)
	for _, filter := range LoadFiltersFromPath(path).Filters {
		RunFilter(filter, absolutePath)
	}
}

func GetAbsoluteWorkingDirectory() string {
	absoluteWorkingDir, _ := filepath.Abs(".regolith/tmp")
	return absoluteWorkingDir
}

func RunSubProcess(command string, args []string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = GetAbsoluteWorkingDirectory()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	if err != nil {
		Logger.Fatal(zap.Error(err))
	}
}
