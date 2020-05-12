package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/fosskers/active/parsing"
	"github.com/fosskers/active/releases"
	"github.com/fosskers/active/utils"
	"github.com/google/go-github/v31/github"
)

var project *string = flag.String("project", ".", "Path to a local clone of a repository.")

func main() {
	// Collect command-line options.
	flag.Parse()

	// Github communication.
	// TODO Auth support.
	client := github.NewClient(nil)

	// Reading workflow files.
	paths, err := workflows(*project)
	utils.Check(err)

	// Detect updates.
	for _, path := range paths {
		fmt.Println(path)
		update(client, path)
		// TODO Print out a diff of the changes.
	}
	fmt.Println("Done.")
}

// Given a local path to a code repository, find the paths of all its Github
// workflow configuration files.
func workflows(project string) ([]string, error) {
	workflowDir := filepath.Join(project, ".github/workflows")
	items, err := ioutil.ReadDir(workflowDir)
	if err != nil {
		return nil, err
	}
	fullPaths := make([]string, 0, len(items))
	for _, file := range items {
		if !file.IsDir() {
			fullPaths = append(fullPaths, filepath.Join(workflowDir, file.Name()))
		}
	}
	return fullPaths, nil
}

// Read a workflow file, detect its actions, and update them if necessary.
func update(client *github.Client, path string) {
	yamlRaw, err := ioutil.ReadFile(path)
	utils.Check(err)
	yaml := string(yamlRaw)
	actions := parsing.Actions(yaml)
	for _, action := range actions {
		version, err := releases.Recent(client, action.Owner, action.Name)
		if err != nil {
			fmt.Printf("No 'latest' release found for %s! Skipping...\n", action.Repo())
		} else if action.Version != version {
			fmt.Printf("%s: %s to %s\n", action.Repo(), action.Version, version)
		}
	}
}
