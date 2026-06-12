package runner

import (
	"context"
	"flag"
	"fmt"

	"github.com/deploys-app/api"
)

func (rn Runner) dropbox(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Dropbox()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
	case "list":
		var req api.DropboxList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Var(timeFlag{&req.After}, "after", "only files after this time (RFC 3339 or YYYY-MM-DD)")
		f.Var(timeFlag{&req.Before}, "before", "only files before this time (RFC 3339 or YYYY-MM-DD)")
		f.IntVar(&req.Limit, "limit", 0, "max entries")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "metrics":
		var (
			req       api.DropboxMetrics
			timeRange string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&timeRange, "time-range", "30d", "time range (7d, 30d, 90d)")
		f.Parse(args[1:])
		req.TimeRange = api.UsageMetricsTimeRange(timeRange)
		resp, err = s.Metrics(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}
