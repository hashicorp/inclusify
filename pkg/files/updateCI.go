package files

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	plumbing "github.com/go-git/go-git/v5/plumbing"
	object "github.com/go-git/go-git/v5/plumbing/object"
	http "github.com/go-git/go-git/v5/plumbing/transport/http"
	log "github.com/sirupsen/logrus"

	"github.com/google/go-github/v32/github"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/inclusify/internal/inputs"
	"github.com/hashicorp/inclusify/pkg/gh"
	"github.com/mitchellh/cli"
)

// UpdateCICommand is a struct used to configure a Command for updating
// CI references
type UpdateCICommand struct {
	UI cli.Ui
}

// CloneRepo creates a temp directory and clones the repo at $tmpBranch into it
func CloneRepo(config *gh.GitHub, tmpBranch string) (repoRef *git.Repository, dir string, err error) {
	// Create a local tmp directory to store the clone
	prefix := "tmp-clone-"
	log.WithFields(log.Fields{
		"dirPrefix": prefix,
	}).Info("Creating local temp dir")
	dir, err = ioutil.TempDir("/Users/mdegges/go/src/github.com/hashicorp/inclusify", prefix)
	if err != nil {
		return nil, "", errwrap.Wrapf("Failed to create tmp directory: {{err}}", err)
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
		return nil, "", errwrap.Wrapf("Failed to clone repo: {{err}}", err)
	}

	log.WithFields(log.Fields{
		"repo":      config.Repo,
		"tmpBranch": tmpBranch,
		"dir":       dir,
	}).Info("Successfully cloned $repo at head of $tmpBranch into $dir")

	return repo, dir, nil

}

// UpdateCIReferences walks through the CI directories/files in the tmp directory, $dir,
// where the repo was cloned locally.
// It then finds and replaces all references from $base to $target within the *.y{a}ml files.
func UpdateCIReferences(config *gh.GitHub, dir string, paths []string) (err error) {
	log.WithFields(log.Fields{
		"base":       config.Base,
		"target":     config.Target,
		"dir":        dir,
		"paths":      paths,
		"fileSuffix": "*.y{a}ml",
	}).Info("Recursively finding and replacing all references from $base to $target in .y{a}ml files in $dir/$paths")

	// Walk through the CI directories/files in the tmp directory, $dir, where the repo was cloned
	for _, path := range paths {
		// Check if the path exists
		if _, err := os.Stat(dir + "/" + path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
		}
		err := filepath.Walk(dir+"/"+path,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
					log.WithFields(log.Fields{
						"path": path,
					}).Info("Checking the file at $path")
					read, err := ioutil.ReadFile(path)
					if err != nil {
						return err
					}
					// Find and replace all references from $base to $target within the *.y{a}ml files
					newContents := strings.Replace(string(read), config.Base, config.Target, -1)
					// Update the file with the new contents
					err = ioutil.WriteFile(path, []byte(newContents), 0)
					if err != nil {
						return err
					}
				}
				return nil
			})
		if err != nil {
			return errwrap.Wrapf("Failed to update all *.y{a}ml files: {{err}}", err)
		}
	}
	return nil
}

// GitPush adds, commits, and pushes all CI changes to $tmpBranch
func GitPush(config *gh.GitHub, tmpBranch string, repo *git.Repository) (err error) {
	// Get the worktree
	w, err := repo.Worktree()
	if err != nil {
		return errwrap.Wrapf("Failed to get worktree: {{err}}", err)
	}

	// Git add all changes
	log.Info("Running `git add .`")
	_, err = w.Add(".")
	if err != nil {
		return errwrap.Wrapf("Failed to 'git add .': {{err}}", err)
	}

	// Create a new commit
	log.Info("Committing changes")
	commitMsg := fmt.Sprintf("Update CI references from %s to %s", config.Base, config.Target)
	commitSha, err := w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Inclusive Language",
			Email: "inclusive-language@hashicorp.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return errwrap.Wrapf("Failed to commit changes: {{err}}", err)
	}

	// Push changes to $tmpBranch
	log.WithFields(log.Fields{
		"tmpBranch": tmpBranch,
		"sha":       commitSha,
	}).Info("Pushing commit to remote branch $tmpBranch")
	err = repo.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: "irrelevant",
			Password: config.Token,
		},
	})
	if err != nil {
		return errwrap.Wrapf("Failed to push changes: {{err}}", err)
	}

	return nil
}

// OpenPull opens the pull request to the merge CI changes from $tmpBranch into $target.
// $tmpBranch is 'update-ci-references', and $target is typically 'main'
func OpenPull(config *gh.GitHub, tmpBranch string) (err error) {
	// Setup PR
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
	pr, _, err := config.Client.PullRequests.Create(config.Ctx, config.Owner, config.Repo, pull)
	if err != nil {
		return errwrap.Wrapf("Failed to open PR: {{err}}", err)
	}
	log.WithFields(log.Fields{
		"tmpBranch": tmpBranch,
		"target":    config.Target,
		"prURL":     pr.GetHTMLURL(),
	}).Info("Created PR to merge CI changes from $tmpBranch into the $target branch")

	return nil
}

// Run updates CI references from $base to $target in the cloned repo
// Example: Update all occurences of 'master' to 'main' in ./.github
func (c *UpdateCICommand) Run(args []string) int {
	// Validate inputs
	config, err := inputs.Validate(args)
	if err != nil {
		return c.exitError(err)
	}

	//Locally clone the repo at $tmpBranch into $dir
	tmpBranch := "update-ci-references"
	repo, dir, err := CloneRepo(config, tmpBranch)
	if err != nil {
		return c.exitError(err)
	}

	// Retrieve the HEAD commit
	ref, err := repo.Head()
	if err != nil {
		return c.exitError(errwrap.Wrapf("Failed to retrieve HEAD commit: {{err}}", err))
	}
	log.WithFields(log.Fields{
		"branch": tmpBranch,
		"sha":    ref.Hash(),
	}).Info("Retrieved HEAD commit of $branch")

	// Update CI references from $base to $target
	paths := []string{".circleci", ".github", ".teamcity", ".travis.yml"}
	err = UpdateCIReferences(config, dir, paths)
	if err != nil {
		return c.exitError(err)
	}

	// Remove the dir when finished
	defer os.RemoveAll(dir)

	// Git add, commit, and push changes to $tmpBranch
	err = GitPush(config, tmpBranch, repo)
	if err != nil {
		return c.exitError(err)
	}

	// Open the pull request to merge changes from $tmpBranch into $target
	err = OpenPull(config, tmpBranch)
	if err != nil {
		return c.exitError(err)
	}

	log.Info("Success!")

	return 0
}

// exitError prints the error to the configured UI Error channel (usually stderr) then
// returns the exit code.
func (c *UpdateCICommand) exitError(err error) int {
	c.UI.Error(err.Error())
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
