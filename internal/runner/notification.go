package runner

import (
	"context"
	"time"

	"github.com/deploys-app/api"
)

func (rn Runner) notification(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("notification")
	}

	s := rn.API.Notification()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("notification", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("notification", args[0])

	case "list":
		var req api.NotificationList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)

	case "get":
		var req api.NotificationGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "channel name")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)

	case "create":
		var (
			req         api.NotificationCreate
			typ         string
			url         string
			secret      string
			insecureTLS bool
			pullTTL     int
			events      multiFlag
			outcomes    multiFlag
			disabled    bool
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "channel name")
		f.StringVar(&typ, "type", "", "channel type: webhook, discord, or pull")
		f.StringVar(&url, "url", "", "delivery URL (http/https; webhook/discord only)")
		f.StringVar(&secret, "secret", "", "webhook signing secret (required for webhook)")
		f.BoolVar(&insecureTLS, "insecure-tls", false, "skip TLS verification for HTTPS targets")
		f.IntVar(&pullTTL, "pull-ttl", 0, "pull channel inactivity TTL in seconds before auto-delete (0 = server default; 60-86400)")
		f.Var(&events, "event", "resource.action event to subscribe to: *, deployment.*, *.delete, deployment.deploy (repeatable; empty = all)")
		f.Var(&outcomes, "outcome", "outcome to subscribe to: success or failure (repeatable; empty = all)")
		f.BoolVar(&disabled, "disabled", false, "create the channel disabled")
		f.Parse(args[1:])
		req.Config = api.NotificationConfig{Type: typ, URL: url, Secret: secret, InsecureSkipVerify: insecureTLS, PullTTLSeconds: pullTTL}
		req.Subscription = api.NotificationSubscription{
			Events:   []string(events),
			Outcomes: []string(outcomes),
		}
		req.Disabled = disabled
		resp, err = s.Create(context.Background(), &req)

	case "update":
		// Merge semantics: seed from the existing channel, override only the flags
		// the user explicitly passed (visitedFlags). The signing secret is never
		// returned by Get, so an omitted -secret leaves it empty and the server
		// keeps the stored one. A subscription axis is replaced only when at least
		// one value is passed for it.
		var (
			req         api.NotificationUpdate
			typ         string
			url         string
			secret      string
			insecureTLS bool
			pullTTL     int
			events      multiFlag
			outcomes    multiFlag
			disabled    bool
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "channel name")
		f.StringVar(&typ, "type", "", "channel type: webhook, discord, or pull")
		f.StringVar(&url, "url", "", "delivery URL (http/https)")
		f.StringVar(&secret, "secret", "", "webhook signing secret (omit to keep existing)")
		f.BoolVar(&insecureTLS, "insecure-tls", false, "skip TLS verification for HTTPS targets")
		f.IntVar(&pullTTL, "pull-ttl", 0, "pull channel inactivity TTL in seconds (0 = server default; 60-86400)")
		f.Var(&events, "event", "resource.action event to subscribe to: *, deployment.*, *.delete (repeatable; replaces all)")
		f.Var(&outcomes, "outcome", "outcome to subscribe to (repeatable; replaces all)")
		f.BoolVar(&disabled, "disabled", false, "disable the channel")
		f.Parse(args[1:])
		set := visitedFlags(f)

		// A distinct name avoids shadowing the outer err so a later Update error
		// still surfaces after the switch.
		cur, getErr := s.Get(context.Background(), &api.NotificationGet{Project: req.Project, Name: req.Name})
		if getErr != nil {
			return getErr
		}
		req.Config = cur.Config // Type + URL + InsecureSkipVerify; Secret stays empty so it is preserved
		req.Subscription = cur.Subscription
		req.Disabled = cur.Disabled

		if set["type"] {
			req.Config.Type = typ
		}
		if set["url"] {
			req.Config.URL = url
		}
		if set["secret"] {
			req.Config.Secret = secret
		}
		if set["insecure-tls"] {
			req.Config.InsecureSkipVerify = insecureTLS
		}
		if set["pull-ttl"] {
			req.Config.PullTTLSeconds = pullTTL
		}
		if len(events) > 0 {
			req.Subscription.Events = []string(events)
		}
		if len(outcomes) > 0 {
			req.Subscription.Outcomes = []string(outcomes)
		}
		if set["disabled"] {
			req.Disabled = disabled
		}
		resp, err = s.Update(context.Background(), &req)

	case "delete":
		var req api.NotificationDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "channel name")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)

	case "test":
		var req api.NotificationTest
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "channel name")
		f.Parse(args[1:])
		resp, err = s.Test(context.Background(), &req)

	case "deliveries":
		var (
			req    api.NotificationDeliveries
			after  time.Time
			before time.Time
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "channel name")
		f.IntVar(&req.Limit, "limit", 0, "max delivery entries (default 50, max 100)")
		f.Var(timeFlag{&after}, "after", "only entries after this time (RFC3339 or YYYY-MM-DD)")
		f.Var(timeFlag{&before}, "before", "only entries before this time (RFC3339 or YYYY-MM-DD)")
		f.Parse(args[1:])
		req.After = after
		req.Before = before
		resp, err = s.Deliveries(context.Background(), &req)

	case "pull":
		// Consume a pull channel's change events. The server stores the cursor;
		// pass -ack <cursor> (from a previous pull) to acknowledge that batch and
		// advance. -follow keeps pulling, auto-acking each batch, until interrupted.
		var (
			req      api.NotificationPull
			follow   bool
			interval time.Duration
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "channel name")
		f.Int64Var(&req.Ack, "ack", 0, "cursor from a previous pull to acknowledge as handled (advances past it)")
		f.IntVar(&req.Limit, "limit", 0, "max events per batch (default 100, max 1000)")
		f.BoolVar(&follow, "follow", false, "keep pulling, auto-acking each batch, until interrupted")
		f.DurationVar(&interval, "interval", 2*time.Second, "poll interval between empty batches when following")
		f.Parse(args[1:])
		if follow {
			return rn.followNotificationPull(s, &req, interval)
		}
		resp, err = s.Pull(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

// followNotificationPull streams a pull channel: pull a batch, print it, then ack
// it (by passing its cursor as the next request's Ack) so the next pull advances.
// It blocks until an error or process interrupt; an empty batch waits one interval
// before retrying. Because each batch is acked only on the next pull, an interrupt
// mid-batch redelivers it on the next run (at-least-once).
func (rn Runner) followNotificationPull(s api.Notification, req *api.NotificationPull, interval time.Duration) error {
	for {
		res, err := s.Pull(context.Background(), req)
		if err != nil {
			return err
		}
		if len(res.Events) > 0 {
			if err := rn.print(res); err != nil {
				return err
			}
		}
		req.Ack = res.Cursor
		if !res.HasMore {
			time.Sleep(interval)
		}
	}
}
