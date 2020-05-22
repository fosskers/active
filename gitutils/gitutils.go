package gitutils

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
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
func Push(r *git.Repository, remote string, branch string, user string, token string) error {
	src := filepath.Join("refs/heads/", branch)
	spec := config.RefSpec(src + ":" + src)
	return r.Push(&git.PushOptions{
		RemoteName: remote,
		RefSpecs:   []config.RefSpec{spec},
		Auth:       &http.BasicAuth{Username: user, Password: token},
	})
}

// Pull the `master` branch.
func PullMaster(w *git.Worktree, remote string, user string, token string) error {
	return w.Pull(&git.PullOptions{
		RemoteName:    remote,
		ReferenceName: plumbing.Master,
		SingleBranch:  true,
		Auth:          &http.BasicAuth{Username: user, Password: token},
	})
}

// Open a pull request, and return its number.
func PullRequest(c *github.Client, owner string, repo string, branch string) (int, error) {
	new := &github.NewPullRequest{
		Title:               github.String("Github CI Action Updates"),
		Head:                github.String(branch),
		Base:                github.String("master"),
		Body:                github.String("This PR was opened automatically by the `active` tool."),
		MaintainerCanModify: github.Bool(true),
	}
	pr, _, e0 := c.PullRequests.Create(context.Background(), owner, repo, new)

	return *pr.Number, e0
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
// Github API token. Yields the name of the remote, as well as the owner of the
// repo.
func PushableRemote(repo *git.Repository) (string, string, error) {
	rs, e0 := repo.Remotes()
	if e0 != nil {
		return "", "", e0
	}

	chosen := ChooseRemote(rs)
	if chosen == nil {
		return "", "", fmt.Errorf("No remotes found.")
	}

	rc := chosen.Config()
	if rc == nil {
		return "", "", fmt.Errorf("Couldn't fetch RemoteConfig.")
	}

	if len(rc.URLs) == 0 {
		return "", "", fmt.Errorf("Given remote had no URLs!")
	}

	// We don't need to create a new remote; the one given uses HTTPS already.
	if rc.URLs[0][0:5] == "https" {
		owner := strings.Split(rc.URLs[0][8:], "/")[1]
		return rc.Name, owner, nil
	}

	base := "https://" + strings.ReplaceAll(rc.URLs[0][4:], ":", "/")
	owner := strings.Split(base[8:], "/")[1]

	new := config.RemoteConfig{
		Name: "active",
		URLs: []string{base},
	}

	_, e1 := repo.CreateRemote(&new)
	if e1 != nil {
		return "", "", e1
	}

	return "active", owner, nil
}
