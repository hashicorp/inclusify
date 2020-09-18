package files

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dchest/uniuri"
	git "github.com/go-git/go-git/v5"
	plumbing "github.com/go-git/go-git/v5/plumbing"
	object "github.com/go-git/go-git/v5/plumbing/object"
	http "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v32/github"

	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/gh"
)

// UpdateCICommand is a struct used to configure a Command for updating
// CI references
type UpdateCICommand struct {
	Config       *config.Config
	GithubClient gh.GithubInteractor
	TempBranch   string
}

// CloneRepo creates a temp directory and clones the repo at $tmpBranch into it
func CloneRepo(c *UpdateCICommand) (repoRef *git.Repository, dir string, err error) {
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

	url := fmt.Sprintf("https://github.com/%s/%s.git", c.Config.Owner, c.Config.Repo)
	refName := fmt.Sprintf("refs/heads/%s", c.TempBranch)
	repo, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL: url,
		Auth: &http.BasicAuth{
			Username: "",
			Password: c.Config.Token,
		},
		ReferenceName: plumbing.ReferenceName(refName),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to clone repo %s %s %w", url, refName, err)
	}

	c.Config.Logger.Info("Successfully cloned repo into local dir", "repo", c.Config.Repo, "dir", dir)

	return repo, dir, nil

}

// UpdateCIReferences walks through the CI directories/files in the tmp directory, $dir,
// where the repo was cloned locally.
// It then finds and replaces all references from $base to $target within the *.y{a}ml files.
func UpdateCIReferences(c *UpdateCICommand, dir string, paths []string) (filesChanged bool, err error) {
	c.Config.Logger.Info("Finding and replacing all refs from base to target in dir/paths", "base", c.Config.Base, "target", c.Config.Target, "dir", dir, "paths", paths, "fileSuffix", "*.y{a}ml")
	// Set a flag to false, and update it to true if any files are modified.
	filesChanged = false
	// Walk through the CI directories/files in the tmp directory, $dir, where the repo was cloned
	for _, path := range paths {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			if os.IsNotExist(err) {
				continue
			}
		}
		err := filepath.Walk(filepath.Join(dir, path),
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
					c.Config.Logger.Info("Checking the file at path", "path", path)
					read, err := ioutil.ReadFile(path)
					if err != nil {
						return err
					}
					// Find and replace all references from $base to $target within the *.y{a}ml files
					newContents := strings.Replace(string(read), c.Config.Base, c.Config.Target, -1)
					// Set flag to true if the file was modified
					if newContents != string(read) {
						filesChanged = true
					}
					// Update the file with the new contents
					err = ioutil.WriteFile(path, []byte(newContents), 0)
					if err != nil {
						return err
					}
				}
				return nil
			})
		if err != nil {
			return filesChanged, fmt.Errorf("failed to update all *.y{a}ml files: %w", err)
		}
	}
	return filesChanged, nil
}

// GitPush adds, commits, and pushes all CI changes to $tmpBranch
func GitPush(c *UpdateCICommand, tmpBranch string, repo *git.Repository) (err error) {
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
	commitMsg := fmt.Sprintf("Update CI references from %s to %s", c.Config.Base, c.Config.Target)
	email := fmt.Sprintf("inclusive-language@%s.com", c.Config.Owner)
	commitSha, err := worktree.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Inclusive Language",
			Email: email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	c.Config.Logger.Info("Pushing commit to remote", "branch", tmpBranch, "sha", commitSha)
	err = repo.Push(&git.PushOptions{
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

// OpenPull opens the pull request to the merge CI changes from $tmpBranch into $target.
// $tmpBranch is 'update-ci-references', and $target is typically 'main'
func OpenPull(c *UpdateCICommand, tmpBranch string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c.Config.Logger.Info("Setting up PR request")
	title := fmt.Sprintf("Update CI References from %s to %s", c.Config.Base, c.Config.Target)
	body := fmt.Sprintf("This PR was created to update all references from '%s' to '%s' in every *.y{a}ml file in the CI directories in this repo.<br /><br />**NOTE**: This PR was generated automatically. Please take a close look before approving and merging!", c.Config.Base, c.Config.Target)
	modify := true
	pull := &github.NewPullRequest{
		Title:               &title,
		Head:                &tmpBranch,
		Base:                &c.Config.Target,
		Body:                &body,
		MaintainerCanModify: &modify,
	}

	c.Config.Logger.Info("Creating PR to merge CI changes from branch into target", "branch", tmpBranch, "target", c.Config.Target)
	pr, _, err := c.GithubClient.GetPRs().Create(ctx, c.Config.Owner, c.Config.Repo, pull)
	if err != nil {
		return fmt.Errorf("failed to open PR: %w", err)
	}
	c.Config.Logger.Info("Success! Review and merge the open PR", "url", pr.GetHTMLURL())

	return nil
}

// Run updates CI references from $base to $target in the cloned repo
// Example: Update all occurences of 'master' to 'main' in ./.github
func (c *UpdateCICommand) Run(args []string) int {
	repo, dir, err := CloneRepo(c)
	if err != nil {
		return c.exitError(err)
	}

	ref, err := repo.Head()
	if err != nil {
		return c.exitError(fmt.Errorf("failed to retrieve HEAD commit: %w", err))
	}
	c.Config.Logger.Info("Retrieved HEAD commit of branch", "branch", c.TempBranch, "sha", ref.Hash())

	paths := []string{".circleci", ".github", ".teamcity", ".travis.yml"}
	filesChanged, err := UpdateCIReferences(c, dir, paths)
	if err != nil {
		return c.exitError(err)
	}

	// Exit if no files were modified during the find and replace
	if !filesChanged {
		c.Config.Logger.Info("Exiting -- No CI files contained base, so there's nothing more to do", "base", c.Config.Base)
		return 0
	}

	defer os.RemoveAll(dir)

	err = GitPush(c, c.TempBranch, repo)
	if err != nil {
		return c.exitError(err)
	}

	err = OpenPull(c, c.TempBranch)
	if err != nil {
		return c.exitError(err)
	}

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *UpdateCICommand) exitError(err error) int {
	c.Config.Logger.Error(err.Error())
	return 1
}

// Help returns the full help text.
func (c *UpdateCICommand) Help() string {
	return `Usage: inclusify updateCI owner repo base target token
	Update all CI *.y{a]ml references. Configuration is pulled from the local environment.
	Flags:
	--owner          The GitHub org that owns the repo, e.g. 'hashicorp'.
	--repo           The repository name, e.g. 'circle-codesign'.
	--base="master"  The name of the current base branch, e.g. 'master'.
	--target="main"  The name of the target branch, e.g. 'main'.
	--token          Your Personal GitHub Access Token.
	`
}

// Synopsis returns a sub 50 character summary of the command.
func (c *UpdateCICommand) Synopsis() string {
	return "Update all CI *.y{a]ml references. [subcommand]"
}
