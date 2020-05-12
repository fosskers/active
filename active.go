package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fosskers/active/parsing"
	"github.com/fosskers/active/releases"
	"github.com/fosskers/active/utils"
	"github.com/google/go-github/v31/github"
)

var project *string = flag.String("project", ".", "Path to a local clone of a repository.")

type Witness struct {
	seen map[string]bool
	mut  sync.Mutex
}

type Lookups struct {
	vers map[string]string
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
	witness := Witness{seen: make(map[string]bool)}
	lookups := Lookups{vers: make(map[string]string)}

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
func update(c *github.Client, w *Witness, l *Lookups, path string) (string, string, error) {
	// Read the workflow file.
	yamlRaw, err := ioutil.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	// Parse the workflow file.
	yaml := string(yamlRaw)
	actions := parsing.Actions(yaml)

	// Concurrently query the Github API for Action versions.
	var wg sync.WaitGroup
	for _, action := range actions {
		wg.Add(1)
		go func(action parsing.Action) {
			versionLookup(c, w, l, action)
			wg.Done()
		}(action)
	}
	wg.Wait()

	// Look for version discrepancies.
	l.mut.Lock()
	ls := l.vers // Grab a quick read-only copy.
	l.mut.Unlock()
	yamlNew := yaml
	dones := make(map[string]bool) // Don't do the find-and-replace more than once.
	for _, action := range actions {
		repo := action.Repo()
		if v, done := ls[repo], dones[repo]; !done && v != "" && action.Version != v {
			// fmt.Printf("%s: %s to %s\n", repo, action.Version, v)
			old := "uses: " + action.Raw()
			new := "uses: " + repo + "@v" + v
			yamlNew = strings.ReplaceAll(yamlNew, old, new)
			dones[repo] = true
		}
	}

	return yaml, yamlNew, nil
}

// Concurrently query the Github API for Action versions.
func versionLookup(c *github.Client, w *Witness, l *Lookups, a parsing.Action) {
	// Have we looked up this Action already?
	w.mut.Lock()
	repo := a.Repo()
	if seen := w.seen[repo]; seen {
		w.mut.Unlock()
		return
	}
	w.seen[repo] = true
	w.mut.Unlock()

	// Version lookup and recording.
	version, err := releases.Recent(c, a.Owner, a.Name)
	if err != nil {
		fmt.Printf("No 'latest' release found for %s! Skipping...\n", repo)
		return
	}
	l.mut.Lock()
	l.vers[repo] = version
	l.mut.Unlock()
}
