package main

import (
	"flag"
	"fmt"

	"github.com/fosskers/active/releases"
	"github.com/google/go-github/v31/github"
)

var project *string = flag.String("project", ".", "Path to a local clone of a repository.")

func main() {
	// Collect command-line options.
	flag.Parse()

	// Github Communication
	client := github.NewClient(nil)
	version, err := releases.Recent(client, "fosskers", "aura")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(version) // This is it!
	}

	// TODO
	// I need the `ReleaseAsset` type.
	// Or perhaps `RepositoryRelease`?
	// func (s *RepositoriesService) GetLatestRelease(ctx context.Context, owner, repo string) (*RepositoryRelease, *Response, error)

	// Work.
	// releases.Recent("fosskers", "aura")
	fmt.Println("Done.")
}
