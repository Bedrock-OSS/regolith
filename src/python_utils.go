package src

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func GetPythonVersion() *string {
	_, err := exec.LookPath("python")
	if err != nil {
		// Python not found
		return nil
	}
	cmd, _ := exec.Command("python", "--version").Output()
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "Python ")
	return &a
}

func DownloadPythonVersion(version string) {
	// https://www.python.org/ftp/python/3.9.6/python-3.9.6-embed-amd64.zip
	//TODO: It should be downloaded to an actual temp file and removed after unpacking
	out, err := os.Create("temp.zip")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	//TODO: Consider other platforms and architectures than 64 bit Windows
	//TODO: Only Python 3.5.0 and up have embed versions
	resp, err := http.Get(fmt.Sprintf("https://www.python.org/ftp/python/%s/python-%s-embed-amd64.zip", version, version))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Bad status code: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Fatal(err)
	}
}
