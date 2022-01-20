package regolith

import (
	"context"

	"github.com/google/go-github/v39/github"
	"golang.org/x/mod/semver"
)

type UpdateStatus struct {
	ShouldUpdate bool
	Url          *string
	Err          *error
}

func CheckUpdate(version string, status chan UpdateStatus) {
	client := github.NewClient(nil)
	// Ignore the error, since it's not critical to regolith
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), "Bedrock-OSS", "regolith")
	if err != nil {
		status <- UpdateStatus{Err: &err}
		return
	}

	status <- UpdateStatus{
		ShouldUpdate: semver.Compare(*release.TagName, version) == 1,
		Url:          release.HTMLURL,
	}
}
