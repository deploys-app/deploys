package runner

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/deploys-app/api"
	"gopkg.in/yaml.v2"
)

func (rn Runner) wafList(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("wafList")
	}

	s := rn.API.WAFList()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("wafList", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("wafList", args[0])
	case "get":
		var req api.WAFListGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "list name")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "list":
		var req api.WAFListList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "set":
		var v wafListSetFlags
		f.StringVar(&v.fn, "f", "", "spec file (yaml: name, description, type, entries)")
		f.Var(&v.entries, "entry", "list entry, an IP or CIDR; repeatable, appended to file entries")
		f.StringVar(&v.entriesFile, "entries-file", "", "file with one entry per line (# comments stripped); overrides -f entries")
		f.StringVar(&v.project, "project", "", "project id")
		f.StringVar(&v.name, "name", "", "list name")
		f.StringVar(&v.description, "description", "", "list description")
		f.StringVar(&v.listType, "type", "", "list type (v1: ip)")
		f.Parse(args[1:])

		req, ferr := buildWAFListSet(v)
		if ferr != nil {
			return ferr
		}
		resp, err = s.Set(context.Background(), &req)
	case "delete":
		var req api.WAFListDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "list name")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

// wafListSetFlags holds the parsed `wafList set` flag values, keeping the
// flag→request assembly in buildWAFListSet pure (unit-testable without an
// API or a flag.FlagSet).
type wafListSetFlags struct {
	fn          string
	entries     multiFlag
	entriesFile string
	project     string
	name        string
	description string
	listType    string
}

// buildWAFListSet assembles the set request. Set replaces the whole list
// all-or-nothing (mirrors waf set), so entries always arrive whole. The base
// entries come from -entries-file when given, else from the -f yaml's
// entries; -entry flags are appended to that base (so "file plus extras"
// works and never silently shrinks a list). Scalar flags override -f values.
//
// A request whose entries came from nowhere is refused: emptying a list must
// be explicit (`entries: []` in -f), never the accident of a missing flag,
// an absent yaml key, or an -entries-file that parsed to nothing — an empty
// list expands to (false), so a block list would silently stop blocking.
func buildWAFListSet(v wafListSetFlags) (api.WAFListSet, error) {
	var req api.WAFListSet
	if v.fn != "" {
		b, err := os.ReadFile(v.fn)
		if err != nil {
			return req, err
		}
		// non-strict so the yaml output of `wafList get` (which carries
		// extra read-only fields) can be edited and fed back in
		err = yaml.Unmarshal(b, &req)
		if err != nil {
			return req, fmt.Errorf("parse %s: %w", v.fn, err)
		}
	}
	if v.entriesFile != "" {
		b, err := os.ReadFile(v.entriesFile)
		if err != nil {
			return req, err
		}
		es := parseWAFListEntries(string(b))
		if len(es) == 0 {
			return req, fmt.Errorf("%s: no entries (empty or comments only)", v.entriesFile)
		}
		req.Entries = es
	}
	if len(v.entries) > 0 {
		req.Entries = append(req.Entries, v.entries...)
	}
	if req.Entries == nil {
		return req, fmt.Errorf("entries required (-entry, -entries-file, or entries in -f)")
	}
	if v.project != "" {
		req.Project = v.project
	}
	if v.name != "" {
		req.Name = v.name
	}
	if v.description != "" {
		req.Description = v.description
	}
	if v.listType != "" {
		req.Type = api.WAFListType(v.listType)
	}
	return req, nil
}

// parseWAFListEntries parses an -entries-file: one entry per line, with blank
// lines and #-comments (whole-line or trailing) stripped.
func parseWAFListEntries(s string) []string {
	var res []string
	for line := range strings.Lines(s) {
		line, _, _ = strings.Cut(line, "#")
		line = strings.TrimSpace(line)
		if line != "" {
			res = append(res, line)
		}
	}
	return res
}
