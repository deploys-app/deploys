package runner

import (
	"context"
	"flag"
	"fmt"

	"github.com/deploys-app/api"
)

func (rn Runner) domain(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Domain()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
	case "create":
		var req api.DomainCreate
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Domain, "domain", "", "domain")
		f.BoolVar(&req.Wildcard, "wildcard", false, "wildcard domain")
		f.Parse(args[1:])
		resp, err = s.Create(context.Background(), &req)
	case "get":
		var req api.DomainGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Domain, "domain", "", "domain")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "list":
		var req api.DomainList
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "delete":
		var req api.DomainDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Domain, "domain", "", "domain")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	case "purgecache":
		var req api.DomainPurgeCache
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Domain, "domain", "", "domain")
		f.StringVar(&req.File, "file", "", "purge a single file path")
		f.StringVar(&req.Prefix, "prefix", "", "purge all files under a path prefix")
		f.Parse(args[1:])
		resp, err = s.PurgeCache(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}
