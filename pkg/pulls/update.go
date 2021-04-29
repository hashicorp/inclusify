package pulls

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/gh"
	"github.com/hashicorp/inclusify/pkg/message"
)

// UpdateCommand is a struct used to configure a Command for updating open
// PR's that target master to target the new $base
type UpdateCommand struct {
	Config       *config.Config
	GithubClient gh.GithubInteractor
}

// GetOpenPRs returns an array of all open PR's that target the $base branch
func GetOpenPRs(c *UpdateCommand) (pulls []*github.PullRequest, err error) {
	c.Config.Logger.Info("Getting all open PR's targeting the branch", "base", c.Config.Base)
	var allPulls []*github.PullRequest
	opts := &github.PullRequestListOptions{
		State:       "open",
		Base:        c.Config.Base,
		ListOptions: github.ListOptions{PerPage: 10},
	}

	// Paginate to get all open PR's and store them in 'allPulls' array
	for {
		// Create a new context per request so the timeout is not applied globally
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		pulls, resp, err := c.GithubClient.GetPRs().List(ctx, c.Config.Owner, c.Config.Repo, opts)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve all open PR's: %w", err)
		}
		allPulls = append(allPulls, pulls...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	c.Config.Logger.Info("Retrieved all open PR's targeting the branch", "base", c.Config.Base, "prCount", len(allPulls))

	return allPulls, nil
}

// UpdateOpenPRs will update all open PR's that pointed to $base to instead point to $target
// Example: Update all open PR's that point to 'master' to point to 'main'
func UpdateOpenPRs(c *UpdateCommand, pulls []*github.PullRequest, targetRef *github.Reference) (err error) {
	for _, pull := range pulls {
		pull.Base.Label = &c.Config.Target
		pull.Base.Ref = targetRef.Ref
		// Create a new context per request so the timeout is not applied globally
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		updatedPull, _, err := c.GithubClient.GetPRs().Edit(ctx, c.Config.Owner, c.Config.Repo, *pull.Number, pull)
		cancel()
		if err != nil {
			errString := fmt.Sprintf("failed to update base branch of PR %s", *pull.URL)
			return fmt.Errorf(errString+": %w", err)
		}
		c.Config.Logger.Info("Successfully updated base branch of PR to target", "base", c.Config.Base, "target", c.Config.Target, "pullNumber", updatedPull.GetNumber(), "pullURL", updatedPull.GetHTMLURL())
	}
	return nil
}

// GetRef returns the ref of the $target branch
func GetRef(c *UpdateCommand) (targetRef *github.Reference, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ref := fmt.Sprintf("heads/%s", c.Config.Target)
	targetRef, _, err = c.GithubClient.GetGit().GetRef(ctx, c.Config.Owner, c.Config.Repo, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get $target ref: %w", err)
	}
	if targetRef == nil {
		return nil, fmt.Errorf("no $target ref, %s, was found", c.Config.Target)
	}
	return targetRef, nil
}

// Run updates all open PR's that point to $base to instead point to $target
// Example: Update all open PR's that point to 'master' to point to 'main'
func (c *UpdateCommand) Run(args []string) int {
	// Get a list of open PR's targeting the $base branch
	pulls, err := GetOpenPRs(c)
	if err != nil {
		return c.exitError(err)
	}
	if len(pulls) == 0 {
		c.Config.Logger.Info(message.Info("Exiting -- There are no open PR's to update"))
		return 0
	}

	// Get the ref of the $target branch
	ref, err := GetRef(c)
	if err != nil {
		return c.exitError(err)
	}

	// Update all open PR's that point to $base to point to $target
	err = UpdateOpenPRs(c, pulls, ref)
	if err != nil {
		return c.exitError(err)
	}

	c.Config.Logger.Info(message.Success("Success!"))

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *UpdateCommand) exitError(err error) int {
	c.Config.Logger.Error(message.Error(err.Error()))
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
