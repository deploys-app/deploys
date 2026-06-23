package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/deploys-app/api"
	"github.com/deploys-app/api/client"
	"golang.org/x/oauth2/google"

	"github.com/deploys-app/deploys/internal/auth"
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

	// Pre-scan the global -account selector before the api client is built:
	// per-subcommand flag.FlagSets parse too late to shape client construction.
	// This runs after the bare/help fast-path above (which the empty-args case
	// below also relies on), and extracts only the non-secret account email.
	selector, args := extractAccountFlag(args)
	if selector == "" {
		selector = os.Getenv("DEPLOYS_ACCOUNT")
	}
	explicit := selector != ""
	if len(args) == 0 {
		// -account consumed the only token, e.g. `deploys -account x`.
		runner.PrintUsage(os.Stdout)
		return
	}

	rn := runner.Runner{
		Output:  os.Stdout,
		Version: resolveVersion(),
		Account: selector,
	}

	// Local utility commands (check-update/version) and the auth surface
	// (login/logout/auth) run without the pre-built client: the former are
	// client-less, the latter establish or read credentials themselves.
	if !runner.IsLocalCommand(args[0]) && !runner.IsAuthCommand(args[0]) {
		c, err := newAPIClient(selector, explicit)
		if err != nil {
			fail(err)
		}
		rn.API = c
	}

	if err := rn.Run(args...); err != nil {
		fail(err)
	}
}

// fail prints an error and exits. An authentication-required error (no usable
// credential, or an expired session reported by the api) maps to exit code 4 and
// goes to stderr with a re-login hint; everything else keeps the legacy
// stdout + exit 1 behavior so existing scripts are unaffected.
func fail(err error) {
	if exitCode(err) == 4 {
		var are *auth.AuthRequiredError
		if errors.As(err, &are) {
			fmt.Fprintln(os.Stderr, "Error: "+err.Error())
		} else {
			fmt.Fprintln(os.Stderr, "Error: your deploys session has expired.")
			fmt.Fprintln(os.Stderr, "Run 'deploys login' to sign in again.")
		}
		os.Exit(4)
	}
	fmt.Println(err)
	os.Exit(1)
}

// exitCode classifies an error into a process exit code: 4 for an
// authentication-required/expired error (a missing-or-expired stored login, or
// the api's typed ErrUnauthorized), 1 otherwise. ErrForbidden (a valid token
// lacking a permission) deliberately stays 1 — a permission denial is not a
// reason to re-login.
func exitCode(err error) int {
	var are *auth.AuthRequiredError
	if errors.As(err, &are) || errors.Is(err, api.ErrUnauthorized) {
		return 4
	}
	return 1
}

// extractAccountFlag pulls a global -account/--account selector (in either
// "-account v" or "-account=v" form) out of args and returns the value plus the
// remaining args. It is a hand-rolled scan, not a flag.FlagSet, because the
// selector must be known before the api client is built. It extracts only the
// account email and must never be widened to read a token or any secret.
func extractAccountFlag(args []string) (string, []string) {
	var selector string
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-account" || a == "--account":
			if i+1 < len(args) {
				selector = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "-account="):
			selector = strings.TrimPrefix(a, "-account=")
		case strings.HasPrefix(a, "--account="):
			selector = strings.TrimPrefix(a, "--account=")
		default:
			rest = append(rest, a)
		}
	}
	return selector, rest
}

// newAPIClient builds the api client, resolving credentials in order:
//  1. DEPLOYS_AUTH_USER+DEPLOYS_AUTH_PASS  (service-account basic auth; CI)
//  2. DEPLOYS_TOKEN                         (bearer; CI; empty is treated unset)
//  3. a stored login for the endpoint       (from `deploys login`)
//  4. Google Application Default Credentials (legacy fallback)
//
// Steps 1–2 are unchanged so CI is untouched. An explicitly-selected account
// (-account/DEPLOYS_ACCOUNT) that does not resolve is a hard error, not an ADC
// fallthrough; with nothing configured at all it returns an AuthRequiredError so
// main can print a login hint instead of a bare "unauthorized" round-trip.
func newAPIClient(selector string, explicit bool) (*client.Client, error) {
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

	if authUser != "" && authPass != "" {
		apiClient.Auth = func(r *http.Request) { r.SetBasicAuth(authUser, authPass) }
		return apiClient, nil
	}
	if token != "" {
		apiClient.Auth = bearerAuth(token)
		return apiClient, nil
	}

	acct, expired, err := auth.Resolve(selector, endpoint, explicit)
	if err != nil {
		return nil, err
	}
	if acct != nil && acct.Token != "" {
		if d := time.Until(acct.ExpiresAt); d > 0 && d < 24*time.Hour {
			fmt.Fprintf(os.Stderr, "warning: your deploys session expires in %s — run 'deploys login' to sign in again\n", shortDur(d))
		}
		apiClient.Auth = bearerAuth(acct.Token)
		return apiClient, nil
	}
	if expired && acct != nil {
		fmt.Fprintf(os.Stderr, "warning: stored session for %s expired — run 'deploys login'\n", acct.Email)
	}

	if adc, aerr := getDefaultToken(); aerr == nil && adc != "" {
		apiClient.Auth = bearerAuth(adc)
		return apiClient, nil
	}

	return nil, &auth.AuthRequiredError{Msg: "not logged in. Run 'deploys login' to sign in."}
}

func bearerAuth(token string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) }
}

// shortDur renders a positive duration compactly for the near-expiry warning.
func shortDur(d time.Duration) string {
	h := int(d.Hours())
	switch {
	case h >= 24:
		return fmt.Sprintf("%dd %dh", h/24, h%24)
	case h >= 1:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
}

// resolveVersion reports this binary's version: the release value injected via
// ldflags, else the module version recorded by the Go toolchain for
// `go install ...@version` builds, else "dev".
func resolveVersion() string {
	if version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	// `go install <module>@<version>` records the module version (a tag like
	// v1.1.3, or a v0.0.0-<date>-<sha> pseudo-version for @main / @<commit>).
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	// Built from a source checkout (go build / go install . in a clone): fall
	// back to the VCS revision the toolchain embeds, so the build is still
	// identifiable (e.g. "dev-1a2b3c4" or "dev-1a2b3c4-dirty").
	return vcsVersion(info)
}

// vcsVersion derives a "dev-<shortsha>[-dirty]" string from the build's VCS
// settings, or "dev" when none are present.
func vcsVersion(info *debug.BuildInfo) string {
	var revision string
	var modified bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if revision == "" {
		return "dev"
	}
	if len(revision) > 7 {
		revision = revision[:7]
	}
	v := "dev-" + revision
	if modified {
		v += "-dirty"
	}
	return v
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
