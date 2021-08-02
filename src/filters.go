package src

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/plus3it/gorecurcopy"
)

func Setup() {

	// Setup Directories
	os.Mkdir("build", 777)
	os.Mkdir(".tmp", 777)
	os.RemoveAll(".tmp")

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
	gorecurcopy.CopyDirectory(".tmp", "build")
	//Done!
	log.Print(color.GreenString("Finished. built contents -> build."))
	// Cleanup
	os.RemoveAll(".tmp")
}

// Runs the filter by selecting the correct filter and running it
func RunFilter(filter Filter) {
	switch filter.RunWith {
	case "python":
		RunPythonFilter(filter)
	default:
		log.Print(color.RedString("Filter type %s not supported", filter.RunWith))
	}
}

func RunPythonFilter(filter Filter) {
	log.Print(color.CyanString("Running filter %s", filter.Name))
	filter.Arguments = append([]string{filter.Location}, filter.Arguments...)
	RunSubProcess(filter.RunWith, filter.Arguments, ".tmp")
}

func RunSubProcess(command string, args []string, workingDir string) {
	fmt.Print(command)
	fmt.Print(args)
	cmd := exec.Command(command, args...)
	cmd.Dir = workingDir

	outputPipe, _ := cmd.StdoutPipe()

	err := cmd.Start()

	scanner := bufio.NewScanner(outputPipe)
	go func() {
		for scanner.Scan() {
			fmt.Printf("\t > %s\n", scanner.Text())
		}
	}()

	cmd.Wait()

	if err != nil {
		log.Fatal(err)
	}
}
