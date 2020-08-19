package branch

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/mitchellh/cli"

	"github.com/hashicorp/inclusify/pkg/gh"
)

// CreateCommand is a struct used to configure a Command for creating new
// GitHub branches in the remote repo
type CreateCommand struct {
	UI           cli.Ui
	Config       *config.Config
	GithubClient gh.GithubInteractor
}

// Create a branch called $target from the head commit of $base
// The $base branch must already exist
// Example: Create a branch 'main' off of 'master'
func CreateBranch(c *CreateCommand, branch string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	refName := fmt.Sprintf("refs/heads/%s", c.Config.Base)
	ref, _, err := c.GithubClient.GetGit().GetRef(ctx, c.Config.Owner, c.Config.Repo, refName)
	if err != nil {
		return fmt.Errorf("call to get master ref returned error: %w", err)
	}
	sha := ref.Object.GetSHA()

	targetRef := fmt.Sprintf("refs/heads/%s", branch)
	targetRefObj := &github.Reference{
		Ref: &targetRef,
		Object: &github.GitObject{
			SHA: &sha,
		},
	}

	_, _, err = c.GithubClient.GetGit().CreateRef(ctx, c.Config.Owner, c.Config.Repo, targetRefObj)
	if err != nil {
		return fmt.Errorf("call to create base ref returned error: %w", err)
	}

	return nil
}

// Run creates the branch $target off of $base
// It also creates a $tmpBranch that will be used for CI changes
// Example: Create branches 'main' and 'update-ci-references' off of master
func (c *CreateCommand) Run(args []string) int {
	c.UI.Info(fmt.Sprintf(
		"Creating new branch %s off of %s", c.Config.Target, c.Config.Base,
	))
	err := CreateBranch(c, c.Config.Target)
	if err != nil {
		return c.exitError(err)
	}

	tmpBranch := "update-ci-references"
	c.UI.Info(fmt.Sprintf(
		"Creating new temp branch %s off of %s", tmpBranch, c.Config.Base,
	))
	err = CreateBranch(c, tmpBranch)
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
