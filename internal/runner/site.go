package runner

import (
	"context"
	"fmt"

	"github.com/deploys-app/api"
	"github.com/deploys-app/api/client"
)

// site handles `deploys site <subcommand>`. Publishing reads the local
// filesystem and uploads via the raw-HTTP /sites/* endpoints, so it uses the
// concrete *client.Client helper (PublishSite) rather than the api.Interface.
func (rn Runner) site(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("site")
	}

	f := rn.subFlagSet("site", args[0])

	switch args[0] {
	default:
		return rn.unknownSub("site", args[0])
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
	case "preview":
		// preview wraps publish → deploy(ttl) → get into one throwaway-preview
		// step: it publishes the local directory, deploys it as a TTL'd Static
		// deployment, and prints the resulting deployment (url + releaseUrl +
		// expiresAt). It auto-deletes at the TTL; `deployment extend-ttl` keeps it
		// alive and `deployment delete` drops it early.
		var (
			opts     client.SitePublishOptions
			location string
			ttl      int64
			force    bool
		)
		f.StringVar(&opts.Project, "project", "", "project id")
		f.StringVar(&opts.Name, "name", "", "deployment name")
		f.StringVar(&opts.Dir, "dir", ".", "local directory to publish")
		f.StringVar(&location, "location", "", "location")
		// Default to a non-production environment so the preview is served
		// X-Robots-Tag: noindex (the gateway only omits noindex for production).
		f.StringVar(&opts.Environment, "environment", "preview", "release environment (non-production previews are served noindex)")
		f.BoolVar(&opts.SPA, "spa", false, "serve index.html for unmatched paths (single-page app)")
		f.StringVar(&opts.NotFound, "notFound", "", "custom 404 document path (e.g. 404.html)")
		f.Int64Var(&ttl, "ttl", 7200, "seconds until the preview auto-deletes (0 = no auto-delete)")
		f.BoolVar(&force, "force", false, "overwrite an existing non-preview (production) deployment of the same name")
		f.Parse(args[1:])

		c, ok := rn.API.(*client.Client)
		if !ok {
			return fmt.Errorf("site preview requires the default api client")
		}

		// Safety: a throwaway preview must not silently convert an existing
		// non-preview (production) deployment into an auto-deleting one. If the
		// target already exists and has no TTL, refuse unless -force. Fail open on
		// any lookup error (not-found, forbidden, etc.) so the guard never blocks a
		// legitimate preview — it only trips on a clean "exists and is permanent".
		if !force {
			if existing, gerr := c.Deployment().Get(context.Background(), &api.DeploymentGet{
				Project:  opts.Project,
				Location: location,
				Name:     opts.Name,
			}); gerr == nil && existing.TTL == 0 {
				return fmt.Errorf("deployment %q already exists and is not a preview (no ttl); pass -force to overwrite it as a throwaway preview", opts.Name)
			}
		}

		// 1. Publish the local directory → a content-addressed site ref.
		pub, err := c.PublishSite(context.Background(), &opts)
		if err != nil {
			return err
		}

		// 2. Deploy it as a TTL'd Static preview.
		deploy := &api.DeploymentDeploy{
			Project:  opts.Project,
			Location: location,
			Name:     opts.Name,
			Type:     api.DeploymentTypeStatic,
			Site:     pub.SiteRef,
		}
		if ttl > 0 {
			deploy.TTL = &ttl
		}
		if _, err := c.Deployment().Deploy(context.Background(), deploy); err != nil {
			return err
		}

		// 3. Read back the rolling url, the immutable releaseUrl, and expiresAt.
		got, err := c.Deployment().Get(context.Background(), &api.DeploymentGet{
			Project:  opts.Project,
			Location: location,
			Name:     opts.Name,
		})
		if err != nil {
			return err
		}
		return rn.print(got)
	}
}
