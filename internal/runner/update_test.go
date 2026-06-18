package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestUpdateAvailable(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"1.0.0", "v1.1.3", true},    // older, goreleaser's v-less form
		{"v1.1.3", "v1.1.3", false},  // same
		{"v2.0.0", "v1.1.3", false},  // ahead (local/dev tag)
		{"dev", "v1.1.3", true},      // unknown build -> any release is newer
		{"", "v1.1.3", true},         // empty build -> treat as needing update
		{"v1.0.0", "garbage", false}, // bad latest -> never claim an update
		{"v1.2.0", "v1.10.0", true},  // numeric, not lexical, comparison
	}
	for _, tc := range cases {
		if got := updateAvailable(tc.current, tc.latest); got != tc.want {
			t.Errorf("updateAvailable(%q, %q) = %v; want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestDisplayVersion(t *testing.T) {
	cases := []struct{ in, want string }{
		{"1.1.3", "v1.1.3"},
		{"v1.1.3", "v1.1.3"},
		{"dev", "dev"},
		{"", "dev"},
	}
	for _, tc := range cases {
		if got := displayVersion(tc.in); got != tc.want {
			t.Errorf("displayVersion(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestFetchLatestVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request missing User-Agent (GitHub rejects it)")
		}
		w.Write([]byte(`{"tag_name":"v1.2.3","name":"v1.2.3"}`))
	}))
	defer srv.Close()

	got, err := fetchLatestVersion(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("fetchLatestVersion: %v", err)
	}
	if got != "v1.2.3" {
		t.Errorf("tag = %q; want v1.2.3", got)
	}
}

func TestFetchLatestVersionErrors(t *testing.T) {
	// Non-200 is an error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	if _, err := fetchLatestVersion(context.Background(), srv.URL); err == nil {
		t.Error("expected error on 404")
	}
	srv.Close()

	// Empty tag_name is an error.
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	if _, err := fetchLatestVersion(context.Background(), srv.URL); err == nil {
		t.Error("expected error on empty tag_name")
	}
}

// checkUpdate end-to-end against a stub release endpoint, including the upgrade
// hint shown in the default table view.
func TestCheckUpdateReportsUpgrade(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v9.9.9"}`))
	}))
	defer srv.Close()

	old := githubLatestURL
	githubLatestURL = srv.URL
	defer func() { githubLatestURL = old }()

	tmp, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer tmp.Close()

	rn := Runner{Output: tmp, Version: "1.0.0"} // API intentionally nil
	if err := rn.checkUpdate(); err != nil {
		t.Fatalf("checkUpdate: %v", err)
	}

	b, _ := os.ReadFile(tmp.Name())
	out := string(b)
	for _, want := range []string{"v1.0.0", "v9.9.9", "go install", "Upgrade"} {
		if !strings.Contains(out, want) {
			t.Errorf("checkUpdate output missing %q:\n%s", want, out)
		}
	}
}

// -h must render usage without an API client or a network call.
func TestCheckUpdateHelpNeedsNoNetwork(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer tmp.Close()

	rn := Runner{Output: tmp}
	if err := rn.checkUpdate("-h"); err != nil {
		t.Fatalf("checkUpdate -h: %v", err)
	}
	b, _ := os.ReadFile(tmp.Name())
	if !strings.Contains(string(b), "check whether a newer deploys cli release") {
		t.Errorf("check-update -h missing banner:\n%s", b)
	}
}

func TestIsLocalCommand(t *testing.T) {
	for _, c := range []string{"check-update", "version"} {
		if !IsLocalCommand(c) {
			t.Errorf("%q should be a local command", c)
		}
	}
	for _, c := range []string{"me", "deployment", "site", ""} {
		if IsLocalCommand(c) {
			t.Errorf("%q should not be a local command", c)
		}
	}
}

// version prints the bare, canonicalized version by default and a structured
// object for -ojson; neither path needs an API client.
func TestVersion(t *testing.T) {
	cases := []struct {
		args    []string
		version string
		want    string
	}{
		{nil, "1.1.3", "v1.1.3\n"},                         // bare, v-prefixed
		{[]string{"-output", "json"}, "1.1.3", `"v1.1.3"`}, // structured
		{nil, "dev-1a2b3c4", "dev-1a2b3c4\n"},              // non-semver stays raw
		{nil, "", "dev\n"},                                 // unknown build
	}
	for _, tc := range cases {
		tmp, err := os.CreateTemp(t.TempDir(), "ver")
		if err != nil {
			t.Fatal(err)
		}
		rn := Runner{Output: tmp, Version: tc.version} // API intentionally nil
		if err := rn.version(tc.args...); err != nil {
			t.Fatalf("version(%v): %v", tc.args, err)
		}
		tmp.Close()
		b, _ := os.ReadFile(tmp.Name())
		if !strings.Contains(string(b), tc.want) {
			t.Errorf("version(%v) version=%q = %q; want it to contain %q", tc.args, tc.version, b, tc.want)
		}
	}
}

func TestVersionHelp(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "ver")
	if err != nil {
		t.Fatal(err)
	}
	defer tmp.Close()
	rn := Runner{Output: tmp, Version: "1.1.3"}
	if err := rn.version("-h"); err != nil {
		t.Fatalf("version -h: %v", err)
	}
	b, _ := os.ReadFile(tmp.Name())
	out := string(b)
	if !strings.Contains(out, "print the deploys cli version") {
		t.Errorf("version -h missing banner:\n%s", out)
	}
	// -h must not print the version itself, only usage.
	if strings.Contains(out, "v1.1.3") {
		t.Errorf("version -h should not print the version:\n%s", out)
	}
}
