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
	Git GithubGitInteractor

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

	return m
}

// GetGit returns an internal mock that represents the GitService Client.
func (m *MockGithubInteractor) GetGit() GithubGitInteractor {
	return m.Git
}

// MockGithubGitInteractor is a mock implementation of the GithubGitInteractor
// interface, which represents the GitService Client.
type MockGithubGitInteractor struct {
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
