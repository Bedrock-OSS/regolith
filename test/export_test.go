package test

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"bedrock-oss.github.com/regolith/regolith"
	"github.com/otiai10/copy"
)

// TestMoveFilesAcl tests for issue #85. It creates a project on the same drive
// as the drive used to store Minecraft files and runs Regolith with
// development export target, then it checks the permissions of the newly
// created packs. If they're not the same as the permissions of
// development_*_packs folders that contain them, the test fails.
// To compare permissions, this function uses "icacls.exe"
func TestMoveFilesAcl(t *testing.T) {
	// Switching working directories in this test, make sure to go back
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Unable to get current working directory")
	}
	defer os.Chdir(wd)
	// Find path to com.mojang
	mojangDir, err := regolith.FindMojangDir()
	if err != nil {
		log.Fatal(err.Error())
	}
	// The project will be tested from C:/regolithtestProject (or whatever
	// drive you use for Minecraft)
	sep := string(filepath.Separator)
	workingDir := filepath.Join(
		// https://github.com/golang/go/issues/26953
		strings.Split(mojangDir, sep)[0]+sep,
		"regolithTestProject")
	if _, err := os.Stat(workingDir); err == nil { // The path SHOULDN'T exist
		log.Fatalf("Clear path %q before testing", workingDir)
	}
	// Copy the test project to the working directory
	err = copy.Copy(
		minimalProjectPath,
		workingDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	if err != nil {
		log.Fatalf(
			"Failed to copy test files %q into the working directory %q",
			minimalProjectPath, workingDir,
		)
	}
	// Before "workingDir" the working dir of this test can't be there
	defer os.RemoveAll(workingDir)
	defer os.Chdir(wd)
	// Switch wd to wrokingDir
	os.Chdir(workingDir)
	// Get the name of the project from config
	project, err := regolith.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load project config: %s", err)
	}
	bpPath := filepath.Join(
		mojangDir, "development_behavior_packs", project.Name+"_bp")
	rpPath := filepath.Join(
		mojangDir, "development_resource_packs", project.Name+"_rp")
	// Run regolith project
	os.Chdir(workingDir)
	regolith.InitLogging(true)
	regolith.RunProfile("dev")

	// Test if the RP and BP were created in the right paths
	assertDirExists := func(dir string) {
		if stats, err := os.Stat(dir); err != nil {
			log.Fatalf("Unable to get stats of %q", dir)
		} else if !stats.IsDir() {
			log.Fatalf("Created path %q is not a directory", dir)
		}
	}
	assertDirExists(rpPath)
	defer os.RemoveAll(rpPath)
	assertDirExists(bpPath)
	defer os.RemoveAll(bpPath)
	// Compare the permissions of the mojang path with the permissions of RP
	// and BP
	getAclPermissions := func(dir string) string {
		result := bytes.NewBufferString("")
		cmd := exec.Command("icacls", ".")
		cmd.Dir = dir
		cmd.Stdout = result
		// cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("icacls.exe execution failed in %q with error:\n%s", dir, err)
		}
		return result.String()
	}
	mojangAcl := getAclPermissions(mojangDir)
	// This solution is an awful hack :(
	// com.mojang has additional property which must be removed so the
	// string comparison below will match
	mojangAcl = strings.Replace(
		mojangAcl, "  Mandatory Label\\Low Mandatory Level:(NW)\n", "", -1)
	assertValidAcl := func(dir string) {
		if acl := getAclPermissions(dir); acl != mojangAcl {
			log.Fatalf(
				"Permissions of the pack and com.mojang are different:"+
					"\n===============\n%s:\n%s\n\n===============\n%s:\n%s"+
					"===============",
				dir, acl, mojangDir, mojangAcl)
		}
	}
	assertValidAcl(rpPath)
	assertValidAcl(bpPath)
}
