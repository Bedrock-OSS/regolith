package src

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/fatih/color"
	"github.com/plus3it/gorecurcopy"
)

func RunProfile(profileName string) {
	log.Println("Configuration:", profileName)
	var project Project = LoadConfig()
	//setup directories
	os.Mkdir("build", 777)
	os.Mkdir(".tmp", 777)
	//the first arg specifies the profile of the manifest used
	profile := project.Profiles[profileName]
	if profile.Unsafe {
		log.Print(color.YellowString("Warning! Profile flagged as unsafe. Excercise caution"))
	}
	//copy the contents of the `regolith` folder to `.tmp`
	log.Print(color.BlueString("Copying project files to tempdir"))
	gorecurcopy.CopyDirectory("regolith", ".tmp")
	log.Print("done")
	//now, we go through the filters!
	for i, filter := range profile.Filters {
		log.Print(color.CyanString("Running filter %s", filter.Name))
		filterTmpName := fmt.Sprintf("filter_%d", i)
		//copy the filter to .tmp for convience(if it's something like python <filter>.py we kinda need it there...
		datab, _ := ioutil.ReadFile(filter.Location)
		ioutil.WriteFile(path.Join(".tmp", filterTmpName), datab, 777)
		//prepend the file to the filter args
		filter.Arguments = append([]string{filterTmpName}, filter.Arguments...)
		RunSubProcess(filter.RunWith, filter.Arguments, ".tmp")
		os.Remove(path.Join(".tmp", filterTmpName))
	}
	//copy contents of .tmp to build
	log.Print(color.BlueString("copying .tmp to build"))

	gorecurcopy.CopyDirectory(".tmp", "build")
	//Done!
	log.Print(color.GreenString("Finished. built contents -> build/"))
	//cleanup
	os.RemoveAll(".tmp")
}

func RunSubProcess(command string, args []string, workingDir string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = workingDir

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Start()

	scanner := bufio.NewScanner(bytes.NewReader(buf.Bytes()))
	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		m := scanner.Text()
		fmt.Println(m)
	}

	cmd.Wait()

	if err != nil {
		log.Fatal(err)
	}
}
