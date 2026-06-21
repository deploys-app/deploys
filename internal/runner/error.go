package runner

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/deploys-app/api"
)

// errorGroup handles the top-level `error` command group, backed by the
// `error.*` API resource (api.Errors): grouped, deduplicated application-error
// issues with a triage lifecycle. It mirrors the flat-group pattern (see cache,
// disk): IsHelpArg → group usage, subFlagSet per leaf, rn.print(resp).
func (rn Runner) errorGroup(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("error")
	}

	s := rn.API.Errors()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("error", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("error", args[0])
	case "list":
		var req api.ErrorList
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location (optional; narrows a project-wide listing, required when -name is set)")
		f.StringVar(&req.Name, "name", "", "deployment name (omit to list issues across the whole project)")
		f.StringVar(&req.Status, "status", "", "triage status filter: open (default), resolved, muted, all")
		f.StringVar(&req.Sort, "sort", "", "sort order: lastSeen (default), firstSeen, count")
		f.IntVar(&req.Limit, "limit", 0, "max issues per page (default 50, max 200)")
		f.StringVar(&req.Cursor, "cursor", "", "opaque page cursor from a previous response's nextCursor")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "get":
		var req api.ErrorGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.StringVar(&req.ID, "id", "", "error issue id")
		f.Parse(args[1:])
		got, gerr := s.Get(context.Background(), &req)
		if gerr != nil {
			return gerr
		}
		// In table mode print the issue summary, then the sample stack and recent
		// occurrences, which the result's flat Table() omits. Other output modes
		// (yaml/json) carry the full struct already, so fall through to print.
		if rn.OutputMode == "" || rn.OutputMode == "table" {
			return rn.printErrorIssueDetail(got)
		}
		resp = got
	case "update":
		var req api.ErrorUpdate
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.StringVar(&req.ID, "id", "", "error issue id")
		f.StringVar(&req.Status, "status", "", "new triage status: resolved, open (reopen), or muted")
		f.Parse(args[1:])
		resp, err = s.Update(context.Background(), &req)
	case "report":
		// report sends a single, minimal error event (one ErrorReport in Events).
		// Frames are optional — omitting them fingerprints by Type alone, which is
		// fine for a hand-reported error.
		var (
			req                           api.ErrorCreate
			kind, typ, title, sample, pod string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.StringVar(&typ, "type", "", "exception/panic class, e.g. TypeError or panic (required)")
		f.StringVar(&kind, "kind", "", "language/runtime family: go, java, python, node, ruby, generic (default generic)")
		f.StringVar(&title, "title", "", "optional display line (type + first message)")
		f.StringVar(&sample, "sample", "", "optional full stack-trace text")
		f.StringVar(&pod, "pod", "", "reporting instance/host (default \"reported\")")
		f.Parse(args[1:])
		req.Events = []api.ErrorReport{{
			Kind:   kind,
			Type:   typ,
			Title:  title,
			Sample: sample,
			Pod:    pod,
		}}
		resp, err = s.Create(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

// printErrorIssueDetail renders an error issue's full detail in table mode: the
// summary (from the result's Table()), then the representative stack trace and
// the recent occurrence pointers that deep-link back into logs history.
func (rn Runner) printErrorIssueDetail(resp *api.ErrorGetResult) error {
	rn.printTable(resp.Table())

	out := rn.output()
	i := resp.Issue
	if i.SampleMessage != "" {
		fmt.Fprintf(out, "\nSample:\n%s\n", i.SampleMessage)
	}
	if len(i.RecentEvents) > 0 {
		fmt.Fprint(out, "\nRecent occurrences:\n")
		table := [][]string{{"TIMESTAMP", "POD", "OBJECT", "OFFSET"}}
		for _, e := range i.RecentEvents {
			ts := ""
			if !e.Timestamp.IsZero() {
				ts = e.Timestamp.UTC().Format(time.RFC3339)
			}
			table = append(table, []string{ts, e.Pod, e.Object, strconv.Itoa(e.Offset)})
		}
		rn.printTable(table)
	}
	return nil
}
