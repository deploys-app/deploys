package runner

import (
	"context"

	"github.com/deploys-app/api"
)

func (rn Runner) metricSource(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("metricsource")
	}

	s := rn.API.MetricSource()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("metricsource", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("metricsource", args[0])

	case "set":
		var req api.MetricSourceSet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "metric source name")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Deployment, "deployment", "", "deployment to scrape")
		f.IntVar(&req.Port, "port", 0, "scrape port")
		f.StringVar(&req.Path, "path", "/metrics", "scrape path (must start with /)")
		f.BoolVar(&req.Disabled, "disabled", false, "disable scraping")
		f.Parse(args[1:])
		resp, err = s.Set(context.Background(), &req)

	case "get":
		var req api.MetricSourceGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "metric source name")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)

	case "list":
		var req api.MetricSourceList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)

	case "delete":
		var req api.MetricSourceDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "metric source name")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)

	case "series":
		var req api.MetricSourceSeries
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "metric source name")
		f.Parse(args[1:])
		resp, err = s.Series(context.Background(), &req)

	case "query":
		var (
			req       api.MetricSourceQuery
			series    multiFlag
			timeRange string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "metric source name")
		f.Var(&series, "series", "series key (repeatable; omit to let the server pick)")
		f.StringVar(&timeRange, "timerange", "1h", "time range (1h, 6h, 12h, 1d, 7d, 30d)")
		f.Parse(args[1:])
		req.Series = series
		req.TimeRange = timeRange
		resp, err = s.Query(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}
