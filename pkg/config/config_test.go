package config

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/go-github/v32/github"
	"github.com/mitchellh/cli"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// Test that the config is generated properly when only env vars are set
func Test_ParseAndValidate_EnvVars(t *testing.T) {
	args := []string{"subcommand"}
	os.Setenv("INCLUSIFY_OWNER", "owner")
	os.Setenv("INCLUSIFY_REPO", "repo")
	os.Setenv("TEST_INCLUSIFY_BASE", "base")
	os.Setenv("TEST_INCLUSIFY_TARGET", "target")
	os.Setenv("INCLUSIFY_TOKEN", "token")

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("INCLUSIFY_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client := github.NewClient(tc)
		resp, err := json.Marshal(client)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
		_, err = w.Write(resp)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
	}))
	defer server.Close()

	ui := &cli.BasicUi{}
	_, err := ParseAndValidate(args, ui)
	require.NoError(t, err)

}

// Test that the config is generated properly when cmd line flags are passed in
func Test_ParseAndValidate_Flags(t *testing.T) {
	os.Unsetenv("INCLUSIFY_OWNER")
	os.Unsetenv("INCLUSIFY_REPO")
	os.Unsetenv("INCLUSIFY_TOKEN")
	os.Unsetenv("TEST_INCLUSIFY_BASE")
	os.Unsetenv("TEST_INCLUSIFY_TARGET")
	args := []string{"subcommand", "--owner", "hashicorp", "--repo", "inclusify", "--token", "github_token"}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "token"},
	)
	tc := oauth2.NewClient(ctx, ts)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client := github.NewClient(tc)
		resp, err := json.Marshal(client)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
		_, err = w.Write(resp)
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
	}))
	defer server.Close()

	ui := &cli.BasicUi{}
	_, err := ParseAndValidate(args, ui)
	require.NoError(t, err)

}
