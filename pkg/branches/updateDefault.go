package branch

import (
	github "github.com/google/go-github/v32/github"
	log "github.com/sirupsen/logrus"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/inclusify/internal/inputs"
	"github.com/mitchellh/cli"
)

// UpdateCommand is a struct used to configure a Command for updating
// the GitHub default branch in the remote repo
type UpdateCommand struct {
	UI cli.Ui
}

// Run updates the default branch in the repo to the new $target branch
// Example: Update the repo's default branch from 'master' to 'main'
func (c *UpdateCommand) Run(args []string) int {
	// Validate inputs
	config, err := inputs.Validate(args)
	if err != nil {
		return c.exitError(err)
	}

	// Update the default branch to $target on the remote GitHub repo
	log.WithFields(log.Fields{
		"repo":   config.Repo,
		"target": config.Target,
	}).Info("Updating the default branch in $repo to $target")
	editRepo := &github.Repository{DefaultBranch: &config.Target}
	_, _, err = config.Client.Repositories.Edit(config.Ctx, config.Owner, config.Repo, editRepo)

	if err != nil {
		return c.exitError(errwrap.Wrapf("Failed to update default branch: {{err}}", err))
	}

	log.Info("Success!")

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *UpdateCommand) exitError(err error) int {
	c.UI.Error(err.Error())
	return 1
}

// Help returns the full help text.
func (c *UpdateCommand) Help() string {
	return `Usage: inclusify updateDefault owner repo target token
Update the default branch in the repo to $target. Configuration is pulled from the local environment.
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
