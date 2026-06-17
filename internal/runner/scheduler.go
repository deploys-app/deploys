package runner

import (
	"context"
	"time"

	"github.com/deploys-app/api"
)

func (rn Runner) scheduler(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("scheduler")
	}

	s := rn.API.Scheduler()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("scheduler", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("scheduler", args[0])

	case "list":
		var req api.SchedulerList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)

	case "get":
		var req api.SchedulerGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "scheduler job name")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)

	case "create":
		var (
			req      api.SchedulerCreate
			header   multiFlag
			authType string
			authUser string
			authPass string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "scheduler job name")
		f.StringVar(&req.Schedule, "schedule", "", "cron schedule (5-field or @descriptor, e.g. \"*/5 * * * *\")")
		f.StringVar(&req.Timezone, "timezone", "", "IANA timezone for the schedule (default UTC)")
		f.StringVar(&req.Method, "method", "GET", "HTTP method")
		f.StringVar(&req.URL, "url", "", "target URL (http/https)")
		f.Var(&header, "header", "HTTP header KEY=VALUE (repeatable)")
		f.StringVar(&req.Body, "body", "", "request body")
		f.StringVar(&authType, "auth-type", "", "auth type: none|basic|bearer")
		f.StringVar(&authUser, "auth-user", "", "basic auth username")
		f.StringVar(&authPass, "auth-secret", "", "basic auth password or bearer token")
		f.BoolVar(&req.InsecureSkipVerify, "insecure-tls", false, "skip TLS verification for HTTPS targets")
		f.BoolVar(&req.Paused, "paused", false, "create the job paused")
		f.Parse(args[1:])
		req.Headers, err = parseKV(header)
		if err != nil {
			return err
		}
		if authType != "" {
			req.Auth = api.SchedulerAuth{Type: authType, Username: authUser, Secret: authPass}
		}
		resp, err = s.Create(context.Background(), &req)

	case "update":
		// Merge semantics: seed from the existing job, override only the flags
		// the user explicitly passed (visitedFlags). The auth secret is never
		// returned by Get, so an omitted -auth-secret leaves it empty and the
		// server keeps the stored one.
		var (
			req         api.SchedulerUpdate
			header      multiFlag
			schedule    string
			timezone    string
			method      string
			url         string
			body        string
			authType    string
			authUser    string
			authPass    string
			insecureTLS bool
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "scheduler job name")
		f.StringVar(&schedule, "schedule", "", "cron schedule (5-field or @descriptor)")
		f.StringVar(&timezone, "timezone", "", "IANA timezone for the schedule")
		f.StringVar(&method, "method", "", "HTTP method")
		f.StringVar(&url, "url", "", "target URL (http/https)")
		f.Var(&header, "header", "HTTP header KEY=VALUE (repeatable; replaces all headers)")
		f.StringVar(&body, "body", "", "request body")
		f.StringVar(&authType, "auth-type", "", "auth type: none|basic|bearer")
		f.StringVar(&authUser, "auth-user", "", "basic auth username")
		f.StringVar(&authPass, "auth-secret", "", "basic auth password or bearer token (omit to keep existing)")
		f.BoolVar(&insecureTLS, "insecure-tls", false, "skip TLS verification for HTTPS targets")
		f.Parse(args[1:])
		set := visitedFlags(f)

		cur, err := s.Get(context.Background(), &api.SchedulerGet{Project: req.Project, Name: req.Name})
		if err != nil {
			return err
		}
		req.Schedule = cur.Schedule
		req.Timezone = cur.Timezone
		req.Method = cur.Method
		req.URL = cur.URL
		req.Headers = cur.Headers
		req.Body = cur.Body
		req.Auth = cur.Auth // Type + Username; Secret stays empty so it is preserved
		req.InsecureSkipVerify = cur.InsecureSkipVerify

		if set["schedule"] {
			req.Schedule = schedule
		}
		if set["timezone"] {
			req.Timezone = timezone
		}
		if set["method"] {
			req.Method = method
		}
		if set["url"] {
			req.URL = url
		}
		if set["body"] {
			req.Body = body
		}
		if set["insecure-tls"] {
			req.InsecureSkipVerify = insecureTLS
		}
		if len(header) > 0 {
			req.Headers, err = parseKV(header)
			if err != nil {
				return err
			}
		}
		if set["auth-type"] {
			req.Auth.Type = authType
		}
		if set["auth-user"] {
			req.Auth.Username = authUser
		}
		if set["auth-secret"] {
			req.Auth.Secret = authPass
		}
		resp, err = s.Update(context.Background(), &req)

	case "delete":
		var req api.SchedulerDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "scheduler job name")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)

	case "pause":
		var req api.SchedulerPause
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "scheduler job name")
		f.Parse(args[1:])
		resp, err = s.Pause(context.Background(), &req)

	case "resume":
		var req api.SchedulerResume
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "scheduler job name")
		f.Parse(args[1:])
		resp, err = s.Resume(context.Background(), &req)

	case "trigger":
		var req api.SchedulerTrigger
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "scheduler job name")
		f.Parse(args[1:])
		resp, err = s.Trigger(context.Background(), &req)

	case "logs":
		var (
			req    api.SchedulerLogs
			after  time.Time
			before time.Time
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "scheduler job name")
		f.IntVar(&req.Limit, "limit", 0, "max log entries (default 50, max 100)")
		f.Var(timeFlag{&after}, "after", "only entries after this time (RFC3339 or YYYY-MM-DD)")
		f.Var(timeFlag{&before}, "before", "only entries before this time (RFC3339 or YYYY-MM-DD)")
		f.Parse(args[1:])
		req.After = after
		req.Before = before
		resp, err = s.Logs(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}
