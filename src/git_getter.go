package src

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type TreeResponse struct {
	Message          string `json:"message"`
	DocumentationUrl string `json:"documentation_url"`
	Sha              string `json:"sha"`
	Url              string `json:"url"`
	Tree             []Leaf `json:"tree"`
	Truncated        bool   `json:"truncated"`
}

type Leaf struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	Sha  string `json:"sha"`
	Url  string `json:"url"`
}

type BlobResponse struct {
	Message          string `json:"message"`
	DocumentationUrl string `json:"documentation_url"`
	Sha              string `json:"sha"`
	NodeId           string `json:"node_id"`
	Size             int64  `json:"size"`
	Url              string `json:"url"`
	Content          string `json:"content"`
}

const treeApiUrl = "https://api.github.com/repos/%s/%s/git/trees/%s"

// DownloadGitHubUrl downloads part of the repository. It will only filter by the first level directory.
// Expects url github.com/owner/repo/folder with host part lowercase
func DownloadGitHubUrl(url string, branch string, localPath string) bool {
	split := strings.Split(path.Clean(url), "/")
	if len(split) < 4 || !strings.HasSuffix(split[0], "github.com") {
		return false
	}
	filterName := split[3]
	data, err := fetchTree(fmt.Sprintf(treeApiUrl, split[1], split[2], branch))
	if err != nil {
		return false
	}
	for _, leaf := range data.Tree {
		if leaf.Path == filterName && leaf.Type == "tree" {
			data, err = fetchTree(leaf.Url + "?recursive=1")
			if err != nil {
				return false
			}
			for _, l := range data.Tree {
				if l.Type == "blob" {
					downloadLeaf(l, localPath)
				}
			}
			return true
		}
	}
	return false
}

// downloadLeaf Downloads file specified by the leaf
func downloadLeaf(l Leaf, localPath string) {
	resp, err := http.Get(l.Url)
	if err != nil {
		Logger.Fatal("Failed to fetch the blob")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	var data BlobResponse
	err = json.Unmarshal(body, &data)
	if data.Message != "" {
		Logger.Fatal(data.Message)
	}
	base64Text := make([]byte, data.Size)
	_, err = base64.StdEncoding.Decode(base64Text, []byte(data.Content))
	dest := filepath.Clean(filepath.Join(localPath, l.Path))
	err = os.MkdirAll(filepath.Dir(dest), os.ModePerm)
	if err != nil {
		Logger.Fatal("Failed to create directories")
	}
	create, err := os.Create(dest)
	if err != nil {
		Logger.Fatal("Failed to create file")
	}
	_, err = create.Write(base64Text)
	if err != nil {
		Logger.Fatal("Failed to write to file")
	}
	err = create.Sync()
	if err != nil {
		Logger.Fatal("Failed to sync the file")
	}
	err = create.Close()
	if err != nil {
		Logger.Fatal("Failed to close the file")
	}
}

// fetchTree fetches tree json
func fetchTree(url string) (TreeResponse, error) {
	resp, err := http.Get(url)
	if err != nil {
		Logger.Fatal("Failed to fetch the tree")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return TreeResponse{}, err
	}
	var data TreeResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return TreeResponse{}, err
	}
	if data.Message != "" {
		return TreeResponse{}, errors.New(data.Message)
	}
	return data, nil
}
