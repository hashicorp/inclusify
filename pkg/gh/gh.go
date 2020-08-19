package gh

import (
	"context"
	"errors"

	github "github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

// GithubInteractor is an interface that represents interaction with the GitHub
// API. This can be the real GitHub, or a mock.
type GithubInteractor interface {
	GetGit() GithubGitInteractor
	GetRepo() GithubRepoInteractor
	GetPRs() GithubPRInteractor
}

// GithubGitInteractor is a more specific interface that represents a GitService
// client in GitHub. This can also be real or fake.
type GithubGitInteractor interface {
	GetRef(ctx context.Context, owner string, repo string, ref string) (*github.Reference, *github.Response, error)
	CreateRef(ctx context.Context, owner string, repo string, ref *github.Reference) (*github.Reference, *github.Response, error)
	DeleteRef(ctx context.Context, owner string, repo string, ref string) (*github.Response, error)
}

// GithubPRInteractor is a more specific interface that represents a PullsRequestService
// in GitHub. This can also be real or fake.
type GithubPRInteractor interface {
	Edit(ctx context.Context, owner string, repo string, number int, pull *github.PullRequest) (*github.PullRequest, *github.Response, error)
	List(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	Create(ctx context.Context, owner string, repo string, pull *github.NewPullRequest) (*github.PullRequest, *github.Response, error)
	Merge(ctx context.Context, owner string, repo string, number int, commitMessage string, options *github.PullRequestOptions) (*github.PullRequestMergeResult, *github.Response, error)
}

// GithubRepoInteractor is a more specific interface that represents a RepositoriesService
// in GitHub. This can also be real or fake.
type GithubRepoInteractor interface {
	Edit(ctx context.Context, owner string, repo string, repository *github.Repository) (*github.Repository, *github.Response, error)
	RemoveBranchProtection(ctx context.Context, owner string, repo string, branch string) (*github.Response, error)
	GetBranchProtection(ctx context.Context, owner string, repo string, branch string) (*github.Protection, *github.Response, error)
	UpdateBranchProtection(ctx context.Context, owner string, repo string, branch string, preq *github.ProtectionRequest) (*github.Protection, *github.Response, error)
}

// baseGithubInteractor is a concrete implementation of the GithubInteractor
// interface. In this case, it implements the methods of this interface by
// calling the real GitHub client.
type baseGithubInteractor struct {
	github *github.Client
	repo   *github.RepositoriesService
	pr     *github.PullRequestsService
}

// GetGit returns the GitService Client.
func (b *baseGithubInteractor) GetGit() GithubGitInteractor {
	return b.github.Git
}

// GetRepo returns the RepositioriesService Client.
func (b *baseGithubInteractor) GetRepo() GithubRepoInteractor {
	return b.github.Repositories
}

// GetRepo returns the PullsRequestService Client.
func (b *baseGithubInteractor) GetPRs() GithubPRInteractor {
	return b.github.PullRequests
}

// NewBaseGithubInteractor is a constructor for baseGithubInteractor.
func NewBaseGithubInteractor(token string) (*baseGithubInteractor, error) {
	if token == "" {
		return nil, errors.New("cannot create GitHub Client with empty token")
	}

	ctx := context.Background()
	oauthToken := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	oathClient := oauth2.NewClient(ctx, oauthToken)
	client := github.NewClient(oathClient)

	return &baseGithubInteractor{
		github: client,
		repo:   client.Repositories,
		pr:     client.PullRequests,
	}, nil
}
