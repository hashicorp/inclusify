package main

import (
	"log"
	"os"

	"github.com/mitchellh/cli"

	"github.com/hashicorp/inclusify/pkg/branches"
	"github.com/hashicorp/inclusify/pkg/config"
	"github.com/hashicorp/inclusify/pkg/files"
	"github.com/hashicorp/inclusify/pkg/gh"
	"github.com/hashicorp/inclusify/pkg/pulls"
	"github.com/hashicorp/inclusify/pkg/message"
	"github.com/hashicorp/inclusify/pkg/version"
)

func main() {
	if err := inner(); err != nil {
		log.Printf("%s: %s\n", message.Error("inclusify error"), err)
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
	client := &gh.BaseGithubInteractor{}

	// Parse and validate cmd line flags and env vars
	cf, err := config.ParseAndValidate(c.Args, ui)
	if err != nil {
		return err
	}

	if cf != nil {
		client, err = gh.NewBaseGithubInteractor(cf.Token)
		if err != nil {
			return err
		}
	}

	tmpBranch := "update-references"
	c.Commands = map[string]cli.CommandFactory{
		"createBranches": func() (cli.Command, error) {
			return &branches.CreateCommand{Config: cf, GithubClient: client, BranchesList: []string{tmpBranch}}, nil
		},
		"updateRefs": func() (cli.Command, error) {
			return &files.UpdateRefsCommand{Config: cf, GithubClient: client, TempBranch: tmpBranch}, nil
		},
		"updatePulls": func() (cli.Command, error) {
			return &pulls.UpdateCommand{Config: cf, GithubClient: client}, nil
		},
		"updateDefault": func() (cli.Command, error) {
			return &branches.UpdateCommand{Config: cf, GithubClient: client}, nil
		},
		"deleteBranches": func() (cli.Command, error) {
			return &branches.DeleteCommand{Config: cf, GithubClient: client, BranchesList: []string{tmpBranch}}, nil
		},
	}

	_, err = c.Run()
	if err != nil {
		return err
	}

	return nil
}
