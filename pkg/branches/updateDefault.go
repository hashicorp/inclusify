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
func SetupBranchProtectionReq(c *UpdateCommand, baseProtection *github.Protection) (req *github.ProtectionRequest, err error) {
	var (
		drUsers, drTeams, brUsers, brTeams, brApps []string
	)

	targetProtectionReq := &github.ProtectionRequest{}
	brRequest := &github.BranchRestrictionsRequest{}
	prRequest := &github.PullRequestReviewsEnforcementRequest{}
	drRequest := &github.DismissalRestrictionsRequest{}

	if baseProtection.GetRequiredPullRequestReviews() != nil {
		if baseProtection.GetRequiredPullRequestReviews().GetDismissalRestrictions() != nil {
			for _, u := range baseProtection.GetRequiredPullRequestReviews().GetDismissalRestrictions().Users {
				drUsers = append(drUsers, u.GetLogin())
			}
			for _, t := range baseProtection.GetRequiredPullRequestReviews().GetDismissalRestrictions().Teams {
				drTeams = append(drTeams, t.GetSlug())
			}
			users := &drUsers
			if len(*users) == 0 {
				users = &[]string{}
			}
			teams := &drTeams
			if len(*teams) == 0 {
				teams = &[]string{}
			}
			drRequest = &github.DismissalRestrictionsRequest{
				Users: users,
				Teams: teams,
			}
			prReviews := 1
			if baseProtection.GetRequiredPullRequestReviews().RequiredApprovingReviewCount != 0 {
				prReviews = baseProtection.GetRequiredPullRequestReviews().RequiredApprovingReviewCount
			}
			prRequest = &github.PullRequestReviewsEnforcementRequest{
				DismissalRestrictionsRequest: drRequest,
				DismissStaleReviews:          baseProtection.GetRequiredPullRequestReviews().DismissStaleReviews,
				RequireCodeOwnerReviews:      baseProtection.GetRequiredPullRequestReviews().RequireCodeOwnerReviews,
				RequiredApprovingReviewCount: prReviews,
			}
		} else {
			prReviews := 1
			if baseProtection.GetRequiredPullRequestReviews().RequiredApprovingReviewCount != 0 {
				prReviews = baseProtection.GetRequiredPullRequestReviews().RequiredApprovingReviewCount
			}
			prRequest = &github.PullRequestReviewsEnforcementRequest{
				DismissalRestrictionsRequest: nil,
				DismissStaleReviews:          baseProtection.GetRequiredPullRequestReviews().DismissStaleReviews,
				RequireCodeOwnerReviews:      baseProtection.GetRequiredPullRequestReviews().RequireCodeOwnerReviews,
				RequiredApprovingReviewCount: prReviews,
			}
		}
	}

	if baseProtection.GetRestrictions() != nil {
		for _, u := range baseProtection.GetRestrictions().Users {
			brUsers = append(brUsers, u.GetLogin())
		}
		for _, t := range baseProtection.GetRestrictions().Teams {
			brTeams = append(brTeams, t.GetSlug())
		}
		for _, a := range baseProtection.GetRestrictions().Apps {
			brApps = append(brApps, a.GetSlug())
		}
		users := &brUsers
		if len(*users) == 0 {
			users = &[]string{}
		}
		teams := &brTeams
		if len(*teams) == 0 {
			teams = &[]string{}
		}
		apps := &brApps
		if len(*apps) == 0 {
			apps = &[]string{}
		}
		brRequest = &github.BranchRestrictionsRequest{
			Users: *users,
			Teams: *teams,
			Apps:  *apps,
		}
	}

	targetProtectionReq = &github.ProtectionRequest{
		RequiredStatusChecks:       baseProtection.GetRequiredStatusChecks(),
		RequiredPullRequestReviews: prRequest,
		EnforceAdmins:              baseProtection.GetEnforceAdmins().Enabled,
		Restrictions:               brRequest,
		RequireLinearHistory:       &baseProtection.GetRequireLinearHistory().Enabled,
		AllowForcePushes:           &baseProtection.GetAllowForcePushes().Enabled,
		AllowDeletions:             &baseProtection.GetAllowDeletions().Enabled,
	}

	return targetProtectionReq, nil
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
