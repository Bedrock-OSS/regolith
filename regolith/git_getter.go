package regolith

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v39/github"
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
func DownloadGitHubUrl(url string, localPath string) (bool, error) {
	split := strings.Split(path.Clean(url), "/")
	if len(split) < 4 || !strings.HasSuffix(split[0], "github.com") {
		return false, wrapError("Remote repositories must be of the form: github.com/owner/repo/folder", nil)
	}
	client := github.NewClient(nil)
	repo, _, err := client.Repositories.Get(context.Background(), split[1], split[2])
	if err != nil {
		return false, wrapError("Failed to get the repository information", err)
	}
	branch := repo.DefaultBranch
	filterName := split[3]
	data, err := fetchTree(fmt.Sprintf(treeApiUrl, split[1], split[2], *branch))
	if err != nil {
		return false, wrapError("Failed to fetch the tree", err)
	}
	if data.Message != "" {
		Logger.Debugf("GitHub API responded with %s", data.Message)
		return false, nil
	}
	for _, leaf := range data.Tree {
		if leaf.Path == filterName && leaf.Type == "tree" {
			data, err = fetchTree(leaf.Url + "?recursive=1")
			if err != nil {
				return false, wrapError(fmt.Sprintf("Failed to fetch the tree %s", leaf.Path), err)
			}
			if data.Message != "" {
				return false, wrapError(fmt.Sprintf("Failed to fetch the tree %s: %s", leaf.Path, data.Message), err)
			}
			for _, l := range data.Tree {
				if l.Type == "blob" {
					err := downloadLeaf(l, localPath)
					if err != nil {
						return false, wrapError(fmt.Sprintf("Failed to download the leaf %s", l.Path), err)
					}
				}
			}
			return true, nil
		}
	}
	return false, nil
}

// downloadLeaf Downloads file specified by the leaf
func downloadLeaf(l Leaf, localPath string) error {
	resp, err := http.Get(l.Url)
	if err != nil {
		return errors.New("Failed to fetch the blob")
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var data BlobResponse
	err = json.Unmarshal(body, &data)
	if data.Message != "" {
		return errors.New(data.Message)
	}
	base64Text := make([]byte, data.Size)
	_, err = base64.StdEncoding.Decode(base64Text, []byte(data.Content))
	dest := filepath.Clean(filepath.Join(localPath, l.Path))
	err = os.MkdirAll(filepath.Dir(dest), os.ModePerm)
	if err != nil {
		return wrapError("Failed to create directories", err)
	}
	create, err := os.Create(dest)
	if err != nil {
		return wrapError("Failed to create file", err)
	}
	_, err = create.Write(base64Text)
	if err != nil {
		return wrapError("Failed to write to file", err)
	}
	err = create.Sync()
	if err != nil {
		return wrapError("Failed to sync the file", err)
	}
	err = create.Close()
	if err != nil {
		return wrapError("Failed to close the file", err)
	}
	return nil
}

// fetchTree fetches tree json
func fetchTree(url string) (*TreeResponse, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, wrapError("Failed to fetch the tree", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var data *TreeResponse
	err = json.Unmarshal(body, &data)
	return data, nil
}
