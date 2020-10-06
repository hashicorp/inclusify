// +build !integration

package config

import (
	"os"
	"strings"
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
	os.Setenv("INCLUSIFY_EXCLUSION", ".circleci/,scripts/hello.py,.teamcity.yml")
	exclusionArr := strings.Split(os.Getenv("INCLUSIFY_EXCLUSION"), ",")

	ui := &cli.BasicUi{}
	config, err := ParseAndValidate(args, ui)
	require.NoError(t, err)

	// Make some assertions about the UI output
	assert.Equal(t, os.Getenv("INCLUSIFY_OWNER"), config.Owner)
	assert.Equal(t, os.Getenv("INCLUSIFY_REPO"), config.Repo)
	assert.Equal(t, os.Getenv("INCLUSIFY_BASE"), config.Base)
	assert.Equal(t, os.Getenv("INCLUSIFY_TARGET"), config.Target)
	assert.Equal(t, os.Getenv("INCLUSIFY_TOKEN"), config.Token)
	assert.Equal(t, config.Exclusion, exclusionArr)
}

// Test that the config is generated properly when cmd line flags are passed in
func Test_ParseAndValidate_Flags(t *testing.T) {
	os.Unsetenv("INCLUSIFY_OWNER")
	os.Unsetenv("INCLUSIFY_REPO")
	os.Unsetenv("INCLUSIFY_TOKEN")
	os.Unsetenv("INCLUSIFY_BASE")
	os.Unsetenv("INCLUSIFY_TARGET")
	os.Unsetenv("INCLUSIFY_EXCLUSION")

	owner := "hashicorp"
	repo := "inclusify"
	token := "github_token"
	exclusion := ".circleci/,scripts/hello.py,.teamcity.yml"
	exclusionArr := strings.Split(exclusion, ",")

	args := []string{"subcommand", "--owner", owner, "--repo", repo, "--token", token, "--exclusion", exclusion}

	ui := &cli.BasicUi{}
	config, err := ParseAndValidate(args, ui)
	require.NoError(t, err)

	assert.Equal(t, owner, config.Owner)
	assert.Equal(t, repo, config.Repo)
	assert.Equal(t, "master", config.Base)
	assert.Equal(t, "main", config.Target)
	assert.Equal(t, token, config.Token)
	assert.Equal(t, exclusionArr, config.Exclusion)
}
