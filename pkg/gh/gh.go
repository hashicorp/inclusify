package gh

import (
	"context"

	github "github.com/google/go-github/v32/github"
)

// GH is the API client struct for interacting with GitHub
type GitHub struct {
	Owner  string
	Repo   string
	Base   string
	Target string
	Token  string
	Ctx    context.Context
	Client *github.Client
}
