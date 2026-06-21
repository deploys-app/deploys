package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/deploys-app/api"
	"github.com/deploys-app/api/client"
)

func (rn Runner) dropbox(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("dropbox")
	}

	s := rn.API.Dropbox()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("dropbox", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("dropbox", args[0])
	case "list":
		var req api.DropboxList
		f.StringVar(&req.Project, "project", "", "project sid")
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
		f.StringVar(&req.Project, "project", "", "project sid")
		f.StringVar(&timeRange, "time-range", "30d", "time range (7d, 30d, 90d)")
		f.Parse(args[1:])
		req.TimeRange = api.UsageMetricsTimeRange(timeRange)
		resp, err = s.Metrics(context.Background(), &req)
	case "upload":
		var opts client.DropboxUploadOptions
		var file string
		f.StringVar(&opts.Project, "project", "", "project sid")
		f.StringVar(&file, "file", "", "path to the file to upload, or - for stdin (default stdin)")
		f.StringVar(&opts.Filename, "filename", "", "filename recorded in the download (defaults to the base name of -file)")
		f.IntVar(&opts.TTLDays, "ttl", 0, "download lifetime in days, 1-7 (default 1)")
		f.Parse(args[1:])

		c, ok := rn.API.(*client.Client)
		if !ok {
			return fmt.Errorf("dropbox upload requires the default api client")
		}

		if file == "" || file == "-" {
			opts.Content, err = io.ReadAll(os.Stdin)
		} else {
			opts.Content, err = os.ReadFile(file)
			if opts.Filename == "" {
				opts.Filename = filepath.Base(file)
			}
		}
		if err != nil {
			return err
		}
		resp, err = c.DropboxUpload(context.Background(), &opts)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}
