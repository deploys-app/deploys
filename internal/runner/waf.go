package runner

import (
	"context"
	"fmt"
	"os"

	"github.com/deploys-app/api"
	"gopkg.in/yaml.v2"
)

func (rn Runner) waf(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("waf")
	}

	s := rn.API.WAF()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("waf", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("waf", args[0])
	case "get":
		var req api.WAFGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "list":
		var req api.WAFList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "set":
		// Set replaces the whole zone (rules and limits) all-or-nothing, so it
		// takes a spec file rather than per-rule flags. The file is the yaml
		// form of waf get (description, rules, limits); project and location
		// flags override values in the file.
		var (
			fn          string
			project     string
			location    string
			description string
		)
		f.StringVar(&fn, "f", "", "spec file (yaml: description, rules, limits)")
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
		// non-strict so the yaml output of `waf get` (which carries extra
		// read-only fields) can be edited and fed back in
		var req api.WAFSet
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
		var req api.WAFDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	case "metrics":
		var (
			req       api.WAFMetrics
			timeRange string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&timeRange, "time-range", "1h", "time range (1h, 6h, 12h, 1d, 7d, 30d)")
		f.Parse(args[1:])
		req.TimeRange = api.WAFMetricsTimeRange(timeRange)
		resp, err = s.Metrics(context.Background(), &req)
	case "limitmetrics":
		var (
			req       api.WAFLimitMetrics
			timeRange string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&timeRange, "time-range", "1h", "time range (1h, 6h, 12h, 1d, 7d, 30d)")
		f.Parse(args[1:])
		req.TimeRange = api.WAFMetricsTimeRange(timeRange)
		resp, err = s.LimitMetrics(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}
