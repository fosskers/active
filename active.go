package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"

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

	// Reading workflow files.
	files, err := workflows(*project)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, file := range files {
		fmt.Println(file)
		yaml, _ := ioutil.ReadFile(file)
		fmt.Printf("%s\n", yaml)
	}
	// releases.Recent("fosskers", "aura")
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
