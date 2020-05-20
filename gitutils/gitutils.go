package gitutils

import (
	"context"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
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

// Switch to a given branch.
func Checkout(r *git.Repository, branch string) error {
	w, e0 := r.Worktree()
	if e0 != nil {
		return e0
	}

	e1 := w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + branch),
	})
	if e1 != nil {
		return e1
	}

	return nil
}

// Create a new branch and switch to it, based off the current branch.
func CheckoutCreate(r *git.Repository, branch string) error {
	ref, e0 := r.Head()
	if e0 != nil {
		return e0
	}

	w, e1 := r.Worktree()
	if e1 != nil {
		return e1
	}

	e2 := w.Checkout(&git.CheckoutOptions{
		Hash:   ref.Hash(),
		Branch: plumbing.ReferenceName("refs/heads/" + branch),
		Create: true,
	})
	if e2 != nil {
		return e2
	}

	return nil
}
