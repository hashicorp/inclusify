package files

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/dchest/uniuri"
	"github.com/google/go-github/v32/github"
	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/gh"
	"github.com/otiai10/copy"

	git "github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	plumbing "github.com/go-git/go-git/v5/plumbing"
	object "github.com/go-git/go-git/v5/plumbing/object"
	http "github.com/go-git/go-git/v5/plumbing/transport/http"
)

// CreateScaffoldCommand is a struct used to configure a Command for creating
// an initial commit and pushing it to the base branch of a given repo
type CreateScaffoldCommand struct {
	Config       *config.Config
	GithubClient gh.GithubInteractor
	TempBranch   string
}

// InitializeRepo creates a temp directory and initializes it as a new repo
func InitializeRepo(c *CreateScaffoldCommand) (repoRef *git.Repository, dir string, err error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	prefix := fmt.Sprintf("tmp-clone-%s", uniuri.NewLen(6))
	c.Config.Logger.Info("Creating local temp dir", "dirPrefix", prefix)
	dir, err = ioutil.TempDir(pwd, prefix)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create tmp directory: %w", err)
	}

	c.Config.Logger.Info("Initializing new repo at dir", "dir", dir)
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		return nil, "", fmt.Errorf("failed to initialize new repo: %w", err)
	}

	c.Config.Logger.Info("Creating a new remote for base", "base", c.Config.Base)
	url := fmt.Sprintf("https://github.com/%s/%s.git", c.Config.Owner, c.Config.Repo)
	_, err = repo.CreateRemote((&gitConfig.RemoteConfig{
		Name: "master",
		URLs: []string{url},
	}))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create remote: %w", err)
	}

	return repo, dir, nil
}

// CopyCIFiles copies the files from tests/fakeCi/* into the temp-dir
func CopyCIFiles(c *CreateScaffoldCommand, dir string) (err error) {
	c.Config.Logger.Info("Copying test CI files into temp directory")
	path, err := os.Getwd()
	if err != nil {
		return err
	}
	parent := filepath.Dir(path)
	fakeCIPath := filepath.Join(parent, "tests", "fakeCI")
	if _, err := os.Stat(fakeCIPath); err != nil {
		if os.IsNotExist(err) {
			return err
		}
	}
	err = copy.Copy(fakeCIPath, dir)
	if err != nil {
		return err
	}

	return nil
}

// GitPushCommit adds, commits, and pushes the fakeCI files to the base branch
func GitPushCommit(c *CreateScaffoldCommand, repo *git.Repository) (err error) {
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	c.Config.Logger.Info("Running `git add .`")
	_, err = worktree.Add(".")
	if err != nil {
		return fmt.Errorf("failed to 'git add .': %w", err)
	}

	c.Config.Logger.Info("Committing changes")
	commitMsg := "Creating initial commit"
	commitSha, err := worktree.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			When: time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// Create a new refspec in order to push to $base for the first time
	ref := fmt.Sprintf("refs/heads/%s", c.Config.Base)
	upstreamReference := plumbing.ReferenceName(ref)
	downstreamReference := plumbing.ReferenceName(ref)
	referenceList := append([]gitConfig.RefSpec{},
		gitConfig.RefSpec(upstreamReference+":"+downstreamReference))

	c.Config.Logger.Info("Pushing initial commit to remote", "branch", c.Config.Base, "sha", commitSha)
	err = repo.Push(&git.PushOptions{
		RemoteName: "master",
		RefSpecs:   referenceList,
		Auth: &http.BasicAuth{
			Username: "irrelevant",
			Password: c.Config.Token,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

// CreateBranchProtection creates a branch protection for the base branch
func CreateBranchProtection(c *CreateScaffoldCommand) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c.Config.Logger.Info("Creating branch protection request", "branch", c.Config.Base)
	strict := true
	protectionRequest := &github.ProtectionRequest{
		RequiredStatusChecks: &github.RequiredStatusChecks{
			Strict: false, Contexts: []string{},
		},
		RequiredPullRequestReviews: &github.PullRequestReviewsEnforcementRequest{
			DismissStaleReviews: strict, RequireCodeOwnerReviews: strict, RequiredApprovingReviewCount: 3,
		},
		EnforceAdmins:        strict,
		RequireLinearHistory: &strict,
		AllowForcePushes:     &strict,
		AllowDeletions:       &strict,
	}

	c.Config.Logger.Info("Applying branch protection", "branch", c.Config.Base)
	_, _, err = c.GithubClient.GetRepo().UpdateBranchProtection(ctx, c.Config.Owner, c.Config.Repo, c.Config.Base, protectionRequest)
	if err != nil {
		return fmt.Errorf("failed to create the base branch protection: %w", err)
	}

	return nil
}

// Run creates an initial commit in the $base of the repo, e.g. 'master'
func (c *CreateScaffoldCommand) Run(args []string) int {
	repo, dir, err := InitializeRepo(c)
	if err != nil {
		return c.exitError(err)
	}

	err = CopyCIFiles(c, dir)
	if err != nil {
		return c.exitError(err)
	}

	err = GitPushCommit(c, repo)
	if err != nil {
		return c.exitError(err)
	}

	err = CreateBranchProtection(c)
	if err != nil {
		return c.exitError(err)
	}

	defer os.RemoveAll(dir)

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *CreateScaffoldCommand) exitError(err error) int {
	c.Config.Logger.Error(err.Error())
	return 1
}
