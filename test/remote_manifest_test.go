package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

func TestManifest(t *testing.T) {
	regolith.InitLogging(true)
	defer os.Chdir(getWdOrFatal(t))

	t.Log("Clearing the test directory...")
	tmpDir := prepareTestDirectory("TestRemoteManifest", t)
	t.Log("Copying the project files into the testing directory...")
	workingDir := filepath.Join(tmpDir, "working-dir")
	copyFilesOrFatal(remoteManifestPath, workingDir, t)
	os.Chdir(workingDir)

	t.Log(tmpDir)

	testUrls := []string{"github.com/akashic-records-of-the-abyss/basic_regolith/nested_filter==1.0.0", "github.com/akashic-records-of-the-abyss/basic_regolith/my_release_filter"}
	var err error

	err = regolith.Install(testUrls, true, false, false, []string{"default"}, false)
	if err != nil {
		t.Fatal("failed to install filter: ", err)
	}

	err = regolith.Run("default", false)

	if err != nil {
		t.Fatal("failed to run filter: ", err)
	}
}
