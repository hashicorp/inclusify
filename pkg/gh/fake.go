package gh

import (
	"context"
	"errors"
	"fmt"

	github "github.com/google/go-github/v32/github"
)

const (
	masterRef = "e5615943864ba6af8a4cec905aceeeb94b2a2ad6"
)

// MockGithubInteractor is a mock implementation of the GithubInteractor
// interface. It makes basic checks about the validity of the inputs and records
// all the References it creates.
type MockGithubInteractor struct {
	Git  GithubGitInteractor
	Repo GithubRepoInteractor
	PRs  GithubPRInteractor

	MasterRef string

	CreatedReferences []*github.Reference
}

// NewMockGithubInteractor is a constructor for MockGithubInteractor. It sets
// the struct up with everything it needs to mock different types of calls.
func NewMockGithubInteractor() *MockGithubInteractor {
	m := &MockGithubInteractor{
		MasterRef: masterRef,
	}

	m.Git = &MockGithubGitInteractor{parent: m}
	m.Repo = &MockGithubRepoInteractor{parent: m}
	m.PRs = &MockGithubPRsInteractor{parent: m}

	return m
}

// GetGit returns an internal mock that represents the GitService Client.
func (m *MockGithubInteractor) GetGit() GithubGitInteractor {
	return m.Git
}

// GetRepo returns an internal mock that represents the Repository Service.
func (m *MockGithubInteractor) GetRepo() GithubRepoInteractor {
	return m.Repo
}

// GetPRs returns an internal mock that represents the Pull Request Service.
func (m *MockGithubInteractor) GetPRs() GithubPRInteractor {
	return m.PRs
}

// MockGithubGitInteractor is a mock implementation of the GithubGitInteractor
// interface, which represents the GitService Client.
type MockGithubGitInteractor struct {
	parent *MockGithubInteractor
}

// MockGithubRepoInteractor is a mock...
type MockGithubRepoInteractor struct {
	parent *MockGithubInteractor
}

// MockGithubPRsInteractor is a mock...
type MockGithubPRsInteractor struct {
	parent *MockGithubInteractor
}

// GetRef validates it is called for hashicorp/test@master, then returns a
// hardcoded SHA.
func (m *MockGithubGitInteractor) GetRef(
	ctx context.Context, owner string, repo string, ref string,
) (*github.Reference, *github.Response, error) {
	// Validate this request was for hashicorp/test
	if owner != "hashicorp" && repo != "test" {
		return nil, nil, errors.New("must be called for hashicorp/test")
	}

	// We can only return the ref for master
	if ref != "refs/heads/master" {
		return nil, nil, fmt.Errorf("must be called for refs/heads/master but got %s", ref)
	}

	// Let's start simple, always return a nicely formatted Ref, and no error:
	return &github.Reference{
		Ref: github.String(ref),
		Object: &github.GitObject{
			SHA: github.String(m.parent.MasterRef),
		},
	}, nil, nil
}

// CreateRef checks it was called for hashicorp/test, then records the requested
// Reference.
func (m *MockGithubGitInteractor) CreateRef(
	ctx context.Context, owner string, repo string, ref *github.Reference,
) (*github.Reference, *github.Response, error) {
	// Validate this request was for hashicorp/test
	if owner != "hashicorp" && repo != "test" {
		return nil, nil, errors.New("must be called for hashicorp/test")
	}

	m.parent.CreatedReferences = append(
		m.parent.CreatedReferences, ref,
	)

	return nil, nil, nil
}

// DeleteRef ..................
func (m *MockGithubGitInteractor) DeleteRef(
	ctx context.Context, owner string, repo string, ref string) (*github.Response, error) {
	return nil, nil
}

// Create .............................
func (m *MockGithubRepoInteractor) Create(
	ctx context.Context, owner string, repository *github.Repository,
) (*github.Repository, *github.Response, error) {
	return nil, nil, nil
}

// Delete .............................
func (m *MockGithubRepoInteractor) Delete(
	ctx context.Context, owner string, repo string,
) (*github.Response, error) {
	return nil, nil
}

// Edit .............................
func (m *MockGithubRepoInteractor) Edit(
	ctx context.Context, owner string, repo string, repository *github.Repository,
) (*github.Repository, *github.Response, error) {
	return nil, nil, nil
}

// RemoveBranchProtection .............................
func (m *MockGithubRepoInteractor) RemoveBranchProtection(
	ctx context.Context, owner string, repo string, branch string,
) (*github.Response, error) {
	return nil, nil
}

// GetBranchProtection .............................
func (m *MockGithubRepoInteractor) GetBranchProtection(
	ctx context.Context, owner string, repo string, branch string,
) (*github.Protection, *github.Response, error) {
	return nil, nil, nil
}

// UpdateBranchProtection .............................
func (m *MockGithubRepoInteractor) UpdateBranchProtection(
	ctx context.Context, owner string, repo string, branch string, preq *github.ProtectionRequest,
) (*github.Protection, *github.Response, error) {
	return nil, nil, nil
}

// PR stuff

// Edit .............................
func (m *MockGithubPRsInteractor) Edit(
	ctx context.Context, owner string, repo string, number int, pull *github.PullRequest,
) (*github.PullRequest, *github.Response, error) {
	return nil, nil, nil
}

// List .............................
func (m *MockGithubPRsInteractor) List(
	ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions,
) ([]*github.PullRequest, *github.Response, error) {
	return nil, nil, nil
}

// Create .............................
func (m *MockGithubPRsInteractor) Create(
	ctx context.Context, owner string, repo string, pull *github.NewPullRequest,
) (*github.PullRequest, *github.Response, error) {
	return nil, nil, nil
}

// Merge .............................
func (m *MockGithubPRsInteractor) Merge(
	ctx context.Context, owner string, repo string, number int, commitMessage string, options *github.PullRequestOptions) (*github.PullRequestMergeResult, *github.Response, error) {
	return nil, nil, nil
}
