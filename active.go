package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/fosskers/active/parsing"
	"github.com/fosskers/active/releases"
	"github.com/fosskers/active/utils"
	"github.com/google/go-github/v31/github"
)

var project *string = flag.String("project", ".", "Path to a local clone of a repository.")

type Witness struct {
	seen map[parsing.Action]bool
	mut  sync.Mutex
}

type Lookups struct {
	vers map[parsing.Action]string
	mut  sync.Mutex
}

func main() {
	// Collect command-line options.
	flag.Parse()

	// Github communication.
	// TODO Auth support.
	client := github.NewClient(nil)

	// Reading workflow files.
	paths, err := workflows(*project)
	utils.Check(err)

	// Concurrency settings
	var wg sync.WaitGroup
	witness := Witness{seen: make(map[parsing.Action]bool)}
	lookups := Lookups{vers: make(map[parsing.Action]string)}

	// Detect updates.
	for _, path := range paths {
		fmt.Println(path)
		wg.Add(1)
		go func(path string) {
			update(client, &witness, &lookups, path)
			wg.Done()
		}(path)
		// TODO Print out a diff of the changes.
	}
	wg.Wait()
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
func update(client *github.Client, witness *Witness, lookups *Lookups, path string) {
	// Read the workflow file.
	yamlRaw, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("Couldn't read %s. Skipping...\n", path)
		return
	}

	// Parse the workflow file.
	yaml := string(yamlRaw)
	actions := parsing.Actions(yaml)

	// Concurrently query the Github API for Action versions.
	var wg sync.WaitGroup
	for _, action := range actions {
		wg.Add(1)
		go func(action parsing.Action) {
			versionLookup(client, witness, lookups, action)
			wg.Done()
		}(action)
	}
	wg.Wait()
}

// Concurrently query the Github API for Action versions.
func versionLookup(client *github.Client, witness *Witness, lookups *Lookups, action parsing.Action) {
	// Have we looked up this Action already?
	witness.mut.Lock()
	if seen := witness.seen[action]; seen {
		witness.mut.Unlock()
		return
	}
	witness.seen[action] = true
	witness.mut.Unlock()

	// Version lookup and recording.
	version, err := releases.Recent(client, action.Owner, action.Name)
	if err != nil {
		fmt.Printf("No 'latest' release found for %s! Skipping...\n", action.Repo())
		return
	}
	lookups.mut.Lock()
	lookups.vers[action] = version
	lookups.mut.Unlock()

	// TODO Remove and do this elsewhere.
	if action.Version != version {
		fmt.Printf("%s: %s to %s\n", action.Repo(), action.Version, version)
	}
}
