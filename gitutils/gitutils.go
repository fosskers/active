package gitutils

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
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

// Commit the changes in some given filepaths.
func Commit(r *git.Repository, name string, email string, files []string) error {
	w, e0 := r.Worktree()
	if e0 != nil {
		return e0
	}
	for _, file := range files {
		_, e1 := w.Add(file)
		if e1 != nil {
			return e1
		}
	}
	_, e2 := w.Commit("[active] Updating Github Actions", &git.CommitOptions{
		Author: &object.Signature{Name: name, Email: email, When: time.Now()},
	})
	if e2 != nil {
		return e2
	}

	return nil
}

// Push the given branch.
func Push(r *git.Repository, branch string, user string, token string) error {
	src := filepath.Join("refs/heads/", branch)
	spec := config.RefSpec(src + ":" + src)
	return r.Push(&git.PushOptions{
		RefSpecs: []config.RefSpec{spec},
		Auth:     &http.BasicAuth{Username: user, Password: token},
	})
}

func PullRequest(client *github.Client, owner string, repo string, branch string) error {
	pr := &github.NewPullRequest{
		Title:               github.String("Github CI Action Updates"),
		Head:                github.String(branch),
		Base:                github.String("master"),
		Body:                github.String("This PR was opened automatically by the `active` tool."),
		MaintainerCanModify: github.Bool(true),
	}
	_, _, e0 := client.PullRequests.Create(context.Background(), owner, repo, pr)

	return e0
}

// Pick a suitable "remote" to push to later, defaulting to one that was created
// previously by this tool.
func ChooseRemote(rs []*git.Remote) *git.Remote {
	for _, r := range rs {
		if r.Config().Name == "active" {
			return r
		}
	}
	for _, r := range rs {
		if r.Config().Name == "origin" {
			return r
		}
	}
	if len(rs) > 0 {
		return rs[0]
	}
	return nil
}

// Fetch or create an HTTP-based remote that we can used to push via the given
// Github API token.
func PushableRemote(repo *git.Repository) (string, error) {
	rs, e0 := repo.Remotes()
	if e0 != nil {
		return "", e0
	}

	chosen := ChooseRemote(rs)
	if chosen == nil {
		return "", fmt.Errorf("No remotes found.")
	}

	config := chosen.Config()
	if config == nil {
		return "", fmt.Errorf("Couldn't fetch RemoteConfig.")
	}

	// We had already created a usable remote.
	if config.Name == "active" {
		return "active", nil
	}

	thing := config.RemoteConfig{
		Name: "active",
		URLs: []string{},
	}

	_, e1 := repo.CreateRemote(&thing)
	if e1 != nil {
		return "", e1
	}

	return "active", nil
}
