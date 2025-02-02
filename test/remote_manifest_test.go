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

	testUrls := []string{"github.com/akashic-records-of-the-abyss/basic_regolith/nested_filter"}

	err := regolith.Install(testUrls, true, false, false, []string{"default"}, true)
	if err != nil {
		t.Fatal("failed to install filter: ", err)
	}
}
