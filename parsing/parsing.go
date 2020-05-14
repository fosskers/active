package parsing

import (
	"strings"
)

type Action struct {
	Owner   string
	Name    string
	Version string
}

// The full `owner/repo@vN` format.
func (a Action) Raw() string {
	return a.Repo() + "@v" + a.Version
}

// The `owner/repo` format.
func (a Action) Repo() string {
	return a.Owner + "/" + a.Name
}

// Given the contents of a workflow YAML file, find all uses of a Github Action.
func Actions(file string) []Action {
	lines := strings.Split(file, "\n")
	actions := make([]Action, 0, len(lines))
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "uses:") {
			action := parseAction(l)
			actions = append(actions, action)
		}
	}
	return actions
}

// Form an `Action`, given a line like:
//
//      uses: actions/checkout@v2
func parseAction(line string) Action {
	dropped := line[6:]
	owner := strings.SplitN(dropped, "/", 2)
	name := strings.SplitN(owner[1], "@", 2)
	return Action{Owner: owner[0], Name: name[0], Version: name[1][1:]}
}
