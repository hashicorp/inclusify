package pulls

import (
	"context"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/gh"
)

// CloseCommand is a struct used to configure a Command for closing an open PR
type CloseCommand struct {
	Config       *config.Config
	GithubClient gh.GithubInteractor
	PullNumber   int
}

// Run closes an open PR on GitHub, given a PullNumber
func (c *CloseCommand) Run(args []string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr := &github.PullRequest{Number: &c.PullNumber, Base: nil}
	_, _, err := c.GithubClient.GetPRs().Edit(ctx, c.Config.Owner, c.Config.Repo, c.PullNumber, pr)
	if err != nil {
		return c.exitError(err)
	}

	c.Config.Logger.Info("Successfully closed PR", "number", c.PullNumber)

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *CloseCommand) exitError(err error) int {
	c.Config.Logger.Error(err.Error())
	return 1
}
