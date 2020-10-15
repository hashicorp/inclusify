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

// UpdateRefsCommand is a struct used to configure a Command for updating
// CI references
type UpdateRefsCommand struct {
	Config       *config.Config
	GithubClient gh.GithubInteractor
	TempBranch   string
}

// CloneRepo creates a temp directory and clones the repo at $tmpBranch into it
func CloneRepo(c *UpdateRefsCommand) (repoRef *git.Repository, dir string, err error) {
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
			Username: "irrelevant", // This cannot be an empty string
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

// UpdateReferences walks through the files in the cloned repo, and updates references from
// $base to $target. It excludes any paths from `INCLUSIFY_PATH_EXCLUSION`
func UpdateReferences(c *UpdateRefsCommand, dir string) (filesChanged bool, err error) {
	c.Config.Logger.Info("Finding and replacing all references from base to target in dir", "base", c.Config.Base, "target", c.Config.Target, "dir", dir)
	// Set a flag to false, and update it to true if any files are modified.
	filesChanged = false
	// Walk through the directories/files in the tmp directory, $dir, where the repo was cloned
	callback := func(path string, fi os.FileInfo, err error) error {
		// Skip directories and files that should be excluded
		if len(c.Config.Exclusion) > 0 {
			for _, fp := range c.Config.Exclusion {
				if strings.Contains(path, fp) {
					return nil
				}
			}
		}
		// Find and replace within the repo's files
		if !fi.IsDir() {
			read, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			// Find and replace all references from $base to $target within the files
			var newContents string
			lines := strings.Split(string(read), "\n")
			for _, line := range lines {
				// Don't find/replace lines that contains go imports
				if !isLineContainsGoImport(line) {
					line = strings.ReplaceAll(line, c.Config.Base, c.Config.Target)
				}
				newContents += line + "\n"
			}

			// Set flag to true if the file was modified
			if newContents != string(read) {
				filesChanged = true
			}
			// Update the file with the new contents
			err = ioutil.WriteFile(path, []byte(newContents), 0)
			if err != nil {
				return err
			}
			c.Config.Logger.Info("Updated the file", "path", path)
		}
		return nil
	}

	err = filepath.Walk(dir, callback)
	if err != nil {
		return false, err
	}

	return filesChanged, nil
}

func isLineContainsGoImport(line string) bool {
	return strings.Contains(line, "github.com")
}


// GitPush adds, commits, and pushes all changes to $tmpBranch
func GitPush(c *UpdateRefsCommand, tmpBranch string, repo *git.Repository, dir string) (err error) {
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	c.Config.Logger.Info("Running `git add ", dir, "`")
	_, err = worktree.Add(".")
	if err != nil {
		return fmt.Errorf("failed to `git add %s`: %w", dir, err)
	}

	c.Config.Logger.Info("Committing changes")
	commitMsg := fmt.Sprintf("Update references from %s to %s", c.Config.Base, c.Config.Target)
	commitSha, err := worktree.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			When: time.Now(),
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

// OpenPull opens the pull request to merge the changes from $tmpBranch into $target.
// $tmpBranch is 'update-references', and $target is typically 'main'
func OpenPull(c *UpdateRefsCommand, tmpBranch string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var body string

	c.Config.Logger.Info("Setting up PR request")
	title := fmt.Sprintf("Update References from %s to %s", c.Config.Base, c.Config.Target)
	if len(c.Config.Exclusion) == 0 {
		body = fmt.Sprintf("This PR was created to update all references from '%s' to '%s' in this repo.<br /><br />**NOTE**: This PR was generated automatically. Please take a close look before approving and merging!", c.Config.Base, c.Config.Target)
	} else {
		body = fmt.Sprintf("This PR was created to update all references from '%s' to '%s' in this repo.<br /><br />The following paths have been excluded: '%v'<br /><br />**NOTE**: This PR was generated automatically. Please take a close look before approving and merging!", c.Config.Base, c.Config.Target, c.Config.Exclusion)
	}

	modify := true
	pull := &github.NewPullRequest{
		Title:               &title,
		Head:                &tmpBranch,
		Base:                &c.Config.Target,
		Body:                &body,
		MaintainerCanModify: &modify,
	}

	c.Config.Logger.Info("Creating PR to merge changes from branch into target", "branch", tmpBranch, "target", c.Config.Target)
	pr, _, err := c.GithubClient.GetPRs().Create(ctx, c.Config.Owner, c.Config.Repo, pull)
	if err != nil {
		return fmt.Errorf("failed to open PR: %w", err)
	}
	c.Config.Logger.Info("Success! Review and merge the open PR", "url", pr.GetHTMLURL())

	return nil
}

// Run updates references from $base to $target in the cloned repo
// Example: Update all occurrences of 'master' to 'main' in ./.github
func (c *UpdateRefsCommand) Run(args []string) int {
	repo, dir, err := CloneRepo(c)
	if err != nil {
		return c.exitError(err)
	}

	ref, err := repo.Head()
	if err != nil {
		return c.exitError(fmt.Errorf("failed to retrieve HEAD commit: %w", err))
	}
	c.Config.Logger.Info("Retrieved HEAD commit of branch", "branch", c.TempBranch, "sha", ref.Hash())

	filesChanged, err := UpdateReferences(c, dir)
	if err != nil {
		return c.exitError(err)
	}

	// Exit if no files were modified during the find and replace
	if !filesChanged {
		c.Config.Logger.Info("Exiting -- No CI files contained base, so there's nothing more to do", "base", c.Config.Base)
		return 0
	}

	err = GitPush(c, c.TempBranch, repo, dir)
	if err != nil {
		return c.exitError(err)
	}

	err = OpenPull(c, c.TempBranch)
	if err != nil {
		return c.exitError(err)
	}

	defer os.RemoveAll(dir)

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *UpdateRefsCommand) exitError(err error) int {
	c.Config.Logger.Error(err.Error())
	return 1
}

// Help returns the full help text.
func (c *UpdateRefsCommand) Help() string {
	return `Usage: inclusify updateRefs owner repo base target token
	Update code references from base to target in the given repo. Any dirs/files provided in exclusion will be excluded. Configuration is pulled from the local environment.
	Flags:
	--owner          The GitHub org that owns the repo, e.g. 'hashicorp'.
	--repo           The repository name, e.g. 'circle-codesign'.
	--base="master"  The name of the current base branch, e.g. 'master'.
	--target="main"  The name of the target branch, e.g. 'main'.
	--token          Your Personal GitHub Access Token.
	--exclusion      Paths to exclude from reference updates.
	`
}

// Synopsis returns a sub 50 character summary of the command.
func (c *UpdateRefsCommand) Synopsis() string {
	return "Update code references from base to target in the given repo. [subcommand]"
}
