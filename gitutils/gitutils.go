package gitutils

import (
	"context"

	"github.com/google/go-github/v31/github"
)

// Given an activated client and a Github project, look up the version of its
// most recent release.
func Recent(client *github.Client, owner, repo string) (string, error) {
	rel, _, err := client.Repositories.GetLatestRelease(context.Background(), owner, repo)
	if err != nil {
		return "", err
	}
	return versionFormat(rel.GetTagName()), nil
}

// Strip the `v` from the beginning of the tag name.
func versionFormat(version string) string {
	return version[1:]
}
