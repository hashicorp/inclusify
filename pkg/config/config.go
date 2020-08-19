package config

import (
	"fmt"

	"github.com/mitchellh/cli"

	hclog "github.com/hashicorp/go-hclog"
	nflag "github.com/namsral/flag"
)

// Config is a struct that contains user inputs and our logger
type Config struct {
	Owner  string
	Repo   string
	Base   string
	Target string
	Token  string
	Logger hclog.Logger
}

// ParseAndValidate parses the cmd line flags / env vars, and verifies that all required
// flags have been set. Users can pass in flags when calling a subcommand, or set env vars
// with the prefix 'INCLUSIFY_'. If both values are set, the env var value will be used.
func ParseAndValidate(args []string, ui cli.Ui) (c *Config, err error) {
	var (
		owner, repo, token string
	)
	var base = "master"
	var target = "main"

	// Values can be passed in to the subcommands as inputs flags,
	// or set as env vars with the prefix "INCLUSIFY_"
	flags := nflag.NewFlagSetWithEnvPrefix("inclusify", "INCLUSIFY", 0)
	flags.StringVar(&owner, "owner", "", "The GitHub org that owns the repo, e.g. 'hashicorp'")
	flags.StringVar(&repo, "repo", "", "The repository name, e.g. 'circle-codesign'")
	flags.StringVar(&base, "base", base, "The name of the current base branch, e.g. 'master'")
	flags.StringVar(&target, "target", target, "The name of the target branch, e.g. 'main'")
	flags.StringVar(&token, "token", "", "Your Personal GitHub Access Token")

	// Pop the subcommand into 'cmd'
	// flags.Parse does not work when the subcommand is included
	cmd, inputFlags := args[0], args[1:]

	if cmd != "inclusify" {
		// Special check for help commands
		if len(inputFlags) == 1 {
			if inputFlags[0] == "--help" || inputFlags[0] == "-help" {
				return &Config{
					Owner:  "irrelevant",
					Repo:   "irrelevant",
					Base:   "irrelevant",
					Target: "irrelevant",
					Token:  "irrelevant",
					Logger: hclog.New(&hclog.LoggerOptions{}),
				}, nil
			}
		}

		if err := flags.Parse(inputFlags); err != nil {
			return c, fmt.Errorf("error parsing inputs: %w", err)
		}

		if owner == "" || repo == "" || token == "" {
			return c, fmt.Errorf("required inputs are missing\nPass in all required flags or set env vars with the prefix INCLUSIFY\nRun [subcommand] --help to view required inputs")
		}

		logger := hclog.New(&hclog.LoggerOptions{
			Name:   "inclusify",
			Level:  hclog.LevelFromString("INFO"),
			Output: &cli.UiWriter{Ui: ui},
		})

		c = &Config{
			Owner:  owner,
			Repo:   repo,
			Base:   base,
			Target: target,
			Token:  token,
			Logger: logger,
		}

		return c, nil
	}

	return nil, nil

}
