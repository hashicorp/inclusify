# inclusify [![CI Status](https://circleci.com/gh/hashicorp/inclusify.svg?style=svg&circle-token=0ae7a4e49ff1f990f45536f92c62dab322f13113)](https://circleci.com/gh/hashicorp/inclusify/tree/master)

Inclusify is a CLI that will rename the default branch of any GitHub repo and perform all other necessary tasks that go along with it, such as updating CI references, updating the base branch of open PR's, copying the branch protection, and deleting the old base.

```
Usage: inclusify [--version] [--help] <command> [<args>]

Available commands are:
    createBranches    Create new branches on GitHub. [subcommand]
    deleteBranches    Delete repo's base branch and other auto-created branches. [subcommand]
    updateCI          Update all CI *.y{a]ml references. [subcommand]
    updateDefault     Update repo's default branch. [subcommand]
    updatePulls       Update base branch of open PR's. [subcommand]
```

## Usage

1. Download the latest release for your os/platform from https://github.com/hashicorp/inclusify/releases, and unzip to access the go binary.

2. Create a `.env` file with the following environment variables:

```
export INCLUSIFY_OWNER="$owner" // REQUIRED: Name of the GitHub org or user account where the repo lives
export INCLUSIFY_REPO="$repo" // REQUIRED: Name of the repo
export INCLUSIFY_TOKEN="$github_personal_access_token" // REQUIRED: Your GitHub personal access token that has write access to the repo
export INCLUSIFY_BASE="master" // OPTIONAL: Name of the current default base branch for the repo. This defaults to "master"
export INCLUSIFY_TARGET="main" // OPTIONAL: Name of the new target base branch for the repo. This defaults to "main"
```

**Note:** You can alternatively pass in the required flags to the subcommands or set environment variables locally without sourcing an env file. For ease of use, however, we recommend sourcing a local env file. 

3. Source the file to set the environment variables locally: `source .env` 

4. Run the below commands in the following order:

Set up the new target branch and temporary branches which will be used in the next steps, and create a PR to update all CI references from `base` to `target`. This happens via a simple find and replace within the root files `.goreleaser.y{a}ml`, `.travis.y{a}ml`, and within any files in the root directories `.circleci`, `.teamcity`, `.github`.
```
./inclusify createBranches
./inclusify updateCI 
```

On success, updateCI will return a pull request URL. **Review the PR carefully, make any required changes, and merge it into the `target` branch before continuing.** 

Continue with the below commands to update the base branch of any open PR's from `base` to `target`. Finally, update the repo's default branch from `base` to `target`. If the `base` branch was protected, copy that protection over to `target`. 
```
./inclusify updatePulls
./inclusify updateDefault
```

After verifying everything is working properly, delete the old base branch. If the `base` branch was protected, the protection will be removed automatically, and then the branch will be deleted. This will also delete the `update-ci-references` branch that was created in the first step. 
```
./inclusify deleteBranches
```

5. All done! For local development, remember to fetch and push to the new origin.

## Local Development

1. Clone the repo
```
git clone git@github.com:hashicorp/inclusify.git ~/go/src/github.com/hashicorp/inclusify
cd ~/go/src/github.com/hashicorp/inclusify
```

2. Make any modifications you wish, and build the go binary
```
go build -o inclusify ./cmd/inclusify
```

3. Pass in the required flags to the subcommands, set env vars, or source a local env file, as explained in step #2 above. Remember that the GitHub token chosen will need to be associated with a user with `write` access on the repo.

4. Set your `$GOPATH`
```
export GOPATH=$HOME/go
```

5. Run the subcommands in the correct order, as explained in step #4 above. 

## Testing

To run the unit tests locally, clone the repo and run `go test ./...` or `gotestsum ./...` in the root of the directory. To run the integration tests, set the environment variables as described in step #2 above and add `--tags=integation` to the test command. These tests will create a new repo under the authenticated user, provision the repo, and create/delete real resources within it. Finally, the test repo will be deleted. 

All tests are run in CI on every push, and run against a repo in the format `$user/inclusify-tests-$random`.

## Docker

The Dockerfile is located in `build/package/docker` and is available on the `hashicorpdev` dockerhub account. Build locally with the following command:

```
docker build -f build/package/docker/Dockerfile -t inclusify .
docker run inclusify
```
