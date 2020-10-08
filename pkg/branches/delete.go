package branches

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/gh"
	"github.com/hashicorp/inclusify/pkg/message"
)

// DeleteCommand is a struct used to configure a Command for deleting the
// GitHub branch, $base, in the remote repo
type DeleteCommand struct {
	Config       *config.Config
	GithubClient gh.GithubInteractor
	BranchesList []string
}

// Run removes the branch protection from the $base branch
// and deletes the $base branch from the remote repo
// $base defaults to "master" if no $base flag or env var is provided
// Example: Delete the 'master' branch
func (c *DeleteCommand) Run(args []string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c.BranchesList = append(c.BranchesList, c.Config.Base)
	for _, branch := range c.BranchesList {
		c.Config.Logger.Info("Attempting to remove branch protection from branch", "branch", branch)
		_, err := c.GithubClient.GetRepo().RemoveBranchProtection(ctx, c.Config.Owner, c.Config.Repo, branch)
		if err != nil {
			// If there's no branch protection for the branch, that's OK! Log it and continue on
			c.Config.Logger.Info("Failed to remove branch protection from branch", "branch", branch, "error", err)
		}

		c.Config.Logger.Info("Attempting to delete branch", "branch", branch)
		refName := fmt.Sprintf("refs/heads/%s", branch)
		_, err = c.GithubClient.GetGit().DeleteRef(ctx, c.Config.Owner, c.Config.Repo, refName)
		if err != nil {
			// If there's no branch to delete, that's OK! Log it and continue on
			c.Config.Logger.Info("Failed to delete ref", "branch", branch, "error", err)
		}

		c.Config.Logger.Info(message.Success("Success! branch has been deleted"), "branch", branch, "ref", refName)
	}

	return 0
}

// Help returns the full help text.
func (c *DeleteCommand) Help() string {
	return `Usage: inclusify deleteBranches owner repo base token
	Delete $base branch and other auto-created branches from the given GitHub repo. Configuration is pulled from the local environment.
	Flags:
	--owner          The GitHub org that owns the repo, e.g. 'hashicorp'.
	--repo           The repository name, e.g. 'circle-codesign'.
	--base="master"  The name of the current base branch, e.g. 'master'.
	--token          Your Personal GitHub Access Token.
	`
}

// Synopsis returns a sub 50 character summary of the command.
func (c *DeleteCommand) Synopsis() string {
	return "Delete repo's base branch and other auto-created branches. [subcommand]"
}
