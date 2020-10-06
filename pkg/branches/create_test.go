// +build !integration

package branches

import (
	"testing"

	"github.com/google/go-github/github"
	hclog "github.com/hashicorp/go-hclog"
	"github.com/mitchellh/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/gh"
)

func TestCreateBranchRun(t *testing.T) {
	ui := cli.NewMockUi()
	client := gh.NewMockGithubInteractor()
	branches := []string{"update-references"}

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
		BranchesList: branches,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, ui.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := ui.OutputWriter.String()
	assert.Contains(t, output, "Creating new branch update-references off of master")
	assert.Contains(t, output, "Creating new branch main off of master")
	assert.Contains(t, output, "Success!")

	// Make some assertions about what we wrote to GitHub
	created := client.CreatedReferences
	assert.Len(t, created, 2)

	want := []*github.Reference{
		{
			Ref:    github.String("refs/heads/update-references"),
			Object: &github.GitObject{SHA: &client.MasterRef},
		},
		{
			Ref:    github.String("refs/heads/main"),
			Object: &github.GitObject{SHA: &client.MasterRef},
		},
	}

	for i, c := range created {
		assert.Equal(t, want[i].GetRef(), c.GetRef())
		assert.Equal(t, want[i].Object.GetSHA(), c.Object.GetSHA())
	}
}
