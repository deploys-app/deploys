package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/deploys-app/api"
	"github.com/deploys-app/api/client"
	"golang.org/x/oauth2/google"

	"github.com/deploys-app/deploys/internal/runner"
)

// version is set at release time via -ldflags "-X main.version=...". For other
// builds it stays empty and resolveVersion falls back to the module's build
// info.
var version string

func main() {
	args := os.Args[1:]
	// Fast path for the bare/top-level help invocation: print usage without
	// building the api client (which may resolve Application Default
	// Credentials). Group- and subcommand-level help (`deploys me -h`, etc.) is
	// dispatched through Run below and still pays that lookup, but it returns
	// before making any API call, so the resolved token is simply unused.
	if len(args) == 0 || (len(args) == 1 && runner.IsHelpArg(args[0])) {
		runner.PrintUsage(os.Stdout)
		return
	}

	rn := runner.Runner{
		Output:  os.Stdout,
		Version: resolveVersion(),
	}

	// Local utility commands (check-update) run entirely client-side; skip
	// building the api client so they never resolve Google credentials.
	if !runner.IsLocalCommand(args[0]) {
		rn.API = newAPIClient()
	}

	err := rn.Run(args...)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func newAPIClient() *client.Client {
	var (
		token    = os.Getenv("DEPLOYS_TOKEN")
		authUser = os.Getenv("DEPLOYS_AUTH_USER")
		authPass = os.Getenv("DEPLOYS_AUTH_PASS")
		endpoint = os.Getenv("DEPLOYS_ENDPOINT")
	)

	apiClient := &client.Client{
		Endpoint: endpoint,
		Channel:  api.AuditChannelCLI,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
	if apiClient.Auth == nil && authUser != "" && authPass != "" {
		apiClient.Auth = func(r *http.Request) {
			r.SetBasicAuth(authUser, authPass)
		}
	}

	if apiClient.Auth == nil && token == "" {
		token, _ = getDefaultToken()
	}

	if apiClient.Auth == nil && token != "" {
		apiClient.Auth = func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+token)
		}
	}

	return apiClient
}

// resolveVersion reports this binary's version: the release value injected via
// ldflags, else the module version recorded by the Go toolchain for
// `go install ...@version` builds, else "dev".
func resolveVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}

func getDefaultToken() (string, error) {
	cred, err := google.FindDefaultCredentials(context.Background())
	if err != nil {
		return "", err
	}

	tk, err := cred.TokenSource.Token()
	if err != nil {
		return "", err
	}

	return tk.AccessToken, nil
}
