package gh

import (
	"context"

	"github.com/hashicorp/go-hclog"

	github "github.com/google/go-github/v32/github"
)

// TODO: Probably get rid of GitHub
// GH is the API client struct for interacting with GitHub
type GitHub struct {
	Owner  string
	Repo   string
	Base   string
	Target string
	Token  string
	Ctx    context.Context
	Client *github.Client
	Logger hclog.Logger
}

// GithubInteractor is an interface that represents interaction with the GitHub
// API. This can be the real GitHub, or a mock.
type GithubInteractor interface {
	GetGit() GithubGitInteractor
}

// GithubGitInteractor is a more specific interface that represents a GitService
// client in GitHub. This can also be real or fake.
type GithubGitInteractor interface {
	GetRef(ctx context.Context, owner string, repo string, ref string) (*github.Reference, *github.Response, error)
	CreateRef(ctx context.Context, owner string, repo string, ref *github.Reference) (*github.Reference, *github.Response, error)
}

// baseGithubInteractor is a concrete implementation of the GithubInteractor
// interface. In this case, it implements the methods of this interface by
// calling the real GitHub client.
type baseGithubInteractor struct {
	github *github.Client
}

// GetGit returns the GitService Client.
func (b *baseGithubInteractor) GetGit() GithubGitInteractor {
	return b.github.Git
}

// NewBaseGithubInteractor is a constructor for baseGithubInteractor.
func NewBaseGithubInteractor(user, token string) (*baseGithubInteractor, error) {
	return nil, nil
}
