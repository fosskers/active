package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fosskers/active/parsing"
	"github.com/fosskers/active/releases"
	"github.com/fosskers/active/utils"
	"github.com/google/go-github/v31/github"
)

var project *string = flag.String("project", ".", "Path to a local clone of a repository.")

// During the lookup of the latest version of an `Action`, we don't want to call
// the Github API more than once per Action. The `seen` map keeps a record of
// lookup attempts.
type Witness struct {
	seen map[string]bool
	mut  sync.Mutex
}

// For Actions that actually had a valid 'latest' release, we store the version
// thereof. This is separate from `Witness`, since all _attempted_ lookups might
// not have had an actual result. Keeping them separate also allows for slightly
// less locking.
type Lookups struct {
	vers map[string]string
	mut  sync.Mutex
}

// If changes were detected for a given workflow file, we want to prompt the
// user for confirmation before applying them. The update detection process is
// concurrent however, and there would be trouble if multiple prompts appeared
// at the same time.
type Terminal struct {
	scan *bufio.Scanner
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
	utils.ExitIfErr(err)
	fmt.Println("Checking the following files:")
	for _, path := range paths {
		fmt.Printf("  --> %s\n", path)
	}

	// Concurrency settings.
	var wg sync.WaitGroup
	witness := Witness{seen: make(map[string]bool)}
	lookups := Lookups{vers: make(map[string]string)}
	terminal := Terminal{scan: bufio.NewScanner(os.Stdin)}

	// Detect and apply updates.
	for _, path := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			old, new, err := update(client, &witness, &lookups, path)
			terminal.mut.Lock()
			defer terminal.mut.Unlock()
			if err != nil {
				fmt.Println(err)
				return
			}
			if old != new {
				fmt.Printf("Updates available for %s:\n\n", path)
				fmt.Printf("Would you like to apply them? [Y/n] ")
				terminal.scan.Scan()
				resp := terminal.scan.Text()
				if resp == "Y" || resp == "y" || resp == "" {
					fmt.Printf("Updating %s...\n", path)
					ioutil.WriteFile(path, []byte(new), 0644)
				} else {
					fmt.Println("Skipping...")
				}
			}
		}(path)
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
		// fmt.Printf("Warning: No 'latest' release found for %s! Skipping...\n", repo)
		return
	}
	l.mut.Lock()
	l.vers[repo] = version
	l.mut.Unlock()
}
