//go:build windows
// +build windows

package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

// TestCustomDevelopmentExportLocation tests if the development export targets
// work as expected. It tesets the "development" target with the "build"
// property set to "standard" it will fail on a computer without
// Minecraft Bedrock Edition installed.
func TestDevelopmentStandardExportLocation(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test on local machine")
	}
	_testCustomDevelopmentExportLocation(
		t,
		func() (string, error) {
			return regolith.FindStandardMojangDir(regolith.PacksPath, false)
		},
		"standard",
		"TestDevelopmentStandardExportLocation")
}

// TestDevelopmentEducationExportLocation tests if the development export
// targets work as expected. It tesets the "development" target with the
// "build" property set to "education" it will fail on a computer without
// Minecraft Bedrock Edition installed.
func TestDevelopmentEducationExportLocation(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test on local machine")
	}
	_testCustomDevelopmentExportLocation(
		t, regolith.FindEducationDir, "education",
		"TestDevelopmentEducationExportLocation")
}

// TestDevelopmentPreviewExportLocation tests if the development export
// targets work as expected. It tesets the "development" target with the
// "build" property set to "preview" it will fail on a computer without
// Minecraft Bedrock Edition Preview installed.
func TestDevelopmentPreviewExportLocation(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test on local machine")
	}
	_testCustomDevelopmentExportLocation(
		t,
		func() (string, error) {
			return regolith.FindPreviewDir(regolith.PacksPath, false)
		},
		"preview",
		"TestDevelopmentPreviewExportLocation")
}

func _testCustomDevelopmentExportLocation(
	t *testing.T, mojangDirGetter func() (string, error), profileToRun string,
	workingDirFolderName string,
) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping test on local machine")
	}
	regolith.InitLogging(true)

	// Switch to current working directory at the end of the test
	defer os.Chdir(getWdOrFatal(t))

	// TEST PREPARATION
	t.Log("Clearing the testing directory...")
	tmpDir := prepareTestDirectory(workingDirFolderName, t)

	t.Log("Copying the project files into the testing directory...")
	copyFilesOrFatal(developmentExportTargets, tmpDir, t)
	os.Chdir(tmpDir)

	// FIND PATH TO com.mojang
	t.Log("Finding the path to com.mojang...")
	mojangDir, err := mojangDirGetter()
	if err != nil {
		t.Fatal(err.Error())
	}

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
	bpPath := filepath.Join(
		mojangDir, "development_behavior_packs", config.Name+"_bp")
	rpPath := filepath.Join(
		mojangDir, "development_resource_packs", config.Name+"_rp")

	// THE TEST
	t.Log("Testing the 'regolith run' command...")
	err = regolith.Run(profileToRun, false, true)
	if err != nil {
		t.Fatal("'regolith run' failed:", err)
	}

	t.Log("Checking if the RP and BP have been exported...")
	assertDirExistsOrFatal(rpPath, t)
	defer os.RemoveAll(rpPath)
	assertDirExistsOrFatal(bpPath, t)
	defer os.RemoveAll(bpPath)
}
