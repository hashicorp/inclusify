package branch

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/gh"
	"github.com/mitchellh/cli"

	github "github.com/google/go-github/v32/github"
)

// UpdateCommand is a struct used to configure a Command for updating
// the GitHub default branch in the remote repo and copying the
// branch protection from $base to $target
type UpdateCommand struct {
	UI           cli.Ui
	Config       *config.Config
	GithubClient gh.GithubInteractor
}

// SetupBranchProtectionReq sets up the branch protection request
func SetupBranchProtectionReq(c *UpdateCommand, base *github.Protection) *github.ProtectionRequest {
	enforceUsers := userStrings(base.RequiredPullRequestReviews.DismissalRestrictions.Users)
	enforceTeams := teamStrings(base.RequiredPullRequestReviews.DismissalRestrictions.Teams)
	enforceReq := &github.PullRequestReviewsEnforcementRequest{
		DismissalRestrictionsRequest: &github.DismissalRestrictionsRequest{
			Users: &enforceUsers,
			Teams: &enforceTeams,
		},
		DismissStaleReviews:          base.RequiredPullRequestReviews.DismissStaleReviews,
		RequireCodeOwnerReviews:      base.RequiredPullRequestReviews.RequireCodeOwnerReviews,
		RequiredApprovingReviewCount: base.RequiredPullRequestReviews.RequiredApprovingReviewCount,
	}

	branchReq := &github.BranchRestrictionsRequest{
		Users: userStrings(base.Restrictions.Users),
		Teams: teamStrings(base.Restrictions.Teams),
		Apps:  appStrings(base.Restrictions.Apps),
	}

	req := &github.ProtectionRequest{
		RequiredStatusChecks:       base.RequiredStatusChecks,
		RequiredPullRequestReviews: enforceReq,
		EnforceAdmins:              base.EnforceAdmins.Enabled,
		Restrictions:               branchReq,
		RequireLinearHistory:       &base.RequireLinearHistory.Enabled,
		AllowForcePushes:           &base.AllowForcePushes.Enabled,
		AllowDeletions:             &base.AllowDeletions.Enabled,
	}
	return req
}

func userStrings(users []*github.User) []string {
	out := make([]string, len(users))
	for i, u := range users {
		out[i] = u.GetName()
	}
	return out
}

func teamStrings(teams []*github.Team) []string {
	out := make([]string, len(teams))
	for i, t := range teams {
		out[i] = t.GetName()
	}
	return out
}

func appStrings(apps []*github.App) []string {
	out := make([]string, len(apps))
	for i, a := range apps {
		out[i] = a.GetName()
	}
	return out
}

// CopyBranchProtection will copy the branch protection from base and apply it to $target
func CopyBranchProtection(c *UpdateCommand, base string, target string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c.Config.Logger.Info("Getting branch protection for branch", "branch", c.Config.Base)
	baseProtection, res, err := c.GithubClient.GetRepo().GetBranchProtection(ctx, c.Config.Owner, c.Config.Repo, base)
	if err != nil {
		if res.StatusCode == 404 {
			c.Config.Logger.Info("Exiting -- The old base branch isn't protected, so there's nothing more to do")
			return nil
		}
		return fmt.Errorf("failed to get $base branch protection: %w", err)
	}

	c.Config.Logger.Info("Creating the branch protection request for branch", "branch", c.Config.Target)
	targetProtectionReq, err := SetupBranchProtectionReq(c, baseProtection)
	if err != nil {
		return fmt.Errorf("failed to create the branch protection request: %w", err)
	}

	c.Config.Logger.Info("Updating the branch protection on branch", "branch", c.Config.Target)
	_, _, err = c.GithubClient.GetRepo().UpdateBranchProtection(ctx, c.Config.Owner, c.Config.Repo, target, targetProtectionReq)
	if err != nil {
		return fmt.Errorf("failed to update the $target branches protection: %w", err)
	}

	return nil
}

// Run updates the default branch in the repo to the new $target branch
// and copies the branch protection rules from $base to $target
// Example: Update the repo's default branch from 'master' to 'main'
func (c *UpdateCommand) Run(args []string) int {
	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()

	// c.Config.Logger.Info("Updating the default branch in $repo from $base to $target", "repo", c.Config.Repo, "base", c.Config.Base, "target", c.Config.Target)
	// editRepo := &github.Repository{DefaultBranch: &c.Config.Target}
	// _, _, err := c.GithubClient.GetRepo().Edit(ctx, c.Config.Owner, c.Config.Repo, editRepo)
	// if err != nil {
	// 	return c.exitError(fmt.Errorf("failed to update default branch: %w", err))
	// }

	c.Config.Logger.Info("Attempting to apply the $base branch protection to $target", "base", c.Config.Base, "target", c.Config.Target)
	err := CopyBranchProtection(c, c.Config.Base, c.Config.Target)
	if err != nil {
		return c.exitError(err)
	}

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *UpdateCommand) exitError(err error) int {
	c.Config.Logger.Error(err.Error())
	return 1
}

// Help returns the full help text.
func (c *UpdateCommand) Help() string {
	return `Usage: inclusify updateDefault owner repo target token
Update the default branch in the repo to $target, and copy branch protection from $base to $target. Configuration is pulled from the local environment.
Flags:
--owner          The GitHub org that owns the repo, e.g. 'hashicorp'.
--repo           The repository name, e.g. 'circle-codesign'.
--target="main"  The name of the target branch, e.g. 'main'.
--token          Your Personal GitHub Access Token.
`
}

// Synopsis returns a sub 50 character summary of the command.
func (c *UpdateCommand) Synopsis() string {
	return "Update repo's default branch. [subcommand]"
}
