# inclusify

Inclusify is a CLI that will rename the default branch of any GitHub repo and perform the other necessary tasks that go along with it. 

```
Usage: inclusify [--version] [--help] <command> [<args>]

Available commands are:
    createBranches    Create new branches on GitHub. [subcommand]
    deleteBase        Delete repo's base branch. [subcommand]
    updateCI          Update all CI *.y{a]ml references. [subcommand]
    updateDefault     Update repo's default branch. [subcommand]
    updatePulls       Update base branch of open PR's. [subcommand]
```

## Usage

Clone the repo
```
git clone git@github.com:hashicorp/inclusify.git ~/go/src/github.com/hashicorp/inclusify
cd ~/go/src/github.com/hashicorp/inclusify
```

Build the binary
```
go build -o inclusify ./cmd/inclusify
```

Pass in the required flags to the subcommands, set env vars, or source a local env file. We recommend setting env vars or sourcing a local env file for ease of use
```
export INCLUSIFY_OWNER="$owner"
export INCLUSIFY_REPO="$repo"
export INCLUSIFY_TOKEN="$github_personal_access_token"
```

To run the following subcommands, the `personal_access_token` will need to be associated with a user with `write` access on the repo.

[Optional] Pass in optional flags or env vars. Defaults are shown below and in --help
```
export INCLUSIFY_BASE="master" // Name of the current default base branch for the repo
export INCLUSIFY_TARGET="main" // Name of the new target base branch for the repo
```

Remember to set your `$GOPATH`
```
export GOPATH=$HOME/go
```

Run the subcommands below in the following order
```
./inclusify createBranches
./inclusify updateCI
```

On success, updateCI will return a PR URL
**Review the PR carefully, make any required changes, and merge it into the $target branch before continuing**

Continue with the below commands to update the base branch of the repo's open PR's and update the repo's default branch
```
./inclusify updatePulls
./inclusify updateDefault
```

After verifying everything is working properly, delete the old base branch. This defaults to `master`. If the `master` branch is protected, you'll get an error. Remove the protection in the UI and re-run the command again. 
```
./inclusify deleteBase
```

## Docker

The Dockerfile is located in `build/package/docker`. Build with the following command:

```
docker build -f build/package/docker/Dockerfile -t inclusify .
docker run inclusify
```