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
		// Set replaces the whole list all-or-nothing (mirrors waf set), so
		// entries always arrive whole: repeated -entry flags, an -entries-file,
		// or -f with the yaml form of wafList get. Entry flags and the other
		// flags override values in the file.
		var (
			fn          string
			entries     multiFlag
			entriesFile string
			project     string
			name        string
			description string
			listType    string
		)
		f.StringVar(&fn, "f", "", "spec file (yaml: name, description, type, entries)")
		f.Var(&entries, "entry", "list entry, an IP or CIDR; repeatable")
		f.StringVar(&entriesFile, "entries-file", "", "file with one entry per line (# comments stripped)")
		f.StringVar(&project, "project", "", "project id")
		f.StringVar(&name, "name", "", "list name")
		f.StringVar(&description, "description", "", "list description")
		f.StringVar(&listType, "type", "", "list type (v1: ip)")
		f.Parse(args[1:])

		if fn == "" && entriesFile == "" && len(entries) == 0 {
			return fmt.Errorf("entries required (-entry, -entries-file, or -f)")
		}

		var req api.WAFListSet
		if fn != "" {
			b, ferr := os.ReadFile(fn)
			if ferr != nil {
				return ferr
			}
			// non-strict so the yaml output of `wafList get` (which carries
			// extra read-only fields) can be edited and fed back in
			ferr = yaml.Unmarshal(b, &req)
			if ferr != nil {
				return fmt.Errorf("parse %s: %w", fn, ferr)
			}
		}
		if entriesFile != "" {
			b, ferr := os.ReadFile(entriesFile)
			if ferr != nil {
				return ferr
			}
			req.Entries = append(parseWAFListEntries(string(b)), entries...)
		} else if len(entries) > 0 {
			req.Entries = entries
		}
		if project != "" {
			req.Project = project
		}
		if name != "" {
			req.Name = name
		}
		if description != "" {
			req.Description = description
		}
		if listType != "" {
			req.Type = api.WAFListType(listType)
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
