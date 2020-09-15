// +build !integration

package config

import (
	"os"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test that the config is generated properly when only env vars are set
func Test_ParseAndValidate_EnvVars(t *testing.T) {
	args := []string{"subcommand"}
	os.Setenv("INCLUSIFY_OWNER", "owner")
	os.Setenv("INCLUSIFY_REPO", "repo")
	os.Setenv("INCLUSIFY_BASE", "base")
	os.Setenv("INCLUSIFY_TARGET", "target")
	os.Setenv("INCLUSIFY_TOKEN", "token")

	ui := &cli.BasicUi{}
	config, err := ParseAndValidate(args, ui)
	require.NoError(t, err)

	// Make some assertions about the UI output
	assert.Equal(t, os.Getenv("INCLUSIFY_OWNER"), config.Owner)
	assert.Equal(t, os.Getenv("INCLUSIFY_REPO"), config.Repo)
	assert.Equal(t, os.Getenv("INCLUSIFY_BASE"), config.Base)
	assert.Equal(t, os.Getenv("INCLUSIFY_TARGET"), config.Target)
	assert.Equal(t, os.Getenv("INCLUSIFY_TOKEN"), config.Token)
}

// Test that the config is generated properly when cmd line flags are passed in
func Test_ParseAndValidate_Flags(t *testing.T) {
	os.Unsetenv("INCLUSIFY_OWNER")
	os.Unsetenv("INCLUSIFY_REPO")
	os.Unsetenv("INCLUSIFY_TOKEN")
	os.Unsetenv("INCLUSIFY_BASE")
	os.Unsetenv("INCLUSIFY_TARGET")

	owner := "hashicorp"
	repo := "inclusify"
	token := "github_token"

	args := []string{"subcommand", "--owner", owner, "--repo", repo, "--token", token}

	ui := &cli.BasicUi{}
	config, err := ParseAndValidate(args, ui)
	require.NoError(t, err)

	assert.Equal(t, owner, config.Owner)
	assert.Equal(t, repo, config.Repo)
	assert.Equal(t, "master", config.Base)
	assert.Equal(t, "main", config.Target)
	assert.Equal(t, token, config.Token)
}
