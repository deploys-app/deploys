package runner

import (
	"context"
	"flag"
	"fmt"

	"github.com/deploys-app/api/client"
)

// site handles `deploys site <subcommand>`. Publishing reads the local
// filesystem and uploads via the raw-HTTP /sites/* endpoints, so it uses the
// concrete *client.Client helper (PublishSite) rather than the api.Interface.
func (rn Runner) site(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)

	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
	case "publish":
		var opts client.SitePublishOptions
		f.StringVar(&opts.Project, "project", "", "project id")
		f.StringVar(&opts.Name, "name", "", "deployment name")
		f.StringVar(&opts.Dir, "dir", ".", "local directory to publish")
		f.StringVar(&opts.Environment, "environment", "", "release environment: production (default) or pr-<n>")
		f.BoolVar(&opts.SPA, "spa", false, "serve index.html for unmatched paths (single-page app)")
		f.StringVar(&opts.NotFound, "notFound", "", "custom 404 document path (e.g. 404.html)")
		f.Parse(args[1:])

		c, ok := rn.API.(*client.Client)
		if !ok {
			return fmt.Errorf("site publish requires the default api client")
		}
		res, err := c.PublishSite(context.Background(), &opts)
		if err != nil {
			return err
		}
		return rn.print(res)
	}
}
