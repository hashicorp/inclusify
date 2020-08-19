package main

import (
	"log"
	"os"

	"github.com/hashicorp/inclusify/pkg/version"
	"github.com/mitchellh/cli"

	branches "github.com/hashicorp/inclusify/pkg/branches"
	config "github.com/hashicorp/inclusify/pkg/config"
	files "github.com/hashicorp/inclusify/pkg/files"
	gh "github.com/hashicorp/inclusify/pkg/gh"
	pulls "github.com/hashicorp/inclusify/pkg/pulls"
)

func main() {
	if err := inner(); err != nil {
		log.Printf("inclusify error: %s\n", err)
		os.Exit(1)
	}
}

func inner() error {
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	c := cli.NewCLI("inclusify", version.Version)
	c.HelpWriter = os.Stdout
	c.ErrorWriter = os.Stderr
	c.Args = os.Args[1:]

	// Parse and validate cmd line flags and env vars
	cf, err := config.ParseAndValidate(c.Args, ui)
	if err != nil {
		return err
	}

	client, err := gh.NewBaseGithubInteractor(cf.Token)
	if err != nil {
		return err
	}

	tmpBranch := "update-ci-references"
	c.Commands = map[string]cli.CommandFactory{
		"createBranches": func() (cli.Command, error) {
			return &branches.CreateCommand{Config: cf, GithubClient: client, BaseBranch: cf.Base, BranchesList: []string{tmpBranch, cf.Target}}, nil
		},
		"updateCI": func() (cli.Command, error) {
			return &files.UpdateCICommand{Config: cf, GithubClient: client, TempBranch: tmpBranch}, nil
		},
		"updatePulls": func() (cli.Command, error) {
			return &pulls.UpdateCommand{Config: cf, GithubClient: client}, nil
		},
		"updateDefault": func() (cli.Command, error) {
			return &branches.UpdateCommand{Config: cf, GithubClient: client}, nil
		},
		"deleteBranches": func() (cli.Command, error) {
			return &branches.DeleteCommand{Config: cf, GithubClient: client, BranchesList: []string{tmpBranch, cf.Base}}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		return err
	}

	os.Exit(exitStatus)
	return nil
}
