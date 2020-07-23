package branch

import (
	"fmt"

	"github.com/hashicorp/inclusify/pkg/gh"

	github "github.com/google/go-github/v32/github"
)

// UpdateCommand is a struct used to configure a Command for updating
// the GitHub default branch in the remote repo and copying the
// branch protection from $base to $target
type UpdateCommand struct {
	Config *gh.GitHub
}

// SetupBranchProtectionReq sets up the branch protection request
func SetupBranchProtectionReq(config *gh.GitHub, baseProtection *github.Protection) (req *github.ProtectionRequest, err error) {
	// Declare vars
	drUsers := []string{}
	drTeams := []string{}
	brUsers := []string{}
	brTeams := []string{}
	brApps := []string{}

	targetProtectionReq := &github.ProtectionRequest{}
	brRequest := &github.BranchRestrictionsRequest{}
	prRequest := &github.PullRequestReviewsEnforcementRequest{}
	drRequest := &github.DismissalRestrictionsRequest{}

	if baseProtection.GetRequiredPullRequestReviews() != nil {
		// Set up dismissal restrictions request
		if baseProtection.GetRequiredPullRequestReviews().GetDismissalRestrictions() != nil {
			for _, u := range baseProtection.GetRequiredPullRequestReviews().GetDismissalRestrictions().Users {
				drUsers = append(drUsers, u.GetLogin())
			}
			for _, t := range baseProtection.GetRequiredPullRequestReviews().GetDismissalRestrictions().Teams {
				drTeams = append(drTeams, t.GetSlug())
			}
			drRequest = &github.DismissalRestrictionsRequest{
				Users: &drUsers,
				Teams: &drTeams,
			}
			// Set up PR request
			prRequest = &github.PullRequestReviewsEnforcementRequest{
				DismissalRestrictionsRequest: drRequest,
				DismissStaleReviews:          baseProtection.GetRequiredPullRequestReviews().DismissStaleReviews,
				RequireCodeOwnerReviews:      baseProtection.GetRequiredPullRequestReviews().RequireCodeOwnerReviews,
				RequiredApprovingReviewCount: baseProtection.GetRequiredPullRequestReviews().RequiredApprovingReviewCount,
			}
		} else {
			// Set up PR request
			prRequest = &github.PullRequestReviewsEnforcementRequest{
				DismissStaleReviews:          baseProtection.GetRequiredPullRequestReviews().DismissStaleReviews,
				RequireCodeOwnerReviews:      baseProtection.GetRequiredPullRequestReviews().RequireCodeOwnerReviews,
				RequiredApprovingReviewCount: baseProtection.GetRequiredPullRequestReviews().RequiredApprovingReviewCount,
			}
		}
	}

	// Set up branch restrictions request
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
		brRequest = &github.BranchRestrictionsRequest{
			Users: brUsers,
			Teams: brTeams,
			Apps:  brApps,
		}
	}

	// Set up main request
	if baseProtection.GetRestrictions() != nil {
		if baseProtection.GetRequiredPullRequestReviews() != nil {
			// PR and BR good
			targetProtectionReq = &github.ProtectionRequest{
				RequiredStatusChecks:       baseProtection.GetRequiredStatusChecks(),
				RequiredPullRequestReviews: prRequest,
				EnforceAdmins:              baseProtection.GetEnforceAdmins().Enabled,
				Restrictions:               brRequest,
				RequireLinearHistory:       &baseProtection.GetRequireLinearHistory().Enabled,
				AllowForcePushes:           &baseProtection.GetAllowForcePushes().Enabled,
				AllowDeletions:             &baseProtection.GetAllowDeletions().Enabled,
			}
		} else {
			// PR nil, BR good
			targetProtectionReq = &github.ProtectionRequest{
				RequiredStatusChecks: baseProtection.GetRequiredStatusChecks(),
				EnforceAdmins:        baseProtection.GetEnforceAdmins().Enabled,
				Restrictions:         brRequest,
				RequireLinearHistory: &baseProtection.GetRequireLinearHistory().Enabled,
				AllowForcePushes:     &baseProtection.GetAllowForcePushes().Enabled,
				AllowDeletions:       &baseProtection.GetAllowDeletions().Enabled,
			}
		}
	} else {
		if baseProtection.GetRequiredPullRequestReviews() != nil {
			// PR good, BR nil
			targetProtectionReq = &github.ProtectionRequest{
				RequiredStatusChecks:       baseProtection.GetRequiredStatusChecks(),
				RequiredPullRequestReviews: prRequest,
				EnforceAdmins:              baseProtection.GetEnforceAdmins().Enabled,
				RequireLinearHistory:       &baseProtection.GetRequireLinearHistory().Enabled,
				AllowForcePushes:           &baseProtection.GetAllowForcePushes().Enabled,
				AllowDeletions:             &baseProtection.GetAllowDeletions().Enabled,
			}
		} else {
			// PR and BR are nil
			targetProtectionReq = &github.ProtectionRequest{
				RequiredStatusChecks: baseProtection.GetRequiredStatusChecks(),
				EnforceAdmins:        baseProtection.GetEnforceAdmins().Enabled,
				RequireLinearHistory: &baseProtection.GetRequireLinearHistory().Enabled,
				AllowForcePushes:     &baseProtection.GetAllowForcePushes().Enabled,
				AllowDeletions:       &baseProtection.GetAllowDeletions().Enabled,
			}
		}
	}

	return targetProtectionReq, nil
}

// CopyBranchProtection will copy the branch protection from base and apply it to $target
func CopyBranchProtection(config *gh.GitHub, base string, target string) (err error) {
	// Get branch protection from base
	config.Logger.Info("Getting branch protection for the $base branch", "base", config.Base)
	baseProtection, res, err := config.Client.Repositories.GetBranchProtection(config.Ctx, config.Owner, config.Repo, base)
	if err != nil {
		if res.StatusCode == 404 {
			return fmt.Errorf("Exiting -- The old base branch isn't protected, so there's nothing more to do")
		}
		return fmt.Errorf("failed to get $base branch protection: %w", err)
	}

	// Create the github branch protection request
	config.Logger.Info("Creating the branch protection request for $target", "target", config.Target)
	targetProtectionReq, err := SetupBranchProtectionReq(config, baseProtection)
	if err != nil {
		return fmt.Errorf("failed to create the branch protection request: %w", err)
	}

	// Update the branch protection on the new default branch, $target
	config.Logger.Info("Updating the branch protection on the $target branch", "target", config.Target)
	_, _, err = config.Client.Repositories.UpdateBranchProtection(config.Ctx, config.Owner, config.Repo, target, targetProtectionReq)
	if err != nil {
		return fmt.Errorf("failed to update the $target branches protection: %w", err)
	}

	return nil
}

// Run updates the default branch in the repo to the new $target branch
// and copies the branch protection rules from $base to $target
// Example: Update the repo's default branch from 'master' to 'main'
func (c *UpdateCommand) Run(args []string) int {
	// Update the default branch to $target on the remote GitHub repo
	c.Config.Logger.Info("Updating the default branch in $repo from $base to $target", "repo", c.Config.Repo, "base", c.Config.Base, "target", c.Config.Target)
	editRepo := &github.Repository{DefaultBranch: &c.Config.Target}
	_, _, err := c.Config.Client.Repositories.Edit(c.Config.Ctx, c.Config.Owner, c.Config.Repo, editRepo)
	if err != nil {
		return c.exitError(fmt.Errorf("failed to update default branch: %w", err))
	}

	// Copy over the branch protection from the old base to the new default branch, $target
	c.Config.Logger.Info("Attempting to apply the $base branch protection to $target", "base", c.Config.Base, "target", c.Config.Target)
	err = CopyBranchProtection(c.Config, c.Config.Base, c.Config.Target)
	if err != nil {
		return c.exitError(err)
	}

	c.Config.Logger.Info("Success!")

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
