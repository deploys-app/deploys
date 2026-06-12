package runner

import (
	"context"
	"flag"
	"fmt"

	"github.com/deploys-app/api"
)

func (rn Runner) registry(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Registry()

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
		var req api.RegistryList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "get":
		var req api.RegistryGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Repository, "repository", "", "repository name")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "tags":
		var req api.RegistryGetTags
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Repository, "repository", "", "repository name")
		f.Parse(args[1:])
		resp, err = s.GetTags(context.Background(), &req)
	case "manifests":
		var req api.RegistryGetManifests
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Repository, "repository", "", "repository name")
		f.Parse(args[1:])
		resp, err = s.GetManifests(context.Background(), &req)
	case "storage":
		var req api.RegistryGetProjectStorage
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.GetProjectStorage(context.Background(), &req)
	case "delete":
		var req api.RegistryDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Repository, "repository", "", "repository name")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	case "deletemanifest":
		var req api.RegistryDeleteManifest
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Repository, "repository", "", "repository name")
		f.StringVar(&req.Digest, "digest", "", "manifest digest")
		f.Parse(args[1:])
		resp, err = s.DeleteManifest(context.Background(), &req)
	case "untag":
		var req api.RegistryUntag
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Repository, "repository", "", "repository name")
		f.StringVar(&req.Tag, "tag", "", "tag")
		f.Parse(args[1:])
		resp, err = s.Untag(context.Background(), &req)
	case "metrics":
		var (
			req       api.RegistryMetrics
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
