package runner

import (
	"context"
	"strings"

	"github.com/deploys-app/api"
)

const (
	alertOpFlagHelp     = "comparison operator: '>=', '<=', gte, or lte (default '>='; quote '>=' / '<=' in shells)"
	alertKindFlagHelp   = "target kind: deployment or custom (empty = deployment)"
	alertMetricFlagHelp = "metric to watch: cpu, memory, requests, or egress (kind=deployment); value or rate (kind=custom)"
	alertSourceFlagHelp = "metric source name (kind=custom)"
	alertSeriesFlagHelp = "series key (kind=custom)"
)

func (rn Runner) alert(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("alert")
	}

	s := rn.API.Alert()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("alert", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("alert", args[0])

	case "list":
		var req api.AlertList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)

	case "get":
		var req api.AlertGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "alert rule name")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)

	case "create":
		var req api.AlertCreate
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "alert rule name")
		f.StringVar(&req.Target.Kind, "kind", "", alertKindFlagHelp)
		f.StringVar(&req.Target.Location, "location", "", "target location (kind=deployment)")
		f.StringVar(&req.Target.Deployment, "deployment", "", "target deployment name (kind=deployment)")
		f.StringVar(&req.Target.Source, "source", "", alertSourceFlagHelp)
		f.StringVar(&req.Target.Series, "series", "", alertSeriesFlagHelp)
		f.StringVar(&req.Condition.Metric, "metric", "", alertMetricFlagHelp)
		f.StringVar(&req.Condition.Op, "op", "", alertOpFlagHelp)
		f.Float64Var(&req.Condition.Threshold, "threshold", 0, "threshold value (percent 0-100 for cpu/memory, req/min for requests, bytes/min for egress, gauge for value, per-minute increase for rate)")
		f.IntVar(&req.Condition.ForMinutes, "for", 0, "minutes the condition must hold continuously (1-60)")
		f.IntVar(&req.RenotifyMinutes, "renotify", 0, "minutes between re-notifications while still firing (0 = disabled)")
		f.BoolVar(&req.Disabled, "disabled", false, "create the rule disabled")
		f.Parse(args[1:])
		req.Condition.Op = normalizeAlertOp(req.Condition.Op)
		applyAlertTargetKind(&req.Target)
		resp, err = s.Create(context.Background(), &req)

	case "update":
		// Merge semantics: seed from the existing rule, override only the flags
		// the user explicitly passed (visitedFlags). Update is otherwise a full
		// replace (see AlertUpdate's doc comment), so every field must be seeded
		// before applying overrides.
		var (
			req        api.AlertUpdate
			kind       string
			location   string
			deployment string
			source     string
			series     string
			metric     string
			op         string
			threshold  float64
			forMin     int
			renotify   int
			disabled   bool
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "alert rule name")
		f.StringVar(&kind, "kind", "", alertKindFlagHelp)
		f.StringVar(&location, "location", "", "target location (kind=deployment)")
		f.StringVar(&deployment, "deployment", "", "target deployment name (kind=deployment)")
		f.StringVar(&source, "source", "", alertSourceFlagHelp)
		f.StringVar(&series, "series", "", alertSeriesFlagHelp)
		f.StringVar(&metric, "metric", "", alertMetricFlagHelp)
		f.StringVar(&op, "op", "", alertOpFlagHelp)
		f.Float64Var(&threshold, "threshold", 0, "threshold value (percent 0-100 for cpu/memory, req/min for requests, bytes/min for egress, gauge for value, per-minute increase for rate)")
		f.IntVar(&forMin, "for", 0, "minutes the condition must hold continuously (1-60)")
		f.IntVar(&renotify, "renotify", 0, "minutes between re-notifications while still firing (0 = disabled)")
		f.BoolVar(&disabled, "disabled", false, "disable the rule")
		f.Parse(args[1:])
		set := visitedFlags(f)

		// A distinct name avoids shadowing the outer err so a later Update error
		// still surfaces after the switch.
		cur, getErr := s.Get(context.Background(), &api.AlertGet{Project: req.Project, Name: req.Name})
		if getErr != nil {
			return getErr
		}
		req.Target = cur.Target
		req.Condition = cur.Condition
		req.RenotifyMinutes = cur.RenotifyMinutes
		req.Disabled = cur.Disabled

		if set["kind"] {
			req.Target.Kind = kind
		}
		if set["location"] {
			req.Target.Location = location
		}
		if set["deployment"] {
			req.Target.Deployment = deployment
		}
		if set["source"] {
			req.Target.Source = source
		}
		if set["series"] {
			req.Target.Series = series
		}
		if set["metric"] {
			req.Condition.Metric = metric
		}
		if set["op"] {
			req.Condition.Op = normalizeAlertOp(op)
		}
		if set["threshold"] {
			req.Condition.Threshold = threshold
		}
		if set["for"] {
			req.Condition.ForMinutes = forMin
		}
		if set["renotify"] {
			req.RenotifyMinutes = renotify
		}
		if set["disabled"] {
			req.Disabled = disabled
		}
		applyAlertTargetKind(&req.Target)
		resp, err = s.Update(context.Background(), &req)

	case "delete":
		var req api.AlertDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "alert rule name")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)

	case "events":
		var req api.AlertEvents
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "alert rule name")
		f.IntVar(&req.Limit, "limit", 0, "max event entries (default 50, max 100)")
		f.Parse(args[1:])
		resp, err = s.Events(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

// applyAlertTargetKind enforces the API shape after flags are applied: kind=custom
// requires Location/Deployment empty; kind=deployment (or empty) requires Source/
// Series empty. Empty Kind is left empty (the API treats it as deployment).
func applyAlertTargetKind(t *api.AlertTarget) {
	if t.Kind == api.AlertTargetKindCustom {
		t.Location = ""
		t.Deployment = ""
		return
	}
	t.Source = ""
	t.Series = ""
}

// normalizeAlertOp maps CLI-friendly aliases to the API operators.
// Shells treat unquoted >= and <= as redirections, so gte/lte are accepted too.
func normalizeAlertOp(op string) string {
	op = strings.TrimSpace(op)
	switch strings.ToLower(op) {
	case "", "gte", ">=":
		return api.AlertOpGTE
	case "lte", "<=":
		return api.AlertOpLTE
	default:
		return op
	}
}
