package runner

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// githubLatestURL returns the latest *stable* release of the cli. The rolling
// "nightly" prerelease is excluded by the releases/latest endpoint, so this is
// always a tagged release like "v1.2.3". It is a package var so tests can point
// it at an httptest server.
var githubLatestURL = "https://api.github.com/repos/deploys-app/deploys/releases/latest"

// IsLocalCommand reports whether name is a command that runs entirely locally
// and needs no API client, so main can skip building one (and resolving Google
// credentials) for it.
func IsLocalCommand(name string) bool {
	switch name {
	case "check-update", "version":
		return true
	}
	return false
}

// IsAuthCommand reports whether name is the login/logout/auth surface. These do
// not use the pre-built rn.API (login establishes credentials; the auth
// management subcommands read the store directly), so main must not build the
// API client for them — building it could short-circuit with a "not logged in"
// error before the user can even log in. Unlike IsLocalCommand, these may hit
// the network (login/logout), so they are a separate predicate.
func IsAuthCommand(name string) bool {
	switch name {
	case "login", "logout", "auth":
		return true
	}
	return false
}

// versionInfo is the structured form of `deploys version` (for -ojson/-oyaml).
type versionInfo struct {
	Version string `json:"version" yaml:"version"`
}

func (v versionInfo) Table() [][]string {
	return [][]string{{v.Version}}
}

// version prints this binary's version. The default view is the bare version
// string; -ojson/-oyaml wrap it as a {version: ...} object for scripting.
func (rn Runner) version(args ...string) error {
	if len(args) > 0 && IsHelpArg(args[0]) {
		writeVersionUsage(rn.output())
		return nil
	}

	f := flag.NewFlagSet("deploys version", flag.ExitOnError)
	f.SetOutput(rn.output())
	rn.registerFlags(f)
	f.Usage = func() { writeVersionUsage(rn.output()) }
	if err := f.Parse(args); err != nil {
		return err
	}

	v := displayVersion(rn.Version)
	if rn.OutputMode == "" || rn.OutputMode == "table" {
		fmt.Fprintln(rn.output(), v)
		return nil
	}
	return rn.print(versionInfo{Version: v})
}

func writeVersionUsage(w io.Writer) {
	fmt.Fprint(w, "version — print the deploys cli version\n\n")
	fmt.Fprint(w, "Usage:\n  deploys version [-output table|yaml|json]\n")
}

// updateCheck is the result of `deploys check-update`.
type updateCheck struct {
	Current         string `json:"current" yaml:"current"`
	Latest          string `json:"latest" yaml:"latest"`
	UpdateAvailable bool   `json:"updateAvailable" yaml:"updateAvailable"`
}

func (u updateCheck) Table() [][]string {
	return [][]string{
		{"CURRENT", "LATEST", "UPDATE AVAILABLE"},
		{u.Current, u.Latest, strconv.FormatBool(u.UpdateAvailable)},
	}
}

// checkUpdate compares this binary's version against the latest stable release
// and reports whether a newer one is available.
func (rn Runner) checkUpdate(args ...string) error {
	if len(args) > 0 && IsHelpArg(args[0]) {
		writeCheckUpdateUsage(rn.output())
		return nil
	}

	f := flag.NewFlagSet("deploys check-update", flag.ExitOnError)
	f.SetOutput(rn.output())
	rn.registerFlags(f)
	f.Usage = func() { writeCheckUpdateUsage(rn.output()) }
	if err := f.Parse(args); err != nil {
		return err
	}

	latest, err := fetchLatestVersion(context.Background(), githubLatestURL)
	if err != nil {
		return fmt.Errorf("check-update: %w", err)
	}

	res := updateCheck{
		Current:         displayVersion(rn.Version),
		Latest:          displayVersion(latest),
		UpdateAvailable: updateAvailable(rn.Version, latest),
	}
	if err := rn.print(res); err != nil {
		return err
	}

	// In the default human view, follow up with how to upgrade.
	if res.UpdateAvailable && (rn.OutputMode == "" || rn.OutputMode == "table") {
		fmt.Fprintf(rn.output(),
			"\nA newer release (%s) is available. Upgrade with:\n"+
				"  go install github.com/deploys-app/deploys@latest\n"+
				"or download it from https://github.com/deploys-app/deploys/releases\n",
			res.Latest)
	}
	return nil
}

// fetchLatestVersion reads the latest release tag from the GitHub releases API.
func fetchLatestVersion(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	// GitHub rejects API requests without a User-Agent.
	req.Header.Set("User-Agent", "deploys-cli")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api returned %s", resp.Status)
	}

	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return "", err
	}
	if body.TagName == "" {
		return "", errors.New("github api returned an empty tag_name")
	}
	return body.TagName, nil
}

// updateAvailable reports whether latest is newer than current. A build whose
// version isn't valid semver (a local `go build`, reported as "dev") can't be
// compared, so any real release is treated as an upgrade.
func updateAvailable(current, latest string) bool {
	cv, lv := normalizeVersion(current), normalizeVersion(latest)
	if !semver.IsValid(lv) {
		return false
	}
	if !semver.IsValid(cv) {
		return true
	}
	return semver.Compare(cv, lv) < 0
}

// normalizeVersion makes v comparable by golang.org/x/mod/semver, which
// requires a leading "v" (goreleaser strips it from the binary's version).
func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v != "" && !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

// displayVersion renders v for output: canonical "vX.Y.Z" when it is valid
// semver, otherwise the raw string (e.g. "dev").
func displayVersion(v string) string {
	if n := normalizeVersion(v); semver.IsValid(n) {
		return n
	}
	if v == "" {
		return "dev"
	}
	return v
}

func writeCheckUpdateUsage(w io.Writer) {
	fmt.Fprint(w, "check-update — check whether a newer deploys cli release is available\n\n")
	fmt.Fprint(w, "Usage:\n  deploys check-update [-output table|yaml|json]\n\n")
	fmt.Fprint(w, "Compares this binary's version against the latest stable release on GitHub\n")
	fmt.Fprint(w, "and prints whether an upgrade is available.\n")
}
