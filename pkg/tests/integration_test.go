// +build integration

package tests

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"
	"testing"

	"github.com/dchest/uniuri"
	"github.com/mitchellh/cli"

	branches "github.com/hashicorp/inclusify/pkg/branches"
	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/files"
	"github.com/hashicorp/inclusify/pkg/gh"
	"github.com/hashicorp/inclusify/pkg/pulls"
	repos "github.com/hashicorp/inclusify/pkg/repos"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var seqMutex sync.Mutex
var i Inputs

// Inputs is a struct that contains all of our test config values
type Inputs struct {
	owner          string
	repo           string
	token          string
	base           string
	target         string
	exclusion      string
	temp           string
	random         string
	branchesList   []string
	pullRequestURL string
}

// SetVals sets test config values that will be used in all integration tests
func (i *Inputs) SetVals(t *testing.T) {
	owner, exists := os.LookupEnv("INCLUSIFY_OWNER")
	if exists != true {
		t.Errorf("Cannot find the required env var INCLUSIFY_OWNER")
	}
	token, exists := os.LookupEnv("INCLUSIFY_TOKEN")
	if exists != true {
		t.Errorf("Cannot find the required env var INCLUSIFY_TOKEN")
	}
	i.owner = owner
	i.token = token
	i.repo = fmt.Sprintf("inclusify-tests-%s", uniuri.NewLenChars(8, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")))
	i.base = "master-clone"
	i.target = "main"
	i.exclusion = "scripts/,.teamcity.yml"
	i.temp = "update-references"
	i.random = "my-fancy-branch"
}

// SetURL receives a pointer to Inputs so it can modify the pullRequestURL field
func (i *Inputs) SetURL(url string) {
	i.pullRequestURL = url
}

// GetVals receives a copy of Inputs and returns the structs values.
func (i Inputs) GetVals() (string, string, string, string, string, string, string) {
	return i.owner, i.repo, i.token, i.base, i.target, i.temp, i.random
}

// GetURL recieces a copy of the pullRequestURL field and returns its value
func (i Inputs) GetURL() string {
	return i.pullRequestURL
}

// Seq is used to ensure all tests run in sequence
func seq() func() {
	seqMutex.Lock()
	return func() {
		seqMutex.Unlock()
	}
}

// TestSetTestValues sets config values to use in our integration tests
func Test_SetTestConfigValues(t *testing.T) {
	defer seq()()
	i.SetVals(t)
}

// Test_CreateRepository creates a new repository for the currently authenticated user
func Test_CreateRepository(t *testing.T) {
	defer seq()()
	mockUI := cli.NewMockUi()
	owner, repo, token, base, target, _, _ := i.GetVals()
	args := []string{"", "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	require.NoError(t, err)

	client, err := gh.NewBaseGithubInteractor(token)
	require.NoError(t, err)

	command := &repos.CreateCommand{
		Config:       config,
		GithubClient: client,
		Repo:         repo,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, fmt.Sprintf("Creating new repo for user: repo=%s user=%s", repo, owner))
	assert.Contains(t, output, fmt.Sprintf("Successfully created new repo: repo=%s url=", repo))
}

// Test_CreateScaffold creates an initial commit in the newly created repository
func Test_CreateScaffold(t *testing.T) {
	defer seq()()
	mockUI := cli.NewMockUi()
	owner, repo, token, base, target, _, _ := i.GetVals()
	base = "master"
	args := []string{"", "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	require.NoError(t, err)

	client, err := gh.NewBaseGithubInteractor(token)
	require.NoError(t, err)

	command := &files.CreateScaffoldCommand{
		Config:       config,
		GithubClient: client,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, "Creating local temp dir: dirPrefix=tmp-clone-")
	assert.Contains(t, output, "Initializing new repo at dir: dir=")
	assert.Contains(t, output, fmt.Sprintf("Creating a new remote for base: base=%s", base))
	assert.Contains(t, output, "Copying test CI files into temp directory")
	assert.Contains(t, output, "Running `git add")
	assert.Contains(t, output, "Committing changes")
	assert.Contains(t, output, fmt.Sprintf("Pushing initial commit to remote: branch=%s sha=", base))
	assert.Contains(t, output, fmt.Sprintf("Creating branch protection request: branch=%s", base))
	assert.Contains(t, output, fmt.Sprintf("Applying branch protection: branch=%s", base))

	assert.NotContains(t, output, "failed to commit changes")
	assert.NotContains(t, output, "failed to push changes")
	assert.NotContains(t, output, "failed to create the base branch protection")
}

// Test_CreateBranches creates the master-clone branch,
// update-references branch, and the main branch, off of the head of master
func Test_CreateBranches(t *testing.T) {
	defer seq()()
	subcommand := "createBranches"
	mockUI := cli.NewMockUi()
	owner, repo, token, base, target, temp, random := i.GetVals()
	list := []string{base, temp, random}
	base = "master"
	args := []string{subcommand, "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	require.NoError(t, err)

	client, err := gh.NewBaseGithubInteractor(token)
	require.NoError(t, err)

	command := &branches.CreateCommand{
		Config:       config,
		GithubClient: client,
		BranchesList: list,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, fmt.Sprintf("Creating new branch %s off of %s", list[0], base))
	assert.Contains(t, output, fmt.Sprintf("Creating new branch %s off of %s", list[1], base))
	assert.Contains(t, output, fmt.Sprintf("Creating new branch %s off of %s", list[2], base))
	assert.Contains(t, output, fmt.Sprintf("Creating new branch %s off of %s", target, base))
	assert.Contains(t, output, "Success!")
}

// Test_UpdateOpenPullRequestsNoOp updates any open pull requests that have 'main' as a base
// Since there are no open PR's targeting that base, this will effectively do nothing
func Test_UpdateOpenPullRequestsNoOp(t *testing.T) {
	defer seq()()
	mockUI := cli.NewMockUi()
	owner, repo, token, base, target, _, _ := i.GetVals()
	args := []string{"updatePulls", "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	require.NoError(t, err)

	client, err := gh.NewBaseGithubInteractor(token)
	require.NoError(t, err)

	command := &pulls.UpdateCommand{
		Config:       config,
		GithubClient: client,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, "Exiting -- There are no open PR's to update")
}

// Test_UpdateRefs finds and replaces all references of 'master' to 'main' in the given CI files
// in the 'update-references' branch, and opens a PR to merge changes from 'update-references' to 'main'
// No files/dirs in `exclusions` is considered.
func Test_UpdateRefs(t *testing.T) {
	defer seq()()
	mockUI := cli.NewMockUi()
	base := "master"
	owner, repo, token, _, target, temp, _ := i.GetVals()
	args := []string{"updateRefs", "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	require.NoError(t, err)

	client, err := gh.NewBaseGithubInteractor(token)
	require.NoError(t, err)

	command := &files.UpdateRefsCommand{
		Config:       config,
		GithubClient: client,
		TempBranch:   temp,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, fmt.Sprintf("Successfully cloned repo into local dir: repo=%s dir=", repo))
	assert.Contains(t, output, fmt.Sprintf("Retrieved HEAD commit of branch: branch=%s", temp))
	assert.Contains(t, output, "Finding and replacing all references from base to target in dir")
	assert.Contains(t, output, "Finding and replacing all references from base to target in dir")
	assert.Contains(t, output, "Running `git add")
	assert.Contains(t, output, "Committing changes")
	assert.Contains(t, output, fmt.Sprintf("Pushing commit to remote: branch=%s sha=", temp))
	assert.Contains(t, output, fmt.Sprintf("Creating PR to merge changes from branch into target: branch=%s target=%s", temp, target))
	assert.Contains(t, output, fmt.Sprintf("Success! Review and merge the open PR: url=https://github.com/%s/%s/pull/", owner, repo))

	// Extract pull request URL from output
	scheme := fmt.Sprintf(`(https:\/\/github.com\/%s\/%s\/pull\/)\d*`, owner, repo)
	r, err := regexp.Compile(scheme)
	if err != nil {
		t.Errorf("REGEX pattern did not compile: %v", err)
	}
	url := r.FindString(output)
	i.SetURL(url)
}

// Test_MergePullRequest merges the pull request created in TestUpdateRefs()
func Test_MergePullRequest(t *testing.T) {
	defer seq()()
	pullRequestURL := i.GetURL()
	mockUI := cli.NewMockUi()
	owner, repo, token, base, target, _, _ := i.GetVals()
	args := []string{"", "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	require.NoError(t, err)

	// Extract pull request number from URL
	r, err := regexp.Compile(`[-]?\d[\d,]*[\.]?[\d{2}]*`)
	require.NoError(t, err)

	result := r.FindString(pullRequestURL)
	prNumber, err := strconv.Atoi(result)
	require.NoError(t, err)

	client, err := gh.NewBaseGithubInteractor(token)
	require.NoError(t, err)

	command := &pulls.MergeCommand{
		Config:       config,
		GithubClient: client,
		PullNumber:   prNumber,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, fmt.Sprintf("Successfully merged PR: number=%d", prNumber))
}

// Test_CreateOpenPullRequest finds and replaces all references of 'master' to 'master-clone'
// in the given CI files, and pushes the changes to 'my-fancy-branch' branch + opens a PR.
// This will let us test that we can successfully update the base branch of an open PR
func Test_CreateOpenPullRequest(t *testing.T) {
	defer seq()()
	mockUI := cli.NewMockUi()
	owner, repo, token, base, _, _, random := i.GetVals()
	args := []string{"updateRefs", "--owner", owner, "--repo", repo, "--base", "master", "--target", base, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	require.NoError(t, err)

	client, err := gh.NewBaseGithubInteractor(token)
	require.NoError(t, err)

	command := &files.UpdateRefsCommand{
		Config:       config,
		GithubClient: client,
		TempBranch:   random,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, fmt.Sprintf("Successfully cloned repo into local dir: repo=%s dir=", repo))
	assert.Contains(t, output, fmt.Sprintf("Retrieved HEAD commit of branch: branch=%s", random))
	assert.Contains(t, output, "Finding and replacing all references from base to target in dir")
	assert.Contains(t, output, "Running `git add")
	assert.Contains(t, output, "Committing changes")
	assert.Contains(t, output, fmt.Sprintf("Pushing commit to remote: branch=%s sha=", random))
	assert.Contains(t, output, fmt.Sprintf("Creating PR to merge changes from branch into target: branch=%s target=%s", random, "master"))
	assert.Contains(t, output, fmt.Sprintf("Success! Review and merge the open PR: url=https://github.com/%s/%s/pull/", owner, repo))

	// Extract pull request URL from output
	scheme := fmt.Sprintf(`(https:\/\/github.com\/%s\/%s\/pull\/)\d*`, owner, repo)
	r, err := regexp.Compile(scheme)
	if err != nil {
		t.Errorf("REGEX pattern did not compile: %v", err)
	}
	url := r.FindString(output)
	i.SetURL(url)
}

// Test_UpdateOpenPullRequests updates the base of the open pull request
// created by TestCreateOpenPullRequest()
func Test_UpdateOpenPullRequests(t *testing.T) {
	defer seq()()
	mockUI := cli.NewMockUi()
	owner, repo, token, base, target, _, _ := i.GetVals()
	args := []string{"updatePulls", "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	require.NoError(t, err)

	client, err := gh.NewBaseGithubInteractor(token)
	require.NoError(t, err)

	command := &pulls.UpdateCommand{
		Config:       config,
		GithubClient: client,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, fmt.Sprintf("Getting all open PR's targeting the branch: base=%s", base))
	assert.Contains(t, output, fmt.Sprintf("Retrieved all open PR's targeting the branch: base=%s", base))
	assert.Contains(t, output, fmt.Sprintf("Successfully updated base branch of PR to target: base=%s target=%s", base, target))
	assert.Contains(t, output, "Success!")
}

// Test_CloseOpenPullRequest closes the pull request created in TestUpdateOpenPullRequests() to cleanup
func Test_CloseOpenPullRequest(t *testing.T) {
	defer seq()()
	pullRequestURL := i.GetURL()
	mockUI := cli.NewMockUi()
	owner, repo, token, base, target, _, _ := i.GetVals()
	args := []string{"", "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	require.NoError(t, err)

	// Extract pull request number from URL
	r, err := regexp.Compile(`[-]?\d[\d,]*[\.]?[\d{2}]*`)
	require.NoError(t, err)

	result := r.FindString(pullRequestURL)
	prNumber, err := strconv.Atoi(result)
	require.NoError(t, err)

	client, err := gh.NewBaseGithubInteractor(token)
	if err != nil {
		t.Errorf("Failed to create client due to error: %v", err)
	}

	command := &pulls.CloseCommand{
		Config:       config,
		GithubClient: client,
		PullNumber:   prNumber,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, "Successfully closed PR: number=")
}

// Test_CreateBaseBranchProtection copies the branch protection from 'master' to 'master-clone'
func Test_CreateBaseBranchProtection(t *testing.T) {
	defer seq()()
	mockUI := cli.NewMockUi()
	owner, repo, token, base, _, _, _ := i.GetVals()
	args := []string{"", "--owner", owner, "--repo", repo, "--base", base, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	require.NoError(t, err)

	client, err := gh.NewBaseGithubInteractor(token)
	require.NoError(t, err)

	c := &branches.UpdateCommand{
		Config:       config,
		GithubClient: client,
	}

	err = branches.CopyBranchProtection(c, "master", base)
	require.NoError(t, err)

	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, "Getting branch protection for branch: branch=master")
	assert.Contains(t, output, fmt.Sprintf("Creating the branch protection request for branch: branch=%s", base))
	assert.Contains(t, output, fmt.Sprintf("Updating the branch protection on branch: branch=%s", base))
}

// Test_UpdateDefaultBranch updates the default branch in the repo from 'master' to 'main'
func Test_UpdateDefaultBranch(t *testing.T) {
	defer seq()()
	mockUI := cli.NewMockUi()
	owner, repo, token, base, target, _, _ := i.GetVals()
	args := []string{"updateDefault", "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	require.NoError(t, err)

	client, err := gh.NewBaseGithubInteractor(token)
	require.NoError(t, err)

	command := &branches.UpdateCommand{
		Config:       config,
		GithubClient: client,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, fmt.Sprintf("Updating the default branch to target: repo=%s base=%s target=%s", repo, base, target))
	assert.Contains(t, output, fmt.Sprintf("Attempting to apply the base branch protection to target: base=%s target=%s", base, target))
	assert.Contains(t, output, fmt.Sprintf("Getting branch protection for branch: branch=%s", base))
	assert.Contains(t, output, fmt.Sprintf("Creating the branch protection request for branch: branch=%s", target))
	assert.Contains(t, output, fmt.Sprintf("Updating the branch protection on branch: branch=%s", target))
	assert.Contains(t, output, "Success!")
}

// Test_DeleteTestBranches deletes the test branches we created in this test suite (including their branch protections)
func Test_DeleteTestBranches(t *testing.T) {
	defer seq()()
	mockUI := cli.NewMockUi()
	owner, repo, token, base, _, temp, random := i.GetVals()
	args := []string{"deleteBranches", "--owner", owner, "--repo", repo, "--base", base, "--token", token}
	list := []string{temp, random, "master"}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	require.NoError(t, err)

	client, err := gh.NewBaseGithubInteractor(token)
	require.NoError(t, err)

	command := &branches.DeleteCommand{
		Config:       config,
		GithubClient: client,
		BranchesList: list,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	for _, branch := range list {
		assert.Contains(t, output, fmt.Sprintf("Attempting to remove branch protection from branch: branch=%s", branch))
		assert.Contains(t, output, fmt.Sprintf("Attempting to delete branch: branch=%s", branch))
		assert.Contains(t, output, fmt.Sprintf("Success! branch has been deleted: branch=%s", branch))
	}
}

// Test_DeleteRepo deletes the repo that was created in this test suite
func Test_DeleteRepo(t *testing.T) {
	defer seq()()
	mockUI := cli.NewMockUi()
	owner, repo, token, base, target, _, _ := i.GetVals()
	args := []string{"", "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	require.NoError(t, err)

	client, err := gh.NewBaseGithubInteractor(token)
	require.NoError(t, err)

	command := &repos.DeleteCommand{
		Config:       config,
		GithubClient: client,
		Repo:         repo,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, fmt.Sprintf("Deleting repo for user: repo=%s user=%s", repo, owner))
	assert.Contains(t, output, fmt.Sprintf("Successfully deleted repo: repo=%s", repo))
}
