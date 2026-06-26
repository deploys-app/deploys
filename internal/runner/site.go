package runner

import (
	"context"
	"fmt"
	"os"

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

		progress, finish := newPublishProgress(os.Stderr)
		opts.Progress = progress
		res, err := c.PublishSite(context.Background(), &opts)
		finish()
		if err != nil {
			return err
		}
		return rn.print(res)
	case "deploy":
		// deploy wraps publish → deploy → get into one command: it publishes the
		// local directory and deploys it as a permanent (non-TTL) Static
		// deployment, then prints the resulting deployment (url + releaseUrl). It
		// is the permanent counterpart of `preview`; the only differences are the
		// production environment default and that it clears any TTL.
		var (
			opts     client.SitePublishOptions
			location string
		)
		f.StringVar(&opts.Project, "project", "", "project id")
		f.StringVar(&opts.Name, "name", "", "deployment name")
		f.StringVar(&opts.Dir, "dir", ".", "local directory to publish")
		f.StringVar(&location, "location", "", "location")
		f.StringVar(&opts.Environment, "environment", "", "release environment: production (default) or pr-<n>")
		f.BoolVar(&opts.SPA, "spa", false, "serve index.html for unmatched paths (single-page app)")
		f.StringVar(&opts.NotFound, "notFound", "", "custom 404 document path (e.g. 404.html)")
		f.Parse(args[1:])

		c, ok := rn.API.(*client.Client)
		if !ok {
			return fmt.Errorf("site deploy requires the default api client")
		}

		// Permanent deploy: pass an explicit TTL of 0 so a deployment of the same
		// name that was previously a throwaway preview is promoted to a
		// non-expiring one (a nil TTL would leave its auto-delete in place).
		zero := int64(0)
		return rn.deployPublishedStatic(c, &opts, location, &zero)
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

		// ttl > 0 sets an auto-delete; ttl <= 0 (the documented "0 = no
		// auto-delete") leaves the deployment's TTL untouched.
		var ttlPtr *int64
		if ttl > 0 {
			ttlPtr = &ttl
		}
		return rn.deployPublishedStatic(c, &opts, location, ttlPtr)
	}
}

// deployPublishedStatic publishes opts.Dir (rendering an upload progress bar)
// and deploys the resulting site ref as a Static deployment at location, then
// prints the deployment (url + releaseUrl + expiresAt). ttl controls
// auto-delete: a nil ttl leaves the deployment's TTL untouched, while &n sets
// it (with 0 clearing any existing TTL). Shared by `site deploy` and
// `site preview`.
func (rn Runner) deployPublishedStatic(c *client.Client, opts *client.SitePublishOptions, location string, ttl *int64) error {
	// 1. Publish the local directory → a content-addressed site ref.
	progress, finish := newPublishProgress(os.Stderr)
	opts.Progress = progress
	pub, err := c.PublishSite(context.Background(), opts)
	finish()
	if err != nil {
		return err
	}

	// 2. Deploy the release as a Static deployment.
	deploy := &api.DeploymentDeploy{
		Project:  opts.Project,
		Location: location,
		Name:     opts.Name,
		Type:     api.DeploymentTypeStatic,
		Site:     pub.SiteRef,
		TTL:      ttl,
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
