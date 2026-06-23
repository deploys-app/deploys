package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/deploys-app/api"

	"github.com/deploys-app/deploys/internal/auth"
)

func TestExtractAccountFlag(t *testing.T) {
	cases := []struct {
		name     string
		in       []string
		wantSel  string
		wantRest []string
	}{
		{"none", []string{"project", "list"}, "", []string{"project", "list"}},
		{"space form", []string{"-account", "a@x", "project", "list"}, "a@x", []string{"project", "list"}},
		{"eq form", []string{"-account=a@x", "project", "list"}, "a@x", []string{"project", "list"}},
		{"double dash", []string{"--account", "a@x", "me", "get"}, "a@x", []string{"me", "get"}},
		{"double dash eq", []string{"--account=a@x", "me", "get"}, "a@x", []string{"me", "get"}},
		{"after subcommand", []string{"deployment", "list", "-account", "a@x"}, "a@x", []string{"deployment", "list"}},
		{"only flag", []string{"-account", "a@x"}, "a@x", []string{}},
		{"last wins", []string{"-account", "a@x", "x", "-account=b@y"}, "b@y", []string{"x"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sel, rest := extractAccountFlag(c.in)
			if sel != c.wantSel {
				t.Errorf("selector = %q; want %q", sel, c.wantSel)
			}
			if !reflect.DeepEqual(rest, c.wantRest) {
				t.Errorf("rest = %#v; want %#v", rest, c.wantRest)
			}
		})
	}
}

func TestExitCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"auth required", &auth.AuthRequiredError{Msg: "not logged in"}, 4},
		{"wrapped auth required", fmt.Errorf("ctx: %w", &auth.AuthRequiredError{Msg: "x"}), 4},
		{"unauthorized", api.ErrUnauthorized, 4},
		{"wrapped unauthorized", fmt.Errorf("ctx: %w", api.ErrUnauthorized), 4},
		{"forbidden stays 1", api.ErrForbidden, 1},
		{"generic", fmt.Errorf("boom"), 1},
	}
	for _, c := range cases {
		if got := exitCode(c.err); got != c.want {
			t.Errorf("%s: exitCode = %d; want %d", c.name, got, c.want)
		}
	}
}
