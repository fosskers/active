package config

import (
	"bufio"
	"context"
	"io/ioutil"
	"os"
	"sync"

	"github.com/fosskers/active/utils"
	"github.com/google/go-github/v31/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

// Settings read from a config file.
type Config struct {
	Projects []string `yaml:"projects"`
	Git      Git      `yaml:"git"`
}

type Git struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
	User  string `yaml:"user"`
	Token string `yaml:"token"`
}

// During the lookup of the latest version of an `Action`, we don't want to call
// the Github API more than once per Action. The `seen` map keeps a record of
// lookup attempts.
type Witness struct {
	Seen map[string]bool
	Mut  sync.Mutex
}

// For Actions that actually had a valid 'latest' release, we store the version
// thereof. This is separate from `Witness`, since all _attempted_ lookups might
// not have had an actual result. Keeping them separate also allows for slightly
// less locking.
type Lookups struct {
	Vers map[string]string
	Mut  sync.Mutex
}

// If changes were detected for a given workflow file, we want to prompt the
// user for confirmation before applying them. The update detection process is
// concurrent however, and there would be trouble if multiple prompts appeared
// at the same time.
type Terminal struct {
	Scan *bufio.Scanner
	Mut  sync.Mutex
}

// Our global runtime environment. Passing these as a group simplifies a number
// of function calls. Not every function that receives `Env` will need every
// value, but in practice this isn't a problem.
type Env struct {
	C *github.Client
	W *Witness
	L *Lookups
	T *Terminal
}

// Doesn't mind if the expected fields are missing from the config file.
// Default values are supplied if they are missing.
// Exits the program if parsing the config file failed.
func ReadConfig(path string) *Config {
	c := Config{}
	file, e0 := ioutil.ReadFile(path)
	if e0 == nil {
		e1 := yaml.Unmarshal(file, &c)
		utils.ExitIfErr(e1)
	}
	return &c
}

func GithubClient(config *Config, token *string) *github.Client {
	if *token == "" && config.Git.Token == "" {
		return github.NewClient(nil)
	} else {
		tok := *token
		if tok == "" {
			tok = config.Git.Token
		}
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tok})
		tc := oauth2.NewClient(ctx, ts)
		return github.NewClient(tc)
	}
}

// Everything necessary for coordinated concurrency and Github lookups.
func RuntimeEnv(client *github.Client) *Env {
	witness := Witness{Seen: make(map[string]bool)}
	lookups := Lookups{Vers: make(map[string]string)}
	terminal := Terminal{Scan: bufio.NewScanner(os.Stdin)}
	env := Env{client, &witness, &lookups, &terminal}
	return &env
}
