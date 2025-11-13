package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

// TestManifest tests the install and run commands for filters from a respository that uses the
// regolith_filter_manifest.json file. It installs and runs multiple filters and checks the if the
// output matches the expected results.
func TestManifest(t *testing.T) {
	regolith.InitLogging(true)
	defer os.Chdir(getWdOrFatal(t))

	t.Log("Clearing the test directory...")
	tmpDir := prepareTestDirectory("TestManifest", t)

	t.Log("Copying the project files into the testing directory...")
	workingDir := filepath.Join(tmpDir, "working-dir")

	examplesPath := absOrFatal(remoteManifestPath, t)

	copyFilesOrFatal(
		filepath.Join(examplesPath, "1_start"),
		workingDir, t)
	os.Chdir(workingDir)

	t.Log(tmpDir)
	comandsAndResults := []struct {
		filterUrl      string
		update         bool
		addToProfile   bool
		expectedResult string
	}{
		{
			// regolith install github.com/Bedrock-OSS/regolith-test-filters-manifest/hello-nested-path-filter==1.0.0 --profile
			"github.com/Bedrock-OSS/regolith-test-filters-manifest/hello-nested-path-filter==1.0.0",
			false,
			true,
			"2_install_and_run_nested_1_0_0",
		},
		{
			// regolith install github.com/Bedrock-OSS/regolith-test-filters-manifest/hello-nested-path-filter --update
			"github.com/Bedrock-OSS/regolith-test-filters-manifest/hello-nested-path-filter",
			true,
			false,
			"3_install_and_run_nested_update",
		},
		{
			// regolith install github.com/Bedrock-OSS/regolith-test-filters-manifest/hello-release-filter --profile
			"github.com/Bedrock-OSS/regolith-test-filters-manifest/hello-release-filter",
			false,
			true,
			"4_install_and_run_release",
		},
		{
			// regolith install github.com/Bedrock-OSS/regolith-test-filters-manifest/hello-release-exe-filter==1.0.0 --profile
			"github.com/Bedrock-OSS/regolith-test-filters-manifest/hello-release-exe-filter==1.0.0",
			false,
			true,
			"5_install_and_run_exe_release_1_0_0",
		},
		{
			// regolith install github.com/Bedrock-OSS/regolith-test-filters-manifest/hello-release-exe-filter --update
			"github.com/Bedrock-OSS/regolith-test-filters-manifest/hello-release-exe-filter",
			true,
			false,
			"6_install_and_run_exe_release_update",
		},
	}
	var err error
	for _, step := range comandsAndResults {
		profiles := []string{}
		if step.addToProfile {
			profiles = append(profiles, "default")
		}
		t.Logf("Installing filter: %s", step.filterUrl)
		err = regolith.Install(
			[]string{step.filterUrl},
			step.update,
			false,
			false,
			profiles,
			false,
		)
		if err != nil {
			t.Fatal("Failed to install filter: ", err)
		}
		t.Log("Running Regolith...")
		err = regolith.Run("default", []string{}, false)
		if err != nil {
			t.Fatal("Failed to run Regolith: ", err)
		}
		// TEST EVALUATION
		t.Log("Checking if the result matches the expectations...")
		comparePaths(filepath.Join(examplesPath, step.expectedResult), ".", t, ".regolith")
	}

}

// TestManifestInstallAll tests the install-all command for filters from a respository that uses the
// regolith_filter_manifest.json file.
func TestManifestInstallAll(t *testing.T) {
	regolith.InitLogging(true)
	defer os.Chdir(getWdOrFatal(t))

	t.Log("Clearing the test directory...")
	tmpDir := prepareTestDirectory("TestManifestInstallAll", t)
	t.Log("Copying the project files into the testing directory...")
	workingDir := filepath.Join(tmpDir, "working-dir")

	examplesPath := absOrFatal(remoteManifestInstallAllPath, t)

	copyFilesOrFatal(filepath.Join(examplesPath, "1_start"), workingDir, t)
	os.Chdir(workingDir)

	t.Log(tmpDir)

	err := regolith.InstallAll(true, false, true, false)
	if err != nil {
		t.Fatal("Failed to install filters: ", err)
	}

	// TEST EVALUATION
	t.Log("Checking if the result matches the expectations...")
	comparePaths(filepath.Join(examplesPath, "2_install_all"), ".", t)
}
