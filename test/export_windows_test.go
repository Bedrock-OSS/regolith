//go:build windows
// +build windows

package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
	"golang.org/x/sys/windows"
)

// TestMoveFilesAcl tests for issue #85. It creates a project on the same drive
// as the drive used to store Minecraft files and runs Regolith with
// development export target, then it checks the permissions of the newly
// created packs. If they're not the same as the permissions of
// development_*_packs folders that contain them, the test fails.
// To compare permissions, this function uses "icacls.exe"
func TestMoveFilesAcl(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test on local machine")
	}
	// Switch to current working directory at the end of the test
	defer os.Chdir(getWdOrFatal(t))

	// TEST PREPARATION
	// Switching working directories in this test, make sure to go back
	originalWd := getWdOrFatal(t)
	defer os.Chdir(originalWd)

	// Find path to com.mojang
	t.Log("Finding the path to com.mojang...")
	mojangDir, err := regolith.FindMojangDir()
	if err != nil {
		t.Fatal(err.Error())
	}

	// The project will be tested from C:/regolithTestProject (or whatever
	// drive you use for Minecraft) Don't change that to ioutil.TmpDir.
	// Current implementation assures that the working dir will be on the
	// same drive as Minecraft which is crucial for this test.
	t.Log("Preparing the working directory in C:/regolithTestProject...")
	sep := string(filepath.Separator)
	workingDir := filepath.Join(
		// https://github.com/golang/go/issues/26953
		strings.Split(mojangDir, sep)[0]+sep,
		"regolithTestProject")
	if _, err := os.Stat(workingDir); err == nil { // The path SHOULDN'T exist
		t.Fatalf(
			"Clear path for this test manually before testing.\n"+
				"Path: %s",
			workingDir)
	}

	t.Log("Copying the project files into the testing directory...")
	copyFilesOrFatal(minimalProjectPath, workingDir, t)

	// Change to the original working directory before defered RemoveAll
	// because otherwise RemoveAll will fail (beacuse of the path being
	// used)
	defer os.RemoveAll(workingDir)
	defer os.Chdir(originalWd)

	// Switch to workingDir
	os.Chdir(workingDir)

	// LOAD DATA FROM CONFIG
	// Get the name of the project from config
	t.Log("Loading the data from config, befor running the test...")
	configJson, err := regolith.LoadConfigAsMap()
	if err != nil {
		t.Fatal(err.Error())
	}
	config, err := regolith.ConfigFromObject(configJson)
	if err != nil {
		t.Fatal(err.Error())
	}
	bpPath := filepath.Join(mojangDir, "development_behavior_packs", config.Name+"_bp")
	rpPath := filepath.Join(mojangDir, "development_resource_packs", config.Name+"_rp")

	// THE TEST
	t.Log("Testing the 'regolith run' command...")
	err = regolith.Run("dev", true)
	if err != nil {
		t.Fatal("'regolith run' failed:", err)
	}

	t.Log("Checking if the RP and BP have been exported...")
	assertDirExistsOrFatal(rpPath, t)
	defer os.RemoveAll(rpPath)
	assertDirExistsOrFatal(bpPath, t)
	defer os.RemoveAll(bpPath)

	t.Log("Checking if the permissions of the exported packs are correct...")
	// getSecurityString gets the string representation of the path security
	// info of the path
	getSecurityString := func(path string) string {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Unable to get the file stats of %q", path)
		}
		securityInfo, err := windows.GetNamedSecurityInfo(
			path,
			windows.SE_FILE_OBJECT,
			windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			t.Fatalf("Unable to get security info about %q", path)
		}
		// dacl, defaulted, err := securityInfo.DACL()
		return securityInfo.String()
	}

	t.Log("Getting the security settings of the com.mojang directory...")
	mojangAcl := getSecurityString(mojangDir)
	assertValidAcl := func(dir string) {
		if acl := getSecurityString(dir); acl != mojangAcl {
			t.Fatalf(
				"Permission settings of the pack and com.mojang are different:"+
					"\n\n%q:\n%s\n\n\n\n%q:\n%s",
				dir, acl, mojangDir, mojangAcl)
		}
	}
	t.Log("Comparing the security settings of com.mojang with the exported packs...")
	assertValidAcl(rpPath)
	assertValidAcl(bpPath)
}
