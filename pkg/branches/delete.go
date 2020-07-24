package branch

import (
	"fmt"

	"github.com/hashicorp/inclusify/pkg/gh"
)

// DeleteCommand is a struct used to configure a Command for deleting the
// GitHub branch, $base, in the remote repo
type DeleteCommand struct {
	Config *gh.GitHub
}

// Run removes the branch protection from the $base branch
// and deletes the $base branch from the remote repo
// $base defaults to "master" if no $base flag or env var is provided
// Example: Delete the 'master' branch
func (c *DeleteCommand) Run(args []string) int {
	// Remove the branch protection from the old base
	c.Config.Logger.Info("Removing branch protection from the old default branch, $base", "base", c.Config.Base)
	_, err := c.Config.Client.Repositories.RemoveBranchProtection(c.Config.Ctx, c.Config.Owner, c.Config.Repo, c.Config.Base)
	if err != nil {
		return c.exitError(err)
	}

	// Delete the old base branch from GitHub
	c.Config.Logger.Info("Attempting to delete branch", "branch", c.Config.Base)
	refName := fmt.Sprintf("refs/heads/%s", c.Config.Base)
	_, err = c.Config.Client.Git.DeleteRef(c.Config.Ctx, c.Config.Owner, c.Config.Repo, refName)
	if err != nil {
		return c.exitError(fmt.Errorf("failed to delete ref: %w", err))
	}

	c.Config.Logger.Info("Success! $branch has been deleted", "branch", c.Config.Base, "ref", refName)

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *DeleteCommand) exitError(err error) int {
	c.Config.Logger.Error(err.Error())
	return 1
}

// Help returns the full help text.
func (c *DeleteCommand) Help() string {
	return `Usage: inclusify deleteBranch owner repo base token
Delete $base branch from the given GitHub repo. Configuration is pulled from the local environment.
Flags:
--owner          The GitHub org that owns the repo, e.g. 'hashicorp'.
--repo           The repository name, e.g. 'circle-codesign'.
--base="master"  The name of the current base branch, e.g. 'master'.
--token          Your Personal GitHub Access Token.
`
}

// Synopsis returns a sub 50 character summary of the command.
func (c *DeleteCommand) Synopsis() string {
	return "Delete repo's base branch. [subcommand]"
}
