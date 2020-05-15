package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fosskers/active/config"
	"github.com/fosskers/active/parsing"
	"github.com/fosskers/active/releases"
	"github.com/fosskers/active/utils"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-github/v31/github"
)

var home, _ = os.UserHomeDir()
var confPath string = filepath.Join(home, ".config/active.yaml")

// Command-line flags.
var localF *bool = flag.Bool("local", false, "Check the local repo you're currently in.")
var tokenF *string = flag.String("token", "", "(optional) Github API OAuth Token.")
var autoF *bool = flag.Bool("y", false, "Automatically apply changes.")
var configPathF *string = flag.String("config", confPath, "Path to config file.")
var pushF *bool = flag.Bool("push", false, "Automatically make commits and open a PR on Github.")

type Project struct {
	name      string
	workflows []Workflow
	repo      *git.Repository
	success   bool // Icky mutable field.
}

// All data pertaining to a fully read and parsed Workflow file.
type Workflow struct {
	path    string // Full filepath to the workflow file.
	yaml    string
	actions []parsing.Action
}

func main() {
	flag.Parse()                             // Collect command-line options.
	c := config.ReadConfig(*configPathF)     // Read the config file.
	client := config.GithubClient(c, tokenF) // Github communication.
	env := config.RuntimeEnv(client)         // Runtime environment.
	paths := getPaths(c)                     // Reading workflow files.

	if len(paths) == 0 {
		fmt.Println("No files to check. Try '--local' or setting your config file.")
		os.Exit(1)
	}

	// Report discovered files.
	longest := 0
	for path := range paths {
		projlen := len(path.project)
		if projlen > longest {
			longest = projlen
		}
	}
	fmt.Println("Checking the following files:")
	for path := range paths {
		spaces := strings.Repeat(" ", longest-len(path.project))
		fmt.Printf("  --> %s: %s%s\n", path.project, spaces, path.name)
	}

	// Perform updates and exit.
	work(env, paths)

	// GIT STUFF
	// TODO Change data representation. `Path`s from the same project
	// need to be kept together, so that we only need to make one commit.
	r, e0 := git.PlainOpen("/home/colin/code/haskell/versions")
	utils.ExitIfErr(e0)
	w, e1 := r.Worktree()
	utils.ExitIfErr(e1)
	_, e3 := w.Add(".github/workflows/ci.yaml")
	utils.ExitIfErr(e3)
	_, e2 := w.Commit("[active] Updating Github Actions", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Colin Woodbury",
			Email: "colin@fosskers.ca",
			When:  time.Now(),
		},
	})
	utils.ExitIfErr(e2)

	fmt.Println("Done.")
}

func allProjects(c *config.Config) []Project {
	return nil
}

// Given a local path to a Git repository, read everything from the filesystem
// that's necessary for further processing.
func project(path string) *Project {
	// If the user has asked for automatic commit pushing, attempt to the open
	// local Git repo.
	var repo *git.Repository
	if *pushF {
		r, e0 := git.PlainOpen(path)
		utils.ExitIfErr(e0)
		repo = r
	}

	// Read and parse all Workflow files.
	wps, e1 := workflows(path)
	utils.ExitIfErr(e1)
	ws := make([]Workflow, 0)
	for _, wp := range wps {
		yaml := readWorkflow(wp)
		actions := parsing.Actions(yaml)
		workflow := Workflow{wp, yaml, actions}
		ws = append(ws, workflow)
	}

	name := filepath.Base(path)
	return &Project{name: name, workflows: ws, repo: repo, success: false}
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
			full := filepath.Join(workflowDir, file.Name())
			fullPaths = append(fullPaths, full)
		}
	}
	return fullPaths, nil
}

// Detect and apply updates.
func work(env *config.Env, paths map[Path]bool) {
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
	ls := env.L.Vers
	for _, workflow := range ws.ws {
		wg.Add(1)
		go func(wf Workflow) {
			defer wg.Done()
			newAs := newActionVers(ls, wf.actions)
			yamlNew := update(newAs, wf.yaml)

			if wf.yaml != yamlNew {
				env.T.Mut.Lock()
				defer env.T.Mut.Unlock()
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
func register(env *config.Env, actions []parsing.Action) {
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
func versionLookup(env *config.Env, a parsing.Action) {
	// Have we looked up this Action already?
	env.W.Mut.Lock()
	repo := a.Repo()
	if seen := env.W.Seen[repo]; seen {
		env.W.Mut.Unlock()
		return
	}
	env.W.Seen[repo] = true
	env.W.Mut.Unlock()

	// Version lookup and recording.
	version, err := releases.Recent(env.C, a.Owner, a.Name)
	if err != nil {
		return
	}
	env.L.Mut.Lock()
	env.L.Vers[repo] = version
	env.L.Mut.Unlock()
}

// For some Actions, what new version should they be assigned to?
func newActionVers(ls map[string]string, actions []parsing.Action) map[parsing.Action]string {
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
func prompt(env *config.Env, path Path, newAs map[parsing.Action]string) bool {
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
	fmt.Printf("\nUpdates available for %s: %s:\n", path.project, path.name)
	for action, v := range newAs {
		repo := action.Repo()
		nameDiff := longestName - len(repo)
		verDiff := longestVer - len(action.Version)
		spaces := strings.Repeat(" ", nameDiff+verDiff+1)
		patt := "  %s" + spaces + "%s --> %s\n"
		fmt.Printf(patt, repo, action.Version, v)
	}
	resp := "NO"
	if !*autoF {
		fmt.Printf("Would you like to apply them? [Y/n] ")
		env.T.Scan.Scan()
		resp = env.T.Scan.Text()
	}
	return *autoF || resp == "Y" || resp == "y" || resp == ""
}
