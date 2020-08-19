package branches

import (
	"testing"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/gh"
	"github.com/mitchellh/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBranchRun(t *testing.T) {
	ui := cli.NewMockUi()
	client := gh.NewMockGithubInteractor()
	branches := []string{"main", "update-ci-references"}

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

	command := &CreateCommand{
		Config:       config,
		GithubClient: client,
		BaseBranch:   "master",
		BranchesList: branches,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, ui.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := ui.OutputWriter.String()
	assert.Contains(t, output, "Creating new branch update-ci-references off of master")
	assert.Contains(t, output, "Creating new branch main off of master")
	assert.Contains(t, output, "Success!")
}
