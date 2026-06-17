package runner

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestIsHelpArg(t *testing.T) {
	for _, s := range []string{"help", "-h", "-help", "--help"} {
		if !IsHelpArg(s) {
			t.Errorf("IsHelpArg(%q) = false; want true", s)
		}
	}
	for _, s := range []string{"", "get", "h", "halp", "-help-me", "list"} {
		if IsHelpArg(s) {
			t.Errorf("IsHelpArg(%q) = true; want false", s)
		}
	}
}

// subFlagSet must bind the shared -output flag to the SAME Runner whose print()
// later renders the response, otherwise -output/-ojson/-oyaml are silently
// ignored. This guards against the binding leaking into a value-receiver copy.
func TestSubFlagSetBindsOutputMode(t *testing.T) {
	rn := Runner{Output: os.Stdout}
	f := rn.subFlagSet("project", "list")
	if err := f.Parse([]string{"-output", "json"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if rn.OutputMode != "json" {
		t.Fatalf("OutputMode = %q; want json (output flag not bound to the caller's Runner)", rn.OutputMode)
	}
}

func TestLookupCommand(t *testing.T) {
	if c := lookupCommand("deployment"); c == nil || c.name != "deployment" {
		t.Fatalf("lookupCommand(deployment) = %v", c)
	}
	// aliases resolve to the canonical group
	for _, alias := range []string{"deploy", "d"} {
		if c := lookupCommand(alias); c == nil || c.name != "deployment" {
			t.Errorf("lookupCommand(%q) did not resolve to deployment: %v", alias, c)
		}
	}
	if c := lookupCommand("nope"); c != nil {
		t.Errorf("lookupCommand(nope) = %v; want nil", c)
	}
}

func TestLookupSubAlias(t *testing.T) {
	me := lookupCommand("me")
	if me == nil {
		t.Fatal("me group missing")
	}
	// generate-token is also reachable by its camelCase alias
	if s := me.lookupSub("generateToken"); s == nil || s.name != "generate-token" {
		t.Errorf("lookupSub(generateToken) = %v; want generate-token", s)
	}
	// the hidden "set image" leaf feeds the banner but is excluded from listings
	dep := lookupCommand("deployment")
	if s := dep.lookupSub("set image"); s == nil || !s.hidden {
		t.Errorf("lookupSub(set image) = %v; want a hidden entry", s)
	}
}

func TestWriteSubUsage(t *testing.T) {
	rn := Runner{Output: os.Stdout}
	f := rn.subFlagSet("me", "generate-token")
	f.String("ttl", "", "token lifetime")

	var buf bytes.Buffer
	writeSubUsage(&buf, f, "me", "generate-token")
	out := buf.String()

	for _, want := range []string{
		"mint a short-lived",               // the short description
		"Usage: deploys me generate-token", // the usage line with the args hint
		"-output",                          // a flag from the live set
		"-ttl",                             // a case-specific flag
	} {
		if !strings.Contains(out, want) {
			t.Errorf("writeSubUsage output missing %q:\n%s", want, out)
		}
	}
}

func TestPrintUsageListsEveryGroup(t *testing.T) {
	var buf bytes.Buffer
	PrintUsage(&buf)
	out := buf.String()
	for _, c := range commands {
		if !strings.Contains(out, c.name) {
			t.Errorf("top-level usage missing group %q", c.name)
		}
	}
	// hidden leaves must not surface in the top-level subcommand listing
	if strings.Contains(out, "set image") {
		t.Errorf("top-level usage should not list the hidden \"set image\" leaf:\n%s", out)
	}
}

// Help at the top and group levels must render without touching the API (so a
// nil API never panics) and without returning an error.
func TestRunHelpNeedsNoAPI(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "help")
	if err != nil {
		t.Fatal(err)
	}
	defer tmp.Close()

	cases := []struct {
		args     []string
		contains string
	}{
		{nil, "deploys.app cli"},
		{[]string{"help"}, "deploys.app cli"},
		{[]string{"-h"}, "deploys.app cli"},
		{[]string{"--help"}, "deploys.app cli"},
		{[]string{"me"}, "Subcommands:"},
		{[]string{"me", "help"}, "identity and access"},
		{[]string{"d", "-h"}, "deployments and their lifecycle"},
	}
	for _, tc := range cases {
		if err := tmp.Truncate(0); err != nil {
			t.Fatal(err)
		}
		if _, err := tmp.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		rn := Runner{Output: tmp} // API intentionally nil
		if err := rn.Run(tc.args...); err != nil {
			t.Errorf("Run(%v) error: %v", tc.args, err)
			continue
		}
		b, _ := os.ReadFile(tmp.Name())
		if !strings.Contains(string(b), tc.contains) {
			t.Errorf("Run(%v) output missing %q:\n%s", tc.args, tc.contains, b)
		}
	}
}

func TestRunUnknownCommand(t *testing.T) {
	rn := Runner{Output: os.Stdout}
	if err := rn.Run("definitelynotacommand"); err == nil {
		t.Error("unknown top-level command should return an error")
	}
}

// Every registry entry must carry a description and resolve by its name and
// aliases, so no -h banner or listing renders blank. This guards the registry's
// internal consistency; it cannot see a dispatch `case` that was never listed
// (see the INVARIANT comment on commands).
func TestRegistryDescriptions(t *testing.T) {
	for _, c := range commands {
		if c.name == "" || c.short == "" {
			t.Errorf("group %q has an empty name or short", c.name)
		}
		if lookupCommand(c.name) != &commands[indexOf(c.name)] {
			t.Errorf("group %q does not resolve to itself", c.name)
		}
		for _, a := range c.aliases {
			if lookupCommand(a) == nil {
				t.Errorf("group %q alias %q does not resolve", c.name, a)
			}
		}
		for _, s := range c.subs {
			if s.name == "" || s.short == "" {
				t.Errorf("group %q sub %q has an empty name or short", c.name, s.name)
			}
			if c.lookupSub(s.name) == nil {
				t.Errorf("group %q sub %q does not resolve by name", c.name, s.name)
			}
			for _, a := range s.aliases {
				if c.lookupSub(a) == nil {
					t.Errorf("group %q sub %q alias %q does not resolve", c.name, s.name, a)
				}
			}
		}
	}
}

func indexOf(group string) int {
	for i := range commands {
		if commands[i].name == group {
			return i
		}
	}
	return -1
}

// The deploy banner is the one help path that goes through a free function
// rather than subFlagSet; it must still honor the injected writer (Runner.Output
// in production) instead of writing to a hardcoded stream.
func TestDeployHelpHonorsWriter(t *testing.T) {
	var buf bytes.Buffer
	_, _, err := parseDeploymentDeploy(&buf, []string{"-h"})
	if err == nil {
		t.Fatal("expected flag.ErrHelp for -h")
	}
	out := buf.String()
	for _, want := range []string{"create or update a deployment", "Usage: deploys deployment deploy", "-image"} {
		if !strings.Contains(out, want) {
			t.Errorf("deploy -h banner missing %q:\n%s", want, out)
		}
	}
}
