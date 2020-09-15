// +build !integration

package branches

import (
	"testing"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/mitchellh/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/gh"
)

func TestDeleteBranchesRun(t *testing.T) {
	ui := cli.NewMockUi()
	client := gh.NewMockGithubInteractor()
	branches := []string{"master"}

	config := &config.Config{
		Owner:  "hashicorp",
		Repo:   "test",
		Base:   "master",
		Target: "main",
		Token:  "token",
		Logger: hclog.New(&hclog.LoggerOptions{
			Output: ui.OutputWriter,
		}),
	}

	command := &DeleteCommand{
		Config:       config,
		GithubClient: client,
		BranchesList: branches,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, ui.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := ui.OutputWriter.String()
	assert.Contains(t, output, "Attempting to remove branch protection from branch: branch=master")
	assert.Contains(t, output, "Attempting to delete branch: branch=master")
	assert.Contains(t, output, "Success! branch has been deleted: branch=master ref=refs/heads/master")
}
