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

	"github.com/fatih/color"
	"github.com/fosskers/active/config"
	"github.com/fosskers/active/gitutils"
	"github.com/fosskers/active/parsing"
	"github.com/fosskers/active/utils"
	"github.com/go-git/go-git/v5"
)

// Paths.
var home, _ = os.UserHomeDir()
var confPath string = filepath.Join(home, ".config/active.yaml")

// Command-line flags.
var localF *bool = flag.Bool("local", false, "Check the local repo you're currently in.")
var tokenF *string = flag.String("token", "", "(optional) Github API OAuth Token.")
var autoF *bool = flag.Bool("y", false, "Automatically apply changes.")
var configPathF *string = flag.String("config", confPath, "Path to config file.")
var pushF *bool = flag.Bool("push", false, "Automatically make commits and open a PR on Github.")

// TODO Flag for turning off colours.

// Coloured output.
var cyan = color.New(color.FgCyan).SprintFunc()
var yellow = color.New(color.FgYellow).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()

type Project struct {
	name      string
	workflows []*Workflow
	repo      *git.Repository
	accepted  []string // Mutable field.
}

// All data pertaining to a fully read and parsed Workflow file.
type Workflow struct {
	path    string // Full filepath to the workflow file.
	yaml    string
	actions []parsing.Action
}

func main() {
	flag.Parse()                         // Collect command-line options.
	c := config.ReadConfig(*configPathF) // Read the config file.

	if *pushF && *tokenF == "" && c.Git.Token == "" {
		utils.PrintExit("A real token must be given when using '--push'.")
	}

	client := config.GithubClient(c, tokenF) // Github communication.
	env := config.RuntimeEnv(client)         // Runtime environment.
	projects := allProjects(c)

	// Report discovered files.
	longest := 0
	for _, proj := range projects {
		projlen := len(proj.name)
		if projlen > longest {
			longest = projlen
		}
	}
	fmt.Println("Checking the following files:")
	for _, proj := range projects {
		for _, w := range proj.workflows {
			spaces := strings.Repeat(" ", longest-len(proj.name))
			fmt.Printf("  --> %s: %s%s\n", cyan(proj.name), spaces, filepath.Base(w.path))
		}
	}

	// Register parsed Actions (calls the Github API).
	var wg sync.WaitGroup
	for _, proj := range projects {
		for _, wf := range proj.workflows {
			wg.Add(1)
			go func(w *Workflow) {
				register(env, w.actions)
				wg.Done()
			}(wf)
		}
	}
	wg.Wait()

	// Perform updates concurrently.
	for _, proj := range projects {
		wg.Add(1)
		go func(p *Project) {
			applyUpdates(env, p)
			wg.Done()
		}(proj)
	}
	wg.Wait()

	// Commit and push updates to Github.
	if *pushF {
		for _, proj := range projects {
			if len(proj.accepted) > 0 {
				wg.Add(1)
				go func(p *Project) {
					defer wg.Done()
					e0 := gitutils.Commit(p.repo, c.Git.Name, c.Git.Email, p.accepted)
					if e0 != nil {
						fmt.Printf("Couldn't commit %s: %s\n", cyan(p.name), e0)
						return
					}
					e1 := gitutils.Push(p.repo, c.Git.User, c.Git.Token)
					if e1 != nil {
						fmt.Printf("Unable to push %s to Github: %s\n", cyan(p.name), e1)
						return
					}
					// gitutils.PR(p.repo)
				}(proj)
			}
		}
		wg.Wait()
	}

	fmt.Println("Done.")
}

// Will exit the program if there are no projects to check, or if a specified
// project has no workflow files.
func allProjects(c *config.Config) []*Project {
	if *localF {
		return []*Project{project(".")}
	}

	if len(c.Projects) == 0 {
		utils.PrintExit("No projects to check. Try '--local' or setting your config file.")
	}

	ps := make([]*Project, 0)
	for _, p := range c.Projects {
		ps = append(ps, project(p))
	}
	return ps
}

// Given a local path to a Git repository, read everything from the filesystem
// that's necessary for further processing.
//
// Exits the program if even one file fails to be read, or if there weren't any
// to be read for the given project.
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
	name := filepath.Base(path)
	wps, e1 := workflows(path)
	utils.ExitIfErr(e1)
	if len(wps) == 0 {
		utils.PrintExit("No workflow files detected for " + name)
	}
	ws := make([]*Workflow, 0)
	for _, wp := range wps {
		yaml := readWorkflow(wp)
		actions := parsing.Actions(yaml)
		workflow := Workflow{wp, yaml, actions}
		ws = append(ws, &workflow)
	}

	return &Project{name: name, workflows: ws, repo: repo, accepted: make([]string, 0)}
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
func applyUpdates(env *config.Env, project *Project) {
	// ASSUMPTION: `env.L.Vers` has been fully written to, and will only be read
	// from here on.
	ls := env.L.Vers
	switched := false

	// Apply updates, if the user wants them.
	for _, wf := range project.workflows {
		newAs := newActionVers(ls, wf.actions)
		yamlNew := update(newAs, wf.yaml)

		// Only proceed if there were actually changes to consider.
		if wf.yaml != yamlNew {
			env.T.Mut.Lock()
			defer env.T.Mut.Unlock()
			resp := prompt(env, project.name, wf, newAs)

			if resp {
				if *pushF && !switched {
					e0 := switchBranches(project.repo)
					if e0 != nil {
						fmt.Println(e0)
						fmt.Printf("Skipping %s...\n", cyan(project.name))
						return
					}
					switched = true
				}

				ioutil.WriteFile(wf.path, []byte(yamlNew), 0644)
				fmt.Println("Updated.")

				// Mutability to communicate back to `main` that the user
				// accepted these changes.
				path := filepath.Join(".github/workflows", filepath.Base(wf.path))
				project.accepted = append(project.accepted, path)
			} else {
				fmt.Println("Skipping...")
			}
		}
	}
}

// Switch git branches, if we haven't already. go-git does not
// support stashing, so if the working tree isn't clean, we have
// to skip this Project entirely.
func switchBranches(r *git.Repository) error {
	wt, e9 := r.Worktree()
	if e9 != nil {
		return e9
	}

	status, e8 := wt.Status()
	if e8 != nil {
		return e8
	}

	// TODO Check staged changes as well.
	if !status.IsClean() {
		return fmt.Errorf("The working tree is not clean.")
	}
	e0 := gitutils.Checkout(r, "master")
	if e0 != nil {
		return fmt.Errorf("Unable to switch branches: %s", e0)
	}
	branch := "active/" + time.Now().Format("2006-01-02-15-04-05")
	e1 := gitutils.CheckoutCreate(r, branch)
	if e1 != nil {
		return fmt.Errorf("Unable to create a new branch: %s", e1)
	}
	return nil
}

// Read the workflow file, if we can. Exit otherwise, since the user
// probably wasn't expecting that their file was unreadable.
func readWorkflow(path string) string {
	yamlRaw, err := ioutil.ReadFile(path)
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
	version, err := gitutils.Recent(env.C, a.Owner, a.Name)
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
func prompt(env *config.Env, projName string, workflow *Workflow, newAs map[parsing.Action]string) bool {
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
	fmt.Printf("\nUpdates available for %s: %s:\n", cyan(projName), filepath.Base(workflow.path))
	for action, v := range newAs {
		repo := action.Repo()
		nameDiff := longestName - len(repo)
		verDiff := longestVer - len(action.Version)
		spaces := strings.Repeat(" ", nameDiff+verDiff+1)
		patt := "  %s" + spaces + "%s --> %s\n"
		fmt.Printf(patt, repo, yellow(action.Version), green(v))
	}

	resp := "NO"
	if !*autoF {
		fmt.Printf("Would you like to apply them? [Y/n] ")
		env.T.Scan.Scan()
		resp = env.T.Scan.Text()
	}
	return *autoF || resp == "Y" || resp == "y" || resp == ""
}
