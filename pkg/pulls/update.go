package branch

import (
	"fmt"

	"github.com/google/go-github/v32/github"
	"github.com/hashicorp/inclusify/pkg/gh"
)

// UpdateCommand is a struct used to configure a Command for updating open
// PR's that target master to target the new $base
type UpdateCommand struct {
	Config *gh.GitHub
}

// GetOpenPRs returns an array of all open PR's that target the $base branch
func GetOpenPRs(config *gh.GitHub) (pulls []*github.PullRequest, err error) {
	config.Logger.Info("Getting all open PR's targetting the $base branch", "base", config.Base)

	// Setup request to list all open PR's targetting the $base branch
	var allPulls []*github.PullRequest
	opts := &github.PullRequestListOptions{
		State:       "open",
		Base:        config.Base,
		ListOptions: github.ListOptions{PerPage: 10},
	}

	// Paginate to get all open PR's and store them in 'allPulls' array
	for {
		pulls, resp, err := config.Client.PullRequests.List(config.Ctx, config.Owner, config.Repo, opts)
		if err != nil {
			return nil, fmt.Errorf("Failed to retrieve all open PR's: %w", err)
		}
		allPulls = append(allPulls, pulls...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	config.Logger.Info("Retrieved all open PR's targetting the $base branch", "base", config.Base, "prCount", len(allPulls))

	return allPulls, nil
}

// UpdateOpenPRs will update all open PR's that pointed to $base to instead point to $target
// Exmaple: Update all open PR's that point to 'master' to point to 'main'
func UpdateOpenPRs(config *gh.GitHub, pulls []*github.PullRequest, targetRef *github.Reference) (err error) {
	for _, pull := range pulls {
		pull.Base.Label = &config.Target
		pull.Base.Ref = targetRef.Ref
		// Attempt to update the PR
		updatedPull, _, err := config.Client.PullRequests.Edit(config.Ctx, config.Owner, config.Repo, *pull.Number, pull)
		if err != nil {
			errString := fmt.Sprintf("failed to update base branch of PR %s", *pull.URL)
			return fmt.Errorf(errString+": %w", err)
		}
		config.Logger.Info("Successfully updated base branch of PR from $base to $target", "base", config.Base, "target", config.Target, "pullNumber", updatedPull.GetNumber(), "pullURL", updatedPull.GetHTMLURL())
	}
	return nil
}

// GetRef returns the ref of the $target branch
func GetRef(config *gh.GitHub) (targetRef *github.Reference, err error) {
	// Get the ref of the $target branch
	ref := fmt.Sprintf("heads/%s", config.Target)
	targetRef, _, err = config.Client.Git.GetRef(config.Ctx, config.Owner, config.Repo, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get $target ref: %w", err)
	}
	if targetRef == nil {
		return nil, fmt.Errorf("no $target ref, %s, was found", config.Target)
	}
	return targetRef, nil
}

// Run updates all open PR's that point to $base to instead point to $target
// Example: Update all open PR's that point to 'master' to point to 'main'
func (c *UpdateCommand) Run(args []string) int {
	// Get a list of open PR's targetting the $base branch
	pulls, err := GetOpenPRs(c.Config)
	if err != nil {
		return c.exitError(err)
	}
	if len(pulls) == 0 {
		c.Config.Logger.Info("Exiting -- There are no open PR's to update")
		return 0
	}

	// Get the ref of the $target branch
	ref, err := GetRef(c.Config)
	if err != nil {
		return c.exitError(err)
	}

	// Update all open PR's that point to $base to point to $target
	err = UpdateOpenPRs(c.Config, pulls, ref)
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
	return `Usage: inclusify updatePulls owner repo base target token
Update the base branch of all open PR's. Configuration is pulled from the local environment.
Flags:
--owner          The GitHub org that owns the repo, e.g. 'hashicorp'.
--repo           The repository name, e.g. 'circle-codesign'.
--base="master"  The name of the current base branch, e.g. 'master'.
--target="main"  The name of the target branch, e.g. 'main'.
--token          Your Personal GitHub Access Token.
`
}

// Synopsis returns a sub 50 character summary of the command.
func (c *UpdateCommand) Synopsis() string {
	return "Update base branch of open PR's. [subcommand]"
}
