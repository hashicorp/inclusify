package branches

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/gh"
	"github.com/hashicorp/inclusify/pkg/message"
)

// DeleteCommand is a struct used to configure a Command for deleting a
// GitHub repo
type DeleteCommand struct {
	Config       *config.Config
	GithubClient gh.GithubInteractor
	Repo         string
}

// Run deletes the repo for the authenticated user
func (c *DeleteCommand) Run(args []string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c.Config.Logger.Info("Deleting repo for user", "repo", c.Repo, "user", c.Config.Owner)

	_, err := c.GithubClient.GetRepo().Delete(ctx, c.Config.Owner, c.Repo)
	if err != nil {
		return c.exitError(fmt.Errorf("call to delete repo returned error: %w", err))
	}

	c.Config.Logger.Info(message.Success("Successfully deleted repo"), "repo", c.Repo)

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *DeleteCommand) exitError(err error) int {
	c.Config.Logger.Error(message.Error(err.Error()))
	return 1
}
