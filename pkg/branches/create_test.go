package branch

import (
	"testing"

	"github.com/google/go-github/v32/github"
	"github.com/hashicorp/inclusify/pkg/gh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mitchellh/cli"
)

func TestCreateBranchRun(t *testing.T) {
	ui := cli.NewMockUi()
	gh := gh.NewMockGithubInteractor()

	command := &CreateCommand{
		UI:           ui,
		GithubClient: gh,

		Owner: "hashicorp",
		Repo:  "test",

		base:   "master",
		target: "main",
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, ui.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := ui.OutputWriter.String()
	assert.Contains(t, output, "Creating new branch main off of master")
	assert.Contains(t, output, "Creating new temp branch update-ci-references off of master")
	assert.Contains(t, output, "Success!")

	// Make some assertions about what we wrote to GitHub
	created := gh.CreatedReferences
	assert.Len(t, created, 2)

	want := []*github.Reference{
		{
			Ref:    github.String("refs/heads/main"),
			Object: &github.GitObject{SHA: &gh.MasterRef},
		},
		{
			Ref:    github.String("refs/heads/update-ci-references"),
			Object: &github.GitObject{SHA: &gh.MasterRef},
		},
	}

	for i, c := range created {
		assert.Equal(t, want[i].GetRef(), c.GetRef())
		assert.Equal(t, want[i].Object.GetSHA(), c.Object.GetSHA())
	}
}
