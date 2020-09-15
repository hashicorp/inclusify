package pulls

import (
	"context"
	"errors"
	"time"

	"github.com/google/go-github/v32/github"

	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/gh"
)

// MergeCommand is a struct used to configure a Command for merging an open PR
type MergeCommand struct {
	Config       *config.Config
	GithubClient gh.GithubInteractor
	PullNumber   int
}

// Run merges an open PR on GitHub, given a PullNumber
func (c *MergeCommand) Run(args []string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	options := &github.PullRequestOptions{MergeMethod: "squash"}
	result, _, err := c.GithubClient.GetPRs().Merge(ctx, c.Config.Owner, c.Config.Repo, c.PullNumber, "Merging Inclusify PR", options)
	if err != nil {
		return c.exitError(err)
	}
	if !*result.Merged {
		return c.exitError(errors.New("Failed to merge PR"))
	}

	c.Config.Logger.Info("Successfully merged PR", "number", c.PullNumber)

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *MergeCommand) exitError(err error) int {
	c.Config.Logger.Error(err.Error())
	return 1
}
