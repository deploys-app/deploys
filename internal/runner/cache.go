package runner

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/deploys-app/api"
	"gopkg.in/yaml.v2"
)

func (rn Runner) cache(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Cache()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
	case "get":
		var req api.CacheGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "list":
		var req api.CacheList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "set":
		// Set replaces the whole zone (all overrides) all-or-nothing, so it takes
		// a spec file rather than per-override flags. The file is the yaml form of
		// `cache get` (description, overrides); project, location, and description
		// flags override values in the file.
		var (
			fn          string
			project     string
			location    string
			description string
		)
		f.StringVar(&fn, "f", "", "spec file (yaml: description, overrides)")
		f.StringVar(&project, "project", "", "project id")
		f.StringVar(&location, "location", "", "location")
		f.StringVar(&description, "description", "", "zone description")
		f.Parse(args[1:])

		if fn == "" {
			return fmt.Errorf("spec file required (-f)")
		}
		b, ferr := os.ReadFile(fn)
		if ferr != nil {
			return ferr
		}
		// non-strict so the yaml output of `cache get` (which carries extra
		// read-only fields) can be edited and fed back in
		var req api.CacheSet
		ferr = yaml.Unmarshal(b, &req)
		if ferr != nil {
			return fmt.Errorf("parse %s: %w", fn, ferr)
		}
		if project != "" {
			req.Project = project
		}
		if location != "" {
			req.Location = location
		}
		if description != "" {
			req.Description = description
		}
		resp, err = s.Set(context.Background(), &req)
	case "delete":
		var req api.CacheDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	case "metrics":
		var (
			req       api.CacheMetrics
			timeRange string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&timeRange, "time-range", "1h", "time range (1h, 6h, 12h, 1d, 7d, 30d)")
		f.Parse(args[1:])
		req.TimeRange = api.WAFMetricsTimeRange(timeRange)
		resp, err = s.Metrics(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}
