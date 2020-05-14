package config

import (
	"context"
	"io/ioutil"

	"github.com/fosskers/active/utils"
	"github.com/google/go-github/v31/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Projects []string `yaml:"projects"`
	Token    string   `yaml:"token"`
}

// Doesn't mind if the expected fields are missing from the config file.
// Default values are supplied if they are missing.
// Exits the program if there were any errors.
func ReadConfig() *Config {
	c := Config{}
	file, e0 := ioutil.ReadFile("/home/colin/code/go/active/active.yaml")
	utils.ExitIfErr(e0)
	e1 := yaml.Unmarshal(file, &c)
	utils.ExitIfErr(e1)
	return &c
}

func GithubClient(config *Config, token *string) *github.Client {
	if *token == "" && config.Token == "" {
		return github.NewClient(nil)
	} else {
		tok := *token
		if tok == "" {
			tok = config.Token
		}
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tok})
		tc := oauth2.NewClient(ctx, ts)
		return github.NewClient(tc)
	}
}
