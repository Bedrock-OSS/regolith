//go:build windows
// +build windows

package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bedrock-oss.github.com/regolith/regolith"
	"github.com/otiai10/copy"
	"golang.org/x/sys/windows"
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
		t.Fatal("Unable to get current working directory")
	}
	defer os.Chdir(wd)
	// Find path to com.mojang
	mojangDir, err := regolith.FindMojangDir()
	if err != nil {
		t.Fatal(err.Error())
	}
	// The project will be tested from C:/regolithtestProject (or whatever
	// drive you use for Minecraft) Don't change that to ioutil.TmpDir.
	// Current implementation assures that the working dir will be on the
	// same drive as Minecraft which is crucial for this test.
	sep := string(filepath.Separator)
	workingDir := filepath.Join(
		// https://github.com/golang/go/issues/26953
		strings.Split(mojangDir, sep)[0]+sep,
		"regolithTestProject")
	if _, err := os.Stat(workingDir); err == nil { // The path SHOULDN'T exist
		t.Fatalf("Clear path %q before testing", workingDir)
	}
	// Copy the test project to the working directory
	err = copy.Copy(
		minimalProjectPath,
		workingDir,
		copy.Options{PreserveTimes: false, Sync: false},
	)
	if err != nil {
		t.Fatalf(
			"Failed to copy test files %q into the working directory %q",
			minimalProjectPath, workingDir,
		)
	}
	// Before "workingDir" the working dir of this test can't be there
	defer os.RemoveAll(workingDir)
	defer os.Chdir(wd)
	// Switch wd to wrokingDir
	os.Chdir(workingDir)
	// Get the name of the config from config
	configJson, err := regolith.LoadConfigAsMap()
	if err != nil {
		t.Fatal(err.Error())
	}
	config, err := regolith.ConfigFromObject(configJson)
	if err != nil {
		t.Fatal(err.Error())
	}

	bpPath := filepath.Join(
		mojangDir, "development_behavior_packs", config.Name+"_bp")
	rpPath := filepath.Join(
		mojangDir, "development_resource_packs", config.Name+"_rp")
	os.Chdir(workingDir)
	// THE TEST
	err = regolith.Run("dev", true)
	if err != nil {
		t.Fatal("'regolith init' failed:", err)
	}
	// Test if the RP and BP were created in the right paths
	assertDirExists := func(dir string) {
		if stats, err := os.Stat(dir); err != nil {
			t.Fatalf("Unable to get stats of %q", dir)
		} else if !stats.IsDir() {
			t.Fatalf("Created path %q is not a directory", dir)
		}
	}
	assertDirExists(rpPath)
	defer os.RemoveAll(rpPath)
	assertDirExists(bpPath)
	defer os.RemoveAll(bpPath)
	// Compare the permissions of the mojang path with the permissions of RP
	// and BP

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
	mojangAcl := getSecurityString(mojangDir)
	assertValidAcl := func(dir string) {
		if acl := getSecurityString(dir); acl != mojangAcl {
			t.Fatalf(
				"Permission settings of the pack and com.mojang are different:"+
					"\n\n%q:\n%s\n\n\n\n%q:\n%s",
				dir, acl, mojangDir, mojangAcl)
		}
	}
	assertValidAcl(rpPath)
	assertValidAcl(bpPath)
}
