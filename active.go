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

	"github.com/fosskers/active/config"
	"github.com/fosskers/active/parsing"
	"github.com/fosskers/active/releases"
	"github.com/fosskers/active/utils"
	"github.com/google/go-github/v31/github"
)

var project *string = flag.String("project", ".", "Path to a local clone of a repository.")
var token *string = flag.String("token", "", "(optional) Github API OAuth Token.")
var auto *bool = flag.Bool("y", false, "Automatically apply changes.")

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

// Our global runtime environment. Passing these as a group simplifies a number
// of function calls. Not every function that receives `Env` will need every
// value, but in practice this isn't a problem.
type Env struct {
	c *github.Client
	w *Witness
	l *Lookups
	t *Terminal
}

// A richer representation of a filepath to a workflow file.
type Path struct {
	project string // The name of the repository.
	name    string // The base name of the file.
	full    string // The full filepath.
}

// All data pertaining to a fully read and parsed Workflow file.
type Workflow struct {
	path    Path
	yaml    string
	actions []parsing.Action
}

// A grouping of `Workflow` files for concurrent reads/writes.
type Workflows struct {
	ws  []Workflow
	mux sync.Mutex
}

func main() {
	// Collect command-line options.
	flag.Parse()

	// Read the config file.
	c, e0 := config.ReadConfig()
	utils.ExitIfErr(e0)

	// Github communication.
	client := config.GithubClient(c, token)

	// Reading workflow files.
	paths := make(map[Path]bool)
	for _, proj := range c.Projects {
		ps, e1 := workflows(proj)
		utils.ExitIfErr(e1)
		for _, p := range ps {
			paths[p] = true
		}
	}
	// paths, err := workflows(*project)
	// utils.ExitIfErr(err)
	fmt.Println("Checking the following files:")
	// TODO Alignment of project names.
	for path := range paths {
		fmt.Printf("  --> %s: %s\n", path.project, path.name)
	}

	// Runtime configuration. Mostly for coordinating concurrency.
	witness := Witness{seen: make(map[string]bool)}
	lookups := Lookups{vers: make(map[string]string)}
	terminal := Terminal{scan: bufio.NewScanner(os.Stdin)}
	env := Env{client, &witness, &lookups, &terminal}

	// Perform updates and exit.
	work(&env, paths)
	fmt.Println("Done.")
}

// Given a local path to a code repository, find the paths of all its Github
// workflow configuration files.
func workflows(project string) ([]Path, error) {
	workflowDir := filepath.Join(project, ".github/workflows")
	items, err := ioutil.ReadDir(workflowDir)
	if err != nil {
		return nil, err
	}
	fullPaths := make([]Path, 0, len(items))
	for _, file := range items {
		if !file.IsDir() {
			proj := filepath.Base(project)
			name := filepath.Base(file.Name())
			full := filepath.Join(workflowDir, file.Name())
			fullPaths = append(fullPaths, Path{proj, name, full})
		}
	}
	return fullPaths, nil
}

// Detect and apply updates.
func work(env *Env, paths map[Path]bool) {
	var wg sync.WaitGroup
	ws := Workflows{ws: make([]Workflow, 0)}

	// Parse all files and synchronize on the Actions.
	for path := range paths {
		wg.Add(1)
		go func(pth Path) {
			defer wg.Done()
			yaml := readWorkflow(pth)
			actions := parsing.Actions(yaml)
			register(env, actions)
			ws.mux.Lock()
			ws.ws = append(ws.ws, Workflow{pth, yaml, actions})
			ws.mux.Unlock()
		}(path)
	}
	wg.Wait()

	// Apply updates, if the user wants them.
	ls := env.l.vers
	for _, workflow := range ws.ws {
		wg.Add(1)
		go func(wf Workflow) {
			defer wg.Done()
			newAs := newActionVersions(ls, wf.actions)
			yamlNew := update(newAs, wf.yaml)

			if wf.yaml != yamlNew {
				env.t.mut.Lock()
				defer env.t.mut.Unlock()
				resp := prompt(env, wf.path, newAs)
				if resp {
					ioutil.WriteFile(wf.path.full, []byte(yamlNew), 0644)
					fmt.Println("Updated.")
				} else {
					fmt.Println("Skipping...")
				}
			}
		}(workflow)
	}
	wg.Wait()
}

// Read the workflow file, if we can. Exit otherwise, since the user
// probably wasn't expecting that their file was unreadable.
func readWorkflow(path Path) string {
	yamlRaw, err := ioutil.ReadFile(path.full)
	utils.ExitIfErr(err)
	return string(yamlRaw)
}

// Given some Actions, call the Github API and check for their latest versions.
func register(env *Env, actions []parsing.Action) {
	var wg sync.WaitGroup
	for _, action := range actions {
		wg.Add(1)
		go func(action parsing.Action) {
			versionLookup(env, action)
			wg.Done()
		}(action)
	}
	wg.Wait()
}

// Concurrently query the Github API for Action versions.
func versionLookup(env *Env, a parsing.Action) {
	// Have we looked up this Action already?
	env.w.mut.Lock()
	repo := a.Repo()
	if seen := env.w.seen[repo]; seen {
		env.w.mut.Unlock()
		return
	}
	env.w.seen[repo] = true
	env.w.mut.Unlock()

	// Version lookup and recording.
	version, err := releases.Recent(env.c, a.Owner, a.Name)
	if err != nil {
		return
	}
	env.l.mut.Lock()
	env.l.vers[repo] = version
	env.l.mut.Unlock()
}

// For some Actions, what new version should they be assigned to?
func newActionVersions(ls map[string]string, actions []parsing.Action) map[parsing.Action]string {
	news := make(map[parsing.Action]string)
	for _, action := range actions {
		if v := ls[action.Repo()]; v != "" && action.Version != v {
			news[action] = v
		}
	}
	return news
}

// Given the Actions detected in some workflow file, try to replace them with
// the newest versions available from Github.
func update(actions map[parsing.Action]string, yaml string) string {
	yamlNew := yaml
	for action, v := range actions {
		old := "uses: " + action.Raw()
		new := "uses: " + action.Repo() + "@v" + v
		yamlNew = strings.ReplaceAll(yamlNew, old, new)
	}
	return yamlNew
}

// We detected some changes to a workflow file, so we inform the user and ask
// whether we should write the changes to disk.
func prompt(env *Env, path Path, newAs map[parsing.Action]string) bool {
	longestName := 0
	longestVer := 0
	for action := range newAs {
		if repo := action.Repo(); len(repo) > longestName {
			longestName = len(repo)
		}
		if len(action.Version) > longestVer {
			longestVer = len(action.Version)
		}
	}
	fmt.Printf("Updates available for %s: %s:\n", path.project, path.name)
	for action, v := range newAs {
		repo := action.Repo()
		nameDiff := longestName - len(repo)
		verDiff := longestVer - len(action.Version)
		spaces := strings.Repeat(" ", nameDiff+verDiff+1)
		patt := "  %s" + spaces + "%s --> %s\n"
		fmt.Printf(patt, repo, action.Version, v)
	}
	resp := "NO"
	if !*auto {
		fmt.Printf("Would you like to apply them? [Y/n] ")
		env.t.scan.Scan()
		resp = env.t.scan.Text()
	}
	return *auto || resp == "Y" || resp == "y" || resp == ""
}
