package files

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/hashicorp/inclusify/pkg/gh"

	git "github.com/go-git/go-git/v5"
	plumbing "github.com/go-git/go-git/v5/plumbing"
	object "github.com/go-git/go-git/v5/plumbing/object"
	http "github.com/go-git/go-git/v5/plumbing/transport/http"
)

// UpdateCICommand is a struct used to configure a Command for updating
// CI references
type UpdateCICommand struct {
	Config *gh.GitHub
}

// CloneRepo creates a temp directory and clones the repo at $tmpBranch into it
func CloneRepo(config *gh.GitHub, tmpBranch string) (repoRef *git.Repository, dir string, err error) {
	// Get current working directory
	pwd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	// Create a local tmp directory to store the clone
	prefix := "tmp-clone-"
	config.Logger.Info("Creating local temp dir", "dirPrefix", prefix)
	dir, err = ioutil.TempDir(pwd, prefix)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create tmp directory: %w", err)
	}

	// Clone the repo into the tmp directory at the HEAD commit of tmpBranch
	url := fmt.Sprintf("https://github.com/%s/%s.git", config.Owner, config.Repo)
	refName := fmt.Sprintf("refs/heads/%s", tmpBranch)
	repo, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL: url,
		Auth: &http.BasicAuth{
			Username: "irrelevant",
			Password: config.Token,
		},
		ReferenceName: plumbing.ReferenceName(refName),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to clone repo: %w", err)
	}

	config.Logger.Info("Successfully cloned $repo at head of $tmpBranch into $dir", "repo", config.Repo, "tmpBranch", tmpBranch, "dir", dir)

	return repo, dir, nil

}

// UpdateCIReferences walks through the CI directories/files in the tmp directory, $dir,
// where the repo was cloned locally.
// It then finds and replaces all references from $base to $target within the *.y{a}ml files.
func UpdateCIReferences(config *gh.GitHub, dir string, paths []string) (filesChanged bool, err error) {
	config.Logger.Info("Finding and replacing all refs from $base to $target in .y{a}ml files in $dir/$paths", "base", config.Base, "target", config.Target, "dir", dir, "paths", paths, "fileSuffix", "*.y{a}ml")
	// Set a flag to false, and update it to true if any files are modified.
	filesChanged = false
	// Walk through the CI directories/files in the tmp directory, $dir, where the repo was cloned
	for _, path := range paths {
		// Check if the path exists
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
					config.Logger.Info("Checking the file at path $path", "path", path)
					read, err := ioutil.ReadFile(path)
					if err != nil {
						return err
					}
					// Find and replace all references from $base to $target within the *.y{a}ml files
					newContents := strings.Replace(string(read), config.Base, config.Target, -1)
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
func GitPush(config *gh.GitHub, tmpBranch string, repo *git.Repository) (err error) {
	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Git add all changes
	config.Logger.Info("Running `git add .`")
	_, err = worktree.Add(".")
	if err != nil {
		return fmt.Errorf("failed to 'git add .': %w", err)
	}

	// Create a new commit
	config.Logger.Info("Committing changes")
	commitMsg := fmt.Sprintf("Update CI references from %s to %s", config.Base, config.Target)
	commitSha, err := worktree.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Inclusive Language",
			Email: "inclusive-language@hashicorp.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// Push changes to $tmpBranch
	config.Logger.Info("Pushing commit to remote branch $tmpBranch", "tmpBranch", tmpBranch, "sha", commitSha)
	err = repo.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: "irrelevant",
			Password: config.Token,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

// OpenPull opens the pull request to the merge CI changes from $tmpBranch into $target.
// $tmpBranch is 'update-ci-references', and $target is typically 'main'
func OpenPull(config *gh.GitHub, tmpBranch string) (err error) {
	// Setup PR request
	config.Logger.Info("Setting up PR request")
	title := fmt.Sprintf("Update CI References from %s to %s", config.Base, config.Target)
	body := fmt.Sprintf("This PR was created to update all references from '%s' to '%s' in every *.y{a}ml file in the CI directories in this repo.<br /><br />**NOTE**: This PR was generated automatically. Please take a close look before approving and merging!", config.Base, config.Target)
	modify := true
	pull := &github.NewPullRequest{
		Title:               &title,
		Head:                &tmpBranch,
		Base:                &config.Target,
		Body:                &body,
		MaintainerCanModify: &modify,
	}

	// Create new PR to merge $tmpBranch into $target brach
	config.Logger.Info("Creating PR to merge CI changes from $tmpBranch into $target", "tmpBranch", tmpBranch, "target", config.Target)
	pr, _, err := config.Client.PullRequests.Create(config.Ctx, config.Owner, config.Repo, pull)
	if err != nil {
		return fmt.Errorf("failed to open PR: %w", err)
	}
	config.Logger.Info("Success! Review and merge the open PR", "url", pr.GetHTMLURL())

	return nil
}

// Run updates CI references from $base to $target in the cloned repo
// Example: Update all occurences of 'master' to 'main' in ./.github
func (c *UpdateCICommand) Run(args []string) int {
	//Locally clone the repo at $tmpBranch into $dir
	tmpBranch := "update-ci-references"
	repo, dir, err := CloneRepo(c.Config, tmpBranch)
	if err != nil {
		return c.exitError(err)
	}

	// Retrieve the HEAD commit
	ref, err := repo.Head()
	if err != nil {
		return c.exitError(fmt.Errorf("failed to retrieve HEAD commit: %w", err))
	}
	c.Config.Logger.Info("Retrieved HEAD commit of $branch", "branch", tmpBranch, "sha", ref.Hash())

	// Update CI references from $base to $target
	paths := []string{".circleci", ".github", ".teamcity", ".travis.yml"}
	filesChanged, err := UpdateCIReferences(c.Config, dir, paths)
	if err != nil {
		return c.exitError(err)
	}
	// Exit if no files were modified during the find and replace
	if !filesChanged {
		c.Config.Logger.Info("Exiting -- No CI files contained $base, so there's nothing more to do", "base", c.Config.Base)
		return 0
	}

	// Remove the dir when finished
	defer os.RemoveAll(dir)

	// Git add, commit, and push changes to $tmpBranch
	err = GitPush(c.Config, tmpBranch, repo)
	if err != nil {
		return c.exitError(err)
	}

	// Open the pull request to merge changes from $tmpBranch into $target
	err = OpenPull(c.Config, tmpBranch)
	if err != nil {
		return c.exitError(err)
	}

	c.Config.Logger.Info("Success!")

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
