package branches

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/gh"
	"github.com/hashicorp/inclusify/pkg/message"
)

// CreateCommand is a struct used to configure a Command for creating a
// new GitHub repo
type CreateCommand struct {
	Config       *config.Config
	GithubClient gh.GithubInteractor
	Repo         string
}

// Run creates a new repo for the authenticated user
func (c *CreateCommand) Run(args []string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c.Config.Logger.Info("Creating new repo for user", "repo", c.Repo, "user", c.Config.Owner)
	repositoryRequest := &github.Repository{
		Name: &c.Repo,
	}
	repo, _, err := c.GithubClient.GetRepo().Create(ctx, "", repositoryRequest)
	if err != nil {
		return c.exitError(fmt.Errorf("call to create repo returned error: %w", err))
	}

	c.Config.Logger.Info(message.Success("Successfully created new repo"), "repo", repo.GetName(), "url", repo.GetHTMLURL())

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *CreateCommand) exitError(err error) int {
	c.Config.Logger.Error(message.Error(err.Error()))
	return 1
}
