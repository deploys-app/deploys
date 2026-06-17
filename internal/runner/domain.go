package runner

import (
	"context"

	"github.com/deploys-app/api"
)

func (rn Runner) domain(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("domain")
	}

	s := rn.API.Domain()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("domain", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("domain", args[0])
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
