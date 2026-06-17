package runner

import (
	"context"
	"fmt"

	"github.com/deploys-app/api"
)

func (rn Runner) auditLog(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("auditlog")
	}

	s := rn.API.AuditLog()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("auditlog", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("auditlog", args[0])
	case "list":
		var (
			req     api.AuditLogList
			outcome string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.ResourceType, "resource-type", "", "filter by resource type")
		f.StringVar(&req.Actor, "actor", "", "filter by actor email")
		f.StringVar(&outcome, "outcome", "", "filter by outcome (success, failure)")
		f.Var(timeFlag{&req.After}, "after", "only entries after this time (RFC 3339 or YYYY-MM-DD)")
		f.Var(timeFlag{&req.Before}, "before", "only entries before this time (RFC 3339 or YYYY-MM-DD)")
		f.IntVar(&req.Limit, "limit", 0, "max entries")
		f.Parse(args[1:])

		switch outcome {
		case "":
		case api.AuditOutcomeSuccess.String():
			req.Outcome = api.AuditOutcomeSuccess
		case api.AuditOutcomeFailure.String():
			req.Outcome = api.AuditOutcomeFailure
		default:
			return fmt.Errorf("invalid outcome: '%s'", outcome)
		}
		resp, err = s.List(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}
