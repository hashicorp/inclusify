package tests

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"
	"testing"

	"github.com/dchest/uniuri"
	"github.com/hashicorp/inclusify/pkg/branches"
	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/files"
	"github.com/hashicorp/inclusify/pkg/gh"
	"github.com/hashicorp/inclusify/pkg/pulls"
	"github.com/mitchellh/cli"
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
	temp           string
	random         string
	branchesList   []string
	pullRequestURL string
}

// SetVals sets test config values that will be used in all integration tests
func (i *Inputs) SetVals() {
	i.owner = os.Getenv("INCLUSIFY_OWNER")
	i.repo = os.Getenv("INCLUSIFY_REPO")
	i.token = os.Getenv("INCLUSIFY_TOKEN")
	i.base = fmt.Sprintf("master-clone-%s", uniuri.NewLen(8))
	i.target = fmt.Sprintf("main-%s", uniuri.NewLen(8))
	i.temp = fmt.Sprintf("update-ci-references-%s", uniuri.NewLen(8))
	i.random = fmt.Sprintf("my-fancy-branch-%s", uniuri.NewLen(8))
	i.branchesList = []string{i.base, i.target, i.temp, i.random}
}

// SetURL receives a pointer to Inputs so it can modify the pullRequestURL field
func (i *Inputs) SetURL(url string) {
	i.pullRequestURL = url
}

// GetVals receives a copy of Inputs and returns the structs values.
func (i Inputs) GetVals() (string, string, string, string, string, string, string, []string) {
	return i.owner, i.repo, i.token, i.base, i.target, i.temp, i.random, i.branchesList
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
	i.SetVals()
}

// Test_CreateMasterClone creates a clone of master called $base, e.g. master-clone-%s
func Test_CreateMasterClone(t *testing.T) {
	defer seq()()
	subcommand := ""
	mockUI := cli.NewMockUi()
	owner, repo, token, base, _, _, _, _ := i.GetVals()
	branchesList := []string{base}
	args := []string{subcommand, "--owner", owner, "--repo", repo, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	if err != nil {
		t.Errorf("Failed to create config due to error: %v", err)
	}

	client, err := gh.NewBaseGithubInteractor(token)
	if err != nil {
		t.Errorf("Failed to create client due to error: %v", err)
	}

	command := &branches.CreateCommand{
		Config:       config,
		GithubClient: client,
		BaseBranch:   "master",
		BranchesList: branchesList,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, fmt.Sprintf("Creating new branch %s off of %s", base, "master"))
	assert.Contains(t, output, "Success!")
}

// Test_CreateBranches creates the update-ci-references-%s branch,
// and the main-%s branch, off of the head of master
func Test_CreateBranches(t *testing.T) {
	defer seq()()
	subcommand := "createBranches"
	mockUI := cli.NewMockUi()
	base := "master"
	owner, repo, token, _, target, temp, random, _ := i.GetVals()
	branchesList := []string{temp, target, random}
	args := []string{subcommand, "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	if err != nil {
		t.Errorf("Failed to create config due to error: %v", err)
	}

	client, err := gh.NewBaseGithubInteractor(token)
	if err != nil {
		t.Errorf("Failed to create client due to error: %v", err)
	}

	command := &branches.CreateCommand{
		Config:       config,
		GithubClient: client,
		BaseBranch:   base,
		BranchesList: branchesList,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, fmt.Sprintf("Creating new branch %s off of %s", branchesList[0], base))
	assert.Contains(t, output, fmt.Sprintf("Creating new branch %s off of %s", branchesList[1], base))
	assert.Contains(t, output, fmt.Sprintf("Creating new branch %s off of %s", branchesList[2], base))
	assert.Contains(t, output, "Success!")
}

// Test_UpdateOpenPullRequestsNoOp updates any open pull requests that have 'main-clone-*' as a base
// Since there are no open PR's targetting that base, this will effectively do nothing
func Test_UpdateOpenPullRequestsNoOp(t *testing.T) {
	defer seq()()
	subcommand := "updatePulls"
	mockUI := cli.NewMockUi()
	owner, repo, token, base, target, _, _, _ := i.GetVals()
	args := []string{subcommand, "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	if err != nil {
		t.Errorf("Failed to create config due to error: %v", err)
	}

	client, err := gh.NewBaseGithubInteractor(token)
	if err != nil {
		t.Errorf("Failed to create client due to error: %v", err)
	}

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

// Test_UpdateCI finds and replaces all references of 'master' to 'main-*' in the given CI files
// in the 'update-ci-references-*' branch, and opens a PR to merge changes
// from 'update-ci-references-*' to 'main-*'
func Test_UpdateCI(t *testing.T) {
	defer seq()()
	subcommand := "updateCI"
	mockUI := cli.NewMockUi()
	base := "master"
	owner, repo, token, _, target, temp, _, _ := i.GetVals()
	args := []string{subcommand, "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	if err != nil {
		t.Errorf("Failed to create config due to error: %v", err)
	}

	client, err := gh.NewBaseGithubInteractor(token)
	if err != nil {
		t.Errorf("Failed to create client due to error: %v", err)
	}

	command := &files.UpdateCICommand{
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
	assert.Contains(t, output, "Finding and replacing all refs from base to target in dir/paths")
	assert.Contains(t, output, "Running `git add .`")
	assert.Contains(t, output, "Committing changes")
	assert.Contains(t, output, fmt.Sprintf("Pushing commit to remote: branch=%s sha=", temp))
	assert.Contains(t, output, fmt.Sprintf("Creating PR to merge CI changes from branch into target: branch=%s target=%s", temp, target))
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

// Test_MergePullRequest merges the pull request created in TestUpdateCI()
func Test_MergePullRequest(t *testing.T) {
	defer seq()()
	pullRequestURL := i.GetURL()
	subcommand := ""
	mockUI := cli.NewMockUi()
	owner, repo, token, base, target, _, _, _ := i.GetVals()
	args := []string{subcommand, "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	if err != nil {
		t.Errorf("Failed to create config due to error: %v", err)
	}

	// Extract pull request number from URL
	r, err := regexp.Compile(`[-]?\d[\d,]*[\.]?[\d{2}]*`)
	if err != nil {
		t.Errorf("REGEX pattern did not compile: %v", err)
	}
	result := r.FindString(pullRequestURL)
	prNumber, err := strconv.Atoi(result)
	if err != nil {
		t.Errorf("Failed to convert pull request number into an integer")
	}

	client, err := gh.NewBaseGithubInteractor(token)
	if err != nil {
		t.Errorf("Failed to create client due to error: %v", err)
	}

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

// Test_CreateOpenPullRequest finds and replaces all references of 'master' to 'master-clone-*'
// in the given CI files, and pushes the changes to 'my-fancy-branch-*' branch + opens a PR.
// This will let us test that we can successfully update the base branch of an open PR
func Test_CreateOpenPullRequest(t *testing.T) {
	defer seq()()
	subcommand := "updateCI"
	mockUI := cli.NewMockUi()
	owner, repo, token, base, _, _, random, _ := i.GetVals()
	args := []string{subcommand, "--owner", owner, "--repo", repo, "--base", "master", "--target", base, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	if err != nil {
		t.Errorf("Failed to create config due to error: %v", err)
	}

	client, err := gh.NewBaseGithubInteractor(token)
	if err != nil {
		t.Errorf("Failed to create client due to error: %v", err)
	}

	command := &files.UpdateCICommand{
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
	assert.Contains(t, output, "Finding and replacing all refs from base to target in dir/paths")
	assert.Contains(t, output, "Running `git add .`")
	assert.Contains(t, output, "Committing changes")
	assert.Contains(t, output, fmt.Sprintf("Pushing commit to remote: branch=%s sha=", random))
	assert.Contains(t, output, fmt.Sprintf("Creating PR to merge CI changes from branch into target: branch=%s target=%s", random, "master"))
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
	subcommand := "updatePulls"
	mockUI := cli.NewMockUi()
	owner, repo, token, base, target, _, _, _ := i.GetVals()
	args := []string{subcommand, "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	if err != nil {
		t.Errorf("Failed to create config due to error: %v", err)
	}

	client, err := gh.NewBaseGithubInteractor(token)
	if err != nil {
		t.Errorf("Failed to create client due to error: %v", err)
	}

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
	assert.Contains(t, output, fmt.Sprintf("Getting all open PR's targetting the branch: base=%s", base))
	assert.Contains(t, output, fmt.Sprintf("Retrieved all open PR's targetting the branch: base=%s", base))
	assert.Contains(t, output, fmt.Sprintf("Successfully updated base branch of PR to target: base=%s target=%s", base, target))
	assert.Contains(t, output, "Success!")
}

// Test_CloseOpenPullRequest closes the pull request created in TestUpdateOpenPullRequests() to cleanup
func Test_CloseOpenPullRequest(t *testing.T) {
	defer seq()()
	pullRequestURL := i.GetURL()
	subcommand := ""
	mockUI := cli.NewMockUi()
	owner, repo, token, base, target, _, _, _ := i.GetVals()
	args := []string{subcommand, "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	if err != nil {
		t.Errorf("Failed to create config due to error: %v", err)
	}

	// Extract pull request number from URL
	r, err := regexp.Compile(`[-]?\d[\d,]*[\.]?[\d{2}]*`)
	if err != nil {
		t.Errorf("REGEX pattern did not compile: %v", err)
	}
	result := r.FindString(pullRequestURL)
	prNumber, err := strconv.Atoi(result)
	if err != nil {
		t.Errorf("Failed to convert pull request number into an integer")
	}

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

// Test_CreateBaseBranchProtection copies the branch protection from 'master' to 'master-clone-*'
func Test_CreateBaseBranchProtection(t *testing.T) {
	defer seq()()
	subcommand := ""
	mockUI := cli.NewMockUi()
	owner, repo, token, base, _, _, _, _ := i.GetVals()
	args := []string{subcommand, "--owner", owner, "--repo", repo, "--base", base, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	if err != nil {
		t.Errorf("Failed to create config due to error: %v", err)
	}

	client, err := gh.NewBaseGithubInteractor(token)
	if err != nil {
		t.Errorf("Failed to create client due to error: %v", err)
	}

	c := &branches.UpdateCommand{
		Config:       config,
		GithubClient: client,
	}

	err = branches.CopyBranchProtection(c, "master", base)
	if err != nil {
		t.Errorf("Failed to copy branch protection from master to %s due to error: %v", base, err)
	}

	output := mockUI.OutputWriter.String()
	assert.Contains(t, output, "Getting branch protection for branch: branch=master")
	assert.Contains(t, output, fmt.Sprintf("Creating the branch protection request for branch: branch=%s", base))
	assert.Contains(t, output, fmt.Sprintf("Updating the branch protection on branch: branch=%s", base))
}

// Test_UpdateDefaultBranch updates the default branch in the repo from 'master' to 'main-clone-*'
func Test_UpdateDefaultBranch(t *testing.T) {
	defer seq()()
	subcommand := "updateDefault"
	mockUI := cli.NewMockUi()
	owner, repo, token, base, target, _, _, _ := i.GetVals()
	args := []string{subcommand, "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	if err != nil {
		t.Errorf("Failed to create config due to error: %v", err)
	}

	client, err := gh.NewBaseGithubInteractor(token)
	if err != nil {
		t.Errorf("Failed to create client due to error: %v", err)
	}

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

// Test_UpdateDefaultBranchToMaster changes the default branch back to master
func Test_UpdateDefaultBranchToMaster(t *testing.T) {
	defer seq()()
	subcommand := "updateDefault"
	mockUI := cli.NewMockUi()
	owner, repo, token, _, base, _, _, _ := i.GetVals()
	target := "master"
	args := []string{subcommand, "--owner", owner, "--repo", repo, "--base", base, "--target", target, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	if err != nil {
		t.Errorf("Failed to create config due to error: %v", err)
	}

	client, err := gh.NewBaseGithubInteractor(token)
	if err != nil {
		t.Errorf("Failed to create client due to error: %v", err)
	}

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
	subcommand := "deleteBranches"
	mockUI := cli.NewMockUi()
	owner, repo, token, base, _, _, _, branchesList := i.GetVals()
	args := []string{subcommand, "--owner", owner, "--repo", repo, "--base", base, "--token", token}

	// Parse and validate cmd line flags and env vars
	config, err := config.ParseAndValidate(args, mockUI)
	if err != nil {
		t.Errorf("Failed to create config due to error: %v", err)
	}

	client, err := gh.NewBaseGithubInteractor(token)
	if err != nil {
		t.Errorf("Failed to create client due to error: %v", err)
	}

	command := &branches.DeleteCommand{
		Config:       config,
		GithubClient: client,
		BranchesList: branchesList,
	}

	exit := command.Run([]string{})

	// Did we exit with a zero exit code?
	if !assert.Equal(t, 0, exit) {
		require.Fail(t, mockUI.ErrorWriter.String())
	}

	// Make some assertions about the UI output
	output := mockUI.OutputWriter.String()
	for _, branch := range branchesList {
		assert.Contains(t, output, fmt.Sprintf("Attempting to remove branch protection from branch: branch=%s", branch))
		assert.Contains(t, output, fmt.Sprintf("Attempting to delete branch: branch=%s", branch))
		assert.Contains(t, output, fmt.Sprintf("Success! branch has been deleted: branch=%s", branch))
	}
}
