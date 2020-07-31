package branch

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/hashicorp/inclusify/pkg/gh"
	"github.com/mitchellh/cli"
)

// CreateCommand is a struct used to configure a Command for creating new
// GitHub branches in the remote repo
type CreateCommand struct {
	UI           cli.Ui
	GithubClient gh.GithubInteractor

	Owner, Repo  string
	base, target string
}

// Create a branch called $target from the head commit of $base
// The $base branch must already exist
// Example: Create a branch 'main' off of 'master'
func (c *CreateCommand) createBranch(branch string) error {
	// Get base Ref
	refName := fmt.Sprintf("refs/heads/%s", c.base)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ref, _, err := c.GithubClient.GetGit().GetRef(ctx, c.Owner, c.Repo, refName)
	if err != nil {
		return fmt.Errorf("call to get master ref returned error: %w", err)
	}

	// Get base SHA
	sha := ref.Object.GetSHA()

	// Setup to create a new ref called $target off of $base
	targetRef := fmt.Sprintf("refs/heads/%s", branch)
	targetRefObj := &github.Reference{
		Ref: &targetRef,
		Object: &github.GitObject{
			SHA: &sha,
		},
	}

	// Create $target ref
	_, _, err = c.GithubClient.GetGit().CreateRef(ctx, c.Owner, c.Repo, targetRefObj)
	if err != nil {
		return fmt.Errorf("call to create base ref returned error: %w", err)
	}

	return nil
}

// Run creates the branch $target off of $base
// It also creates a $tmpBranch that will be used for CI changes
// Example: Create branches 'main' and 'update-ci-references' off of master
func (c *CreateCommand) Run(args []string) int {
	// flag parsin'
	fs := flag.NewFlagSet("create", flag.ExitOnError)

	fs.StringVar(&c.target, "target", c.target, "")
	fs.StringVar(&c.base, "base", c.base, "")

	if err := fs.Parse(args); err != nil {
		return c.exitError(fmt.Errorf("error parsing command line flags: %w", err))
	}

	// Create branch $target off of head commit in $base
	c.UI.Info(fmt.Sprintf(
		"Creating new branch %s off of %s", c.target, c.base,
	))
	err := c.createBranch(c.target)
	if err != nil {
		return c.exitError(err)
	}

	// Create $tmpBranch off of head commit in $target
	// CI changes will be pushed to the $tmpBranch and a PR will be opened
	// to merge those changes into $target
	tmpBranch := "update-ci-references"
	c.UI.Info(fmt.Sprintf(
		"Creating new temp branch %s off of %s", tmpBranch, c.base,
	))
	err = c.createBranch(tmpBranch)
	if err != nil {
		return c.exitError(err)
	}

	c.UI.Info("Success!")

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *CreateCommand) exitError(err error) int {
	c.UI.Error(err.Error())
	return 1
}

// Help returns the full help text.
func (c *CreateCommand) Help() string {
	return `Usage: inclusify createBranches owner repo base target token
Create a new branch called $target off $base, with all history included. Configuration is pulled from the local environment.
Flags:
--owner          The GitHub org that owns the repo, e.g. 'hashicorp'.
--repo           The repository name, e.g. 'circle-codesign'.
--base="master"  The name of the current base branch, e.g. 'master'.
--target="main"  The name of the target branch, e.g. 'main'.
--token          Your Personal GitHub Access Token.
`
}

// Synopsis returns a sub 50 character summary of the command.
func (c *CreateCommand) Synopsis() string {
	return "Create new branches on GitHub. [subcommand]"
}
