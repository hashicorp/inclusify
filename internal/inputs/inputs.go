package inputs

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"golang.org/x/oauth2"

	github "github.com/google/go-github/v32/github"
	gh "github.com/hashicorp/inclusify/pkg/gh"
	nflag "github.com/namsral/flag"
)

// ParseAndValidate parses the cmd line flags / env vars, and verifies that all required
// flags have been set. Users can pass in flags when calling a subcommand, or set env vars
// with the prefix 'INCLUSIFY_'. If both values are set, the env var value will be used.
func ParseAndValidate(args []string) (config *gh.GitHub, err error) {
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

	// Parse args and check for errors
	if err := flags.Parse(args); err != nil {
		return config, fmt.Errorf("error parsing inputs: %w", err)
	}
	if owner == "" || repo == "" || token == "" {
		return config, fmt.Errorf("required inputs are missing\nPass in all required flags or set env vars with the prefix INCLUSIFY\nRun [subcommand] --help to view required inputs")
	}

	// Setup GitHub Client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Setup a global logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "inclusify",
		Level: hclog.LevelFromString("INFO"),
	})

	// Store all inputs in a struct
	config = &gh.GitHub{
		Owner:  owner,
		Repo:   repo,
		Base:   base,
		Target: target,
		Token:  token,
		Client: github.NewClient(tc),
		Ctx:    ctx,
		Logger: logger,
	}

	return config, nil

}
