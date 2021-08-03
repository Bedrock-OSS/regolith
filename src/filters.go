package src

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/plus3it/gorecurcopy"
)

func Setup() {

	// Setup Directories
	os.Mkdir("build", 777)
	os.Mkdir(".tmp", 777)
	os.RemoveAll(".tmp")
	os.RemoveAll("build")

	// Copy the contents of the `regolith` folder to `.tmp`
	log.Print(color.BlueString("Copying project files to tempdir... "))
	gorecurcopy.CopyDirectory("regolith", ".tmp")
	log.Print("done")
}

func RunProfile(profileName string) {
	log.Println("Profile:", profileName)
	var project Project = LoadConfig()
	// The first arg specifies the profile of the manifest used
	profile := project.Profiles[profileName]
	if profile.Unsafe {
		log.Print(color.YellowString("Warning! Profile flagged as unsafe. Exercise caution!"))
	}

	Setup()

	//now, we go through the filters!
	for _, filter := range profile.Filters {
		RunFilter(filter)
	}

	//copy contents of .tmp to build
	log.Print(color.BlueString("copying .tmp to build"))
	os.Rename(".tmp", "build")
	//Done!
	log.Print(color.GreenString("Finished. built contents -> build."))
}

// Runs the filter by selecting the correct filter and running it
func RunFilter(filter Filter) {
	absoluteWorkingDir, _ := filepath.Abs(".tmp")
	switch filter.RunWith {
	case "python":
		RunPythonFilter(filter, absoluteWorkingDir)
	default:
		log.Print(color.RedString("Filter type %s not supported", filter.RunWith))
	}
}

func RunPythonFilter(filter Filter, workingDir string) {
	log.Print(color.CyanString("Running filter %s", filter.Name))
	absoluteLocation, _ := filepath.Abs(filter.Location)
	RunSubProcess(filter.RunWith, append([]string{"-u", absoluteLocation}, filter.Arguments...), workingDir)
}

func RunSubProcess(command string, args []string, workingDir string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}
}
