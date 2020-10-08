package config

import (
	"fmt"
	"strings"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/inclusify/pkg/message"
	"github.com/mitchellh/cli"
	nflag "github.com/namsral/flag"
)

// Config is a struct that contains user inputs and our logger
type Config struct {
	Owner     string
	Repo      string
	Base      string
	Target    string
	Token     string
	Exclusion []string
	Logger    hclog.Logger
}

// ParseAndValidate parses the cmd line flags / env vars, and verifies that all required
// flags have been set. Users can pass in flags when calling a subcommand, or set env vars
// with the prefix 'INCLUSIFY_'. If both values are set, the env var value will be used.
func ParseAndValidate(args []string, ui cli.Ui) (c *Config, err error) {
	var (
		owner, repo, token, base, target, exclusion string
	)
	var exclusionArr []string

	// Values can be passed in to the subcommands as inputs flags,
	// or set as env vars with the prefix "INCLUSIFY_"
	flags := nflag.NewFlagSetWithEnvPrefix("inclusify", "INCLUSIFY", 0)
	flags.StringVar(&owner, "owner", "", "The GitHub org that owns the repo, e.g. 'hashicorp'")
	flags.StringVar(&repo, "repo", "", "The repository name, e.g. 'circle-codesign'")
	flags.StringVar(&base, "base", "master", "The name of the current base branch, e.g. 'master'")
	flags.StringVar(&target, "target", "main", "The name of the target branch, e.g. 'main'")
	flags.StringVar(&token, "token", "", "Your Personal GitHub Access Token")
	flags.StringVar(&exclusion, "exclusion", "", "Paths to exclude from reference updates, e.g. '.circleci/config.yml,.teamcity.yml'")

	// Special check for ./inclusify invocation without any args
	// Return the help message
	if len(args) == 0 {
		args = append(args, "--help")
	}

	// Pop the subcommand into 'cmd'
	// flags.Parse does not work when the subcommand is included
	cmd, inputFlags := args[0], args[1:]

	// Special check for help commands
	// command is ./inclusify --help or --version
	if len(inputFlags) == 0 && (cmd == "help" || cmd == "--help" || cmd == "-help" || cmd == "version" || cmd == "--version" || cmd == "-version") {
		return nil, nil
	}
	// command is ./inclusify $subcommand --help
	if len(inputFlags) == 1 && (inputFlags[0] == "help" || inputFlags[0] == "--help" || inputFlags[0] == "-help") {
		return nil, nil
	}

	if err := flags.Parse(inputFlags); err != nil {
		return c, fmt.Errorf("error parsing inputs: %w", err)
	}

	if owner == "" || repo == "" || token == "" {
		return c, fmt.Errorf(
			"%s\nPass in all required flags or set environment variables with the 'INCLUSIFY_' prefix.\nRun [subcommand] --help to view required inputs.",
			message.Error("required inputs are missing"),
		)
	}

	if len(exclusion) > 0 {
		exclusionArr = strings.Split(exclusion, ",")
	}
	exclusionArr = append(exclusionArr, ".git/", "go.mod", "go.sum")

	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "inclusify",
		Level:  hclog.LevelFromString("INFO"),
		Output: &cli.UiWriter{Ui: ui},
	})

	c = &Config{
		Owner:     owner,
		Repo:      repo,
		Base:      base,
		Target:    target,
		Token:     token,
		Exclusion: exclusionArr,
		Logger:    logger,
	}

	return c, nil
}
