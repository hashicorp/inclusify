package branch

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/inclusify/internal/inputs"
	"github.com/mitchellh/cli"
)

// DeleteCommand is a struct used to configure a Command for deleting the
// GitHub branch, $base, in the remote repo
type DeleteCommand struct {
	UI cli.Ui
}

// Run deletes the $base branch in the remote repo
// $base defaults to "master" if no $base flag or env var is provided
// Example: Delete the 'master' branch
func (c *DeleteCommand) Run(args []string) int {
	// Validate inputs
	config, err := inputs.Validate(args)
	if err != nil {
		return c.exitError(err)
	}

	log.WithFields(log.Fields{
		"branch": config.Base,
	}).Info("Attempting to delete branch")

	// Delete Ref on GitHub
	refName := fmt.Sprintf("refs/heads/%s", config.Base)
	_, err = config.Client.Git.DeleteRef(config.Ctx, config.Owner, config.Repo, refName)
	if err != nil {
		return c.exitError(errwrap.Wrapf("Failed to delete ref: {{err}}", err))
	}

	log.WithFields(log.Fields{
		"branch": config.Base,
		"ref":    refName,
	}).Info("Success!")

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *DeleteCommand) exitError(err error) int {
	c.UI.Error(err.Error())
	return 1
}

// Help returns the full help text.
func (c *DeleteCommand) Help() string {
	return `Usage: inclusify deleteBranch owner repo base token
Delete $base branch from the given GitHub repo. Configuration is pulled from the local environment.
Flags:
--owner          The GitHub org that owns the repo, e.g. 'hashicorp'.
--repo           The repository name, e.g. 'circle-codesign'.
--base="master"  The name of the current base branch, e.g. 'master'.
--token          Your Personal GitHub Access Token.
`
}

// Synopsis returns a sub 50 character summary of the command.
func (c *DeleteCommand) Synopsis() string {
	return "Delete repo's base branch. [subcommand]"
}
