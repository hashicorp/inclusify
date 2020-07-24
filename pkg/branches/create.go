package branch

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/hashicorp/inclusify/pkg/gh"
)

// CreateCommand is a struct used to configure a Command for creating new
// GitHub branches in the remote repo
type CreateCommand struct {
	Config *gh.GitHub
}

// Create a branch called $target from the head commit of $base
// The $base branch must already exist
// Example: Create a branch 'main' off of 'master'
func createBranch(config *gh.GitHub, base string, target string) error {
	// Get base Ref
	refName := fmt.Sprintf("refs/heads/%s", base)
	ctx, _ := context.WithTimeout(config.Ctx, 10*time.Second)
	ref, _, err := config.Client.Git.GetRef(ctx, config.Owner, config.Repo, refName)
	if err != nil {
		return fmt.Errorf("call to get master ref returned error: %w", err)
	}

	// Get base SHA
	sha := ref.Object.GetSHA()

	// Setup to create a new ref called $target off of $base
	targetRef := fmt.Sprintf("refs/heads/%s", target)
	targetRefObj := &github.Reference{
		Ref: &targetRef,
		Object: &github.GitObject{
			SHA: &sha,
		},
	}

	// Create $target ref
	_, _, err = config.Client.Git.CreateRef(config.Ctx, config.Owner, config.Repo, targetRefObj)
	if err != nil {
		return fmt.Errorf("call to create base ref returned error: %w", err)
	}

	return nil
}

// Run creates the branch $target off of $base
// It also creates a $tmpBranch that will be used for CI changes
// Example: Create branches 'main' and 'update-ci-references' off of master
func (c *CreateCommand) Run(args []string) int {
	// Create branch $target off of head commit in $base
	c.Config.Logger.Info("Creating new branch $target off of $base", "target", c.Config.Target, "base", c.Config.Base)
	err := createBranch(c.Config, c.Config.Base, c.Config.Target)
	if err != nil {
		return c.exitError(err)
	}

	// Create $tmpBranch off of head commit in $target
	// CI changes will be pushed to the $tmpBranch and a PR will be opened
	// to merge those changes into $target
	tmpBranch := "update-ci-references"
	c.Config.Logger.Info("Creating new temp branch $target off of $base", "target", tmpBranch, "base", c.Config.Target)
	err = createBranch(c.Config, c.Config.Base, tmpBranch)
	if err != nil {
		return c.exitError(err)
	}

	c.Config.Logger.Info("Success!")

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *CreateCommand) exitError(err error) int {
	c.Config.Logger.Error(err.Error())
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
