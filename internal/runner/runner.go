package runner

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
	"unicode/utf8"

	"github.com/deploys-app/api"
	"gopkg.in/yaml.v2"
)

type tablePrinter interface {
	Table() [][]string
}

type Runner struct {
	API        api.Interface
	Output     *os.File
	OutputMode string
	// Version is this binary's version, shown by check-update. Empty for an
	// unknown/local build (reported as "dev").
	Version string
}

func (rn Runner) output() *os.File {
	if rn.Output == nil {
		return os.Stdout
	}
	return rn.Output
}

func (rn Runner) print(v any) error {
	switch rn.OutputMode {
	case "", "table":
		tp, ok := v.(tablePrinter)
		if !ok {
			// no table representation, fall back to yaml
			return yaml.NewEncoder(rn.output()).Encode(v)
		}
		rn.printTable(tp.Table())
		return nil
	case "yaml":
		return yaml.NewEncoder(rn.output()).Encode(v)
	case "json":
		enc := json.NewEncoder(rn.output())
		enc.SetIndent("", "    ")
		return enc.Encode(v)
	default:
		return fmt.Errorf("invalid output")
	}
}

func (rn Runner) printTable(table [][]string) {
	// find longest for each column
	ll := make([]int, len(table[0]))
	for i := range table {
		for l := range ll {
			c := utf8.RuneCountInString(table[i][l]) + 3
			if c > ll[l] {
				ll[l] = c
			}
		}
	}

	output := rn.output()
	for _, rows := range table {
		for l, cell := range rows {
			fmt.Fprintf(output, fmt.Sprintf("%%-%ds", ll[l]), cell)
		}
		fmt.Fprintln(output)
	}
}

func (rn *Runner) registerFlags(f *flag.FlagSet) {
	f.StringVar(&rn.OutputMode, "output", "table", "output mode: table, yaml, json")
}

func (rn *Runner) replaceShortFlag(args []string) {
	for i := range args {
		switch args[i] {
		case "-oyaml":
			args[i] = "--output=yaml"
		case "-ojson":
			args[i] = "--output=json"
		case "-otable":
			args[i] = "--output=table"
		}
	}
}

func (rn Runner) Run(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.topUsage()
	}

	rn.replaceShortFlag(args)

	switch args[0] {
	default:
		return fmt.Errorf("invalid command: %q (run \"deploys help\")", args[0])
	case "me":
		return rn.me(args[1:]...)
	case "billing":
		return rn.billing(args[1:]...)
	case "location":
		return rn.location(args[1:]...)
	case "project":
		return rn.project(args[1:]...)
	case "role":
		return rn.role(args[1:]...)
	case "deployment", "deploy", "d":
		return rn.deployment(args[1:]...)
	case "error", "errors":
		return rn.errorGroup(args[1:]...)
	case "domain":
		return rn.domain(args[1:]...)
	case "route":
		return rn.route(args[1:]...)
	case "waf":
		return rn.waf(args[1:]...)
	case "cache":
		return rn.cache(args[1:]...)
	case "disk":
		return rn.disk(args[1:]...)
	case "pullsecret", "ps":
		return rn.pullSecret(args[1:]...)
	case "workloadidentity", "wi":
		return rn.workloadIdentity(args[1:]...)
	case "serviceaccount", "sa":
		return rn.serviceAccount(args[1:]...)
	case "email":
		return rn.email(args[1:]...)
	case "registry":
		return rn.registry(args[1:]...)
	case "envgroup", "eg":
		return rn.envGroup(args[1:]...)
	case "auditlog":
		return rn.auditLog(args[1:]...)
	case "dropbox":
		return rn.dropbox(args[1:]...)
	case "github":
		return rn.github(args[1:]...)
	case "collector":
		return rn.collector(args[1:]...)
	case "site":
		return rn.site(args[1:]...)
	case "scheduler":
		return rn.scheduler(args[1:]...)
	case "notification":
		return rn.notification(args[1:]...)
	case "check-update":
		return rn.checkUpdate(args[1:]...)
	case "version":
		return rn.version(args[1:]...)
	}
}

func (rn Runner) me(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("me")
	}

	s := rn.API.Me()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("me", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("me", args[0])
	case "get":
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &api.Empty{})
	case "authorized":
		var (
			req         api.MeAuthorized
			permissions string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&permissions, "permissions", "", "permissions (comma separated values)")
		f.Parse(args[1:])
		req.Permissions = splitComma(permissions)
		resp, err = s.Authorized(context.Background(), &req)
	case "permissions":
		var req api.MePermissions
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.Permissions(context.Background(), &req)
	case "generate-token", "generateToken":
		var (
			req         api.MeGenerateToken
			permissions string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&permissions, "permissions", "", "permissions (comma separated; any you hold except wildcards and role.*/serviceaccount.key.*/billing.*/pullsecret.get)")
		f.IntVar(&req.TTLSeconds, "ttl", 0, "token lifetime in seconds (60-3600, default 900)")
		f.StringVar(&req.Label, "label", "", "optional attribution label for the agent session (e.g. claude-code:pr-42)")
		f.Parse(args[1:])
		req.Permissions = splitComma(permissions)
		resp, err = s.GenerateToken(context.Background(), &req)
	case "list-tokens", "listTokens":
		var req api.MeListTokens
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.ListTokens(context.Background(), &req)
	case "revoke-token", "revokeToken":
		var req api.MeRevokeToken
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.ID, "id", "", "scoped token id (from list-tokens)")
		f.Parse(args[1:])
		resp, err = s.RevokeToken(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) location(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("location")
	}

	s := rn.API.Location()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("location", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("location", args[0])
	case "list":
		var req api.LocationList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "get":
		var req api.LocationGet
		f.StringVar(&req.ID, "id", "", "location id")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) project(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("project")
	}

	s := rn.API.Project()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("project", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("project", args[0])
	case "create":
		var req api.ProjectCreate
		f.StringVar(&req.SID, "id", "", "project id")
		f.StringVar(&req.Name, "name", "", "project name")
		f.Int64Var(&req.BillingAccount, "billingaccount", 0, "billing account id")
		f.Parse(args[1:])
		resp, err = s.Create(context.Background(), &req)
	case "list":
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &api.Empty{})
	case "get":
		var req api.ProjectGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "update":
		var (
			req            api.ProjectUpdate
			name           string
			billingAccount int64
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&name, "name", "", "project name")
		f.Int64Var(&billingAccount, "billingaccount", 0, "billing account id")
		f.Parse(args[1:])

		if name != "" {
			req.Name = &name
		}
		if billingAccount > 0 {
			req.BillingAccount = &billingAccount
		}

		resp, err = s.Update(context.Background(), &req)
	case "delete":
		var req api.ProjectDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	case "usage":
		var req api.ProjectUsage
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.Usage(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) role(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("role")
	}

	s := rn.API.Role()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("role", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("role", args[0])
	case "create":
		var (
			req         api.RoleCreate
			permissions string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Role, "role", "", "role id")
		f.StringVar(&req.Name, "name", "", "role name")
		f.StringVar(&permissions, "permissions", "", "permissions")
		f.Parse(args[1:])
		req.Permissions = splitComma(permissions)
		resp, err = s.Create(context.Background(), &req)
	case "list":
		var req api.RoleList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "get":
		var req api.RoleGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Role, "role", "", "role id")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "delete":
		var req api.RoleDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Role, "role", "", "role id")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	case "grant":
		var req api.RoleGrant
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Role, "role", "", "role id")
		f.StringVar(&req.Email, "email", "", "email")
		f.Parse(args[1:])
		resp, err = s.Grant(context.Background(), &req)
	case "revoke":
		var req api.RoleRevoke
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Role, "role", "", "role id")
		f.StringVar(&req.Email, "email", "", "email")
		f.Parse(args[1:])
		resp, err = s.Revoke(context.Background(), &req)
	case "users":
		var req api.RoleUsers
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.Users(context.Background(), &req)
	case "bind":
		var (
			req   api.RoleBind
			roles string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Email, "email", "", "email")
		f.StringVar(&roles, "roles", "", "roles")
		f.Parse(args[1:])
		req.Roles = splitComma(roles)
		resp, err = s.Bind(context.Background(), &req)
	case "permissions":
		f.Parse(args[1:])
		resp, err = s.Permissions(context.Background(), &api.Empty{})
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) deployment(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("deployment")
	}

	// deploy and set own their flag handling (including -h); the shared
	// subFlagSet below is only for the simple lifecycle subcommands. Error issues
	// now live in their own top-level `error` group (backed by the api `error.*`
	// resource), no longer under `deployment errors`.
	switch args[0] {
	case "deploy":
		return rn.deploymentDeploy(args[1:]...)
	case "set":
		return rn.deploymentSet(args[1:]...)
	}

	s := rn.API.Deployment()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("deployment", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("deployment", args[0])
	case "list":
		var req api.DeploymentList
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "get":
		var req api.DeploymentGet
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.IntVar(&req.Revision, "revision", 0, "deployment revision")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "delete":
		var req api.DeploymentDelete
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	case "revisions":
		var req api.DeploymentRevisions
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.Parse(args[1:])
		resp, err = s.Revisions(context.Background(), &req)
	case "pause":
		var req api.DeploymentPause
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.Parse(args[1:])
		resp, err = s.Pause(context.Background(), &req)
	case "resume":
		var req api.DeploymentResume
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.Parse(args[1:])
		resp, err = s.Resume(context.Background(), &req)
	case "restart":
		var req api.DeploymentRestart
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.Parse(args[1:])
		resp, err = s.Restart(context.Background(), &req)
	case "rollback":
		var req api.DeploymentRollback
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.IntVar(&req.Revision, "revision", 0, "revision to rollback to")
		f.Parse(args[1:])
		resp, err = s.Rollback(context.Background(), &req)
	case "metrics":
		var (
			req       api.DeploymentMetrics
			timeRange string
		)
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.StringVar(&timeRange, "time-range", "1h", "time range (1h, 6h, 12h, 1d)")
		f.Parse(args[1:])
		req.TimeRange = api.DeploymentMetricsTimeRange(timeRange)
		resp, err = s.Metrics(context.Background(), &req)
	case "status":
		var req api.DeploymentStatus
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.Parse(args[1:])
		resp, err = s.Status(context.Background(), &req)
	case "logs":
		var (
			req    api.DeploymentLogs
			follow bool
		)
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.StringVar(&req.Pod, "pod", "", "single pod name (default: all pods of the deployment)")
		f.BoolVar(&req.Previous, "previous", false, "read the last crashed container (crash post-mortem)")
		f.IntVar(&req.TailLines, "tail", 0, "lines per pod (default 200, max 1000)")
		f.BoolVar(&follow, "follow", false, "re-poll for new lines (client-side; the API stays snapshot-only)")
		f.Parse(args[1:])
		if follow {
			// --follow is a CLI-only convenience: re-poll the bounded snapshot on
			// an interval and print lines not seen before. The API/MCP contract
			// stays snapshot-only.
			return rn.deploymentLogsFollow(s, &req)
		}
		resp, err = s.Logs(context.Background(), &req)
	case "logsHistory", "logs-history":
		var (
			req          api.DeploymentLogsHistory
			since, until string
		)
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.StringVar(&req.Pod, "pod", "", "single pod name (default: all pods of the deployment)")
		f.StringVar(&since, "since", "", "window start, RFC3339 or relative (e.g. 24h, 1h, 30m = now minus that); required")
		f.StringVar(&until, "until", "", "window end, RFC3339 or relative (e.g. 1h); empty defaults to now")
		f.IntVar(&req.Limit, "limit", 0, "max lines per page (default 200, max 1000)")
		f.BoolVar(&req.Reverse, "reverse", false, "return newest-first and page backward into the past")
		f.StringVar(&req.Cursor, "cursor", "", "opaque page cursor from a previous response's nextCursor")
		f.Parse(args[1:])
		if req.Since, err = parseHistoryTime(since); err != nil {
			return fmt.Errorf("invalid -since: %w", err)
		}
		if req.Until, err = parseHistoryTime(until); err != nil {
			return fmt.Errorf("invalid -until: %w", err)
		}
		resp, err = s.LogsHistory(context.Background(), &req)
	case "extend-ttl", "extendTTL":
		var req api.DeploymentExtendTTL
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.Int64Var(&req.TTL, "ttl", 0, "seconds from now until auto-delete (must be > 0)")
		f.Parse(args[1:])
		resp, err = s.ExtendTTL(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

// parseHistoryTime parses a -since/-until flag value for deployment logs
// history. An empty value yields the zero time (the server resolves a zero
// Until to now; a zero Since is rejected upstream as required). A value is
// either an RFC3339 timestamp or a Go duration (e.g. 24h, 30m) interpreted as
// now minus that duration.
func parseHistoryTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected RFC3339 timestamp or duration (e.g. 24h): %q", s)
	}
	return time.Now().Add(-d), nil
}

// deploymentLogsFollow client-side tails a deployment by re-polling the bounded
// deployment.logs snapshot and printing only lines it hasn't printed yet. It
// runs until interrupted. The seen-set is soft-capped so a long session can't
// grow it without bound (a reset may reprint the current tail once).
func (rn Runner) deploymentLogsFollow(s api.Deployment, req *api.DeploymentLogs) error {
	const seenCap = 20000
	seen := map[string]bool{}
	out := rn.output()
	for {
		res, err := s.Logs(context.Background(), req)
		if err != nil {
			return err
		}
		if len(seen) > seenCap {
			seen = map[string]bool{}
		}
		for _, l := range res.Lines {
			key := l.Timestamp.Format(time.RFC3339Nano) + "\x00" + l.Pod + "\x00" + l.Log
			if seen[key] {
				continue
			}
			seen[key] = true
			fmt.Fprintf(out, "%s %s %s\n", l.Timestamp.Format(time.RFC3339), l.Pod, l.Log)
		}
		time.Sleep(2 * time.Second)
	}
}

func (rn Runner) route(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("route")
	}

	s := rn.API.Route()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("route", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("route", args[0])
	case "list":
		var req api.RouteList
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "get":
		var req api.RouteGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Domain, "domain", "", "domain")
		f.StringVar(&req.Path, "path", "", "path")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "create":
		var (
			req        api.RouteCreateV2
			deployment string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Domain, "domain", "", "domain")
		f.StringVar(&req.Path, "path", "", "path")
		f.StringVar(&req.Target, "target", "", "target (for v2)")
		f.StringVar(&deployment, "deployment", "", "deployment name (for v1)")
		f.StringVar(&req.Config.Host, "host", "", "override the Host header sent upstream (external http:// targets only)")
		f.Parse(args[1:])

		if req.Target == "" && deployment != "" {
			req.Target = "deployment://" + deployment
		}
		resp, err = s.CreateV2(context.Background(), &req)
	case "delete":
		var req api.RouteDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Domain, "domain", "", "domain")
		f.StringVar(&req.Path, "path", "", "path")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) deploymentDeploy(args ...string) error {
	// Pass the runner's output so the -h banner honors Runner.Output like every
	// other subcommand (subFlagSet's Usage targets rn.output()).
	req, outputMode, err := parseDeploymentDeploy(rn.output(), args)
	if errors.Is(err, flag.ErrHelp) {
		return nil // usage already printed; -h is a clean exit, matching ExitOnError
	}
	if err != nil {
		return err
	}
	rn.OutputMode = outputMode

	resp, err := rn.API.Deployment().Deploy(context.Background(), &req)
	if err != nil {
		return err
	}
	return rn.print(resp)
}

// parseDeploymentDeploy maps the full api.DeploymentDeploy surface to flags. It
// is pure (no API calls) so the flag→request mapping is unit-testable.
//
// The request is a merge: every optional field is left nil/empty unless its
// flag is provided, so omitting a flag preserves the previous revision's value.
// Scalar pointers use visitedFlags so an explicit zero/empty can be sent (e.g.
// -ttl 0 to clear, -internal=false), while the long-standing -port/-minReplicas/
// -maxReplicas keep their >0 semantics for backward compatibility.
//
// helpOut receives the -h/-help banner (so it can be redirected and asserted in
// tests); all other output is discarded and surfaced to the caller as an error.
func parseDeploymentDeploy(helpOut io.Writer, args []string) (api.DeploymentDeploy, string, error) {
	var (
		req         api.DeploymentDeploy
		outputMode  string
		typ         string
		port        int
		minReplicas int
		maxReplicas int

		protocol string
		internal bool

		env             multiFlag
		addEnv          multiFlag
		removeEnv       string
		envGroups       string
		addEnvGroups    string
		removeEnvGroups string

		command string
		cmdArgs string

		workloadIdentity string
		pullSecret       string
		schedule         string
		ttl              int64

		diskName      string
		diskMountPath string
		diskSubPath   string

		cpuRequest string
		memRequest string
		cpuLimit   string
		memLimit   string

		mountData          multiFlag
		requireGoogleLogin bool
		allowedEmails      string
		allowedDomains     string
		sidecarsFile       string
	)

	// ContinueOnError (not ExitOnError) so a parse error returns instead of
	// calling os.Exit — keeps the function pure and testable; the caller surfaces
	// the error. Output is discarded so the error is reported once, by main.
	f := flag.NewFlagSet("deployment deploy", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&outputMode, "output", "table", "output mode: table, yaml, json")
	f.StringVar(&req.Location, "location", "", "location")
	f.StringVar(&req.Project, "project", "", "project id")
	f.StringVar(&req.Name, "name", "", "deployment name")
	f.StringVar(&req.Image, "image", "", "docker image")
	f.StringVar(&req.Site, "site", "", "static site release ref (site://...); use with -type Static (see `deploys site publish`)")
	f.StringVar(&typ, "type", "", "deployment type (WebService, Worker, CronJob, TCPService, InternalTCPService, Static)")
	f.IntVar(&port, "port", 0, "port")
	f.IntVar(&minReplicas, "minReplicas", 0, "autoscale min replicas")
	f.IntVar(&maxReplicas, "maxReplicas", 0, "autoscale max replicas")
	f.StringVar(&protocol, "protocol", "", "WebService protocol: http, https, h2c")
	f.BoolVar(&internal, "internal", false, "run WebService as an internal service")
	f.Var(&env, "env", "env KEY=VALUE, replaces all env (repeatable)")
	f.Var(&addEnv, "addEnv", "env KEY=VALUE to add to the previous revision (repeatable)")
	f.StringVar(&removeEnv, "removeEnv", "", "env keys to remove (comma separated)")
	f.StringVar(&envGroups, "envGroups", "", "env groups, replaces all (comma separated)")
	f.StringVar(&addEnvGroups, "addEnvGroups", "", "env groups to add (comma separated)")
	f.StringVar(&removeEnvGroups, "removeEnvGroups", "", "env groups to remove (comma separated)")
	f.StringVar(&command, "command", "", "container entrypoint command (comma separated)")
	f.StringVar(&cmdArgs, "args", "", "container args (comma separated)")
	f.StringVar(&workloadIdentity, "workloadIdentity", "", "workload identity name")
	f.StringVar(&pullSecret, "pullSecret", "", "pull secret name")
	f.StringVar(&schedule, "schedule", "", "cron schedule (CronJob)")
	f.Int64Var(&ttl, "ttl", 0, "seconds until auto-delete; pass 0 to clear an existing TTL")
	f.StringVar(&diskName, "diskName", "", "disk name (Stateful)")
	f.StringVar(&diskMountPath, "diskMountPath", "", "disk mount path")
	f.StringVar(&diskSubPath, "diskSubPath", "", "disk sub path")
	f.StringVar(&cpuRequest, "cpuRequest", "", "CPU request (e.g. 250m)")
	f.StringVar(&memRequest, "memRequest", "", "memory request (e.g. 256Mi)")
	f.StringVar(&cpuLimit, "cpuLimit", "", "CPU limit (e.g. 500m)")
	f.StringVar(&memLimit, "memLimit", "", "memory limit (e.g. 512Mi)")
	f.Var(&mountData, "mountData", "mounted file PATH=VALUE (repeatable)")
	f.BoolVar(&requireGoogleLogin, "requireGoogleLogin", false, "require Google login to access the deployment")
	f.StringVar(&allowedEmails, "allowedEmails", "", "allowed emails for access (comma separated)")
	f.StringVar(&allowedDomains, "allowedDomains", "", "allowed domains for access (comma separated)")
	f.StringVar(&sidecarsFile, "sidecarsFile", "", "path to a YAML/JSON file with the sidecars list")
	if err := f.Parse(args); err != nil {
		// -h/-help: render the same banner as the other subcommands, then let
		// the caller treat it as a clean (non-error) exit. Other parse errors
		// stay quiet here (output is discarded) and are surfaced by the caller.
		if errors.Is(err, flag.ErrHelp) {
			writeSubUsage(helpOut, f, "deployment", "deploy")
		}
		return req, "", err
	}

	set := visitedFlags(f)

	req.Type = api.ParseDeploymentTypeString(typ)
	if port > 0 {
		req.Port = &port
	}
	if minReplicas > 0 {
		req.MinReplicas = &minReplicas
	}
	if maxReplicas > 0 {
		req.MaxReplicas = &maxReplicas
	}
	if set["protocol"] {
		p := api.DeploymentProtocol(protocol)
		req.Protocol = &p
	}
	if set["internal"] {
		req.Internal = &internal
	}
	if set["workloadIdentity"] {
		req.WorkloadIdentity = &workloadIdentity
	}
	if set["pullSecret"] {
		req.PullSecret = &pullSecret
	}
	if set["schedule"] {
		req.Schedule = &schedule
	}
	if set["ttl"] {
		req.TTL = &ttl
	}

	var err error
	if req.Env, err = parseKV(env); err != nil {
		return req, "", err
	}
	if req.AddEnv, err = parseKV(addEnv); err != nil {
		return req, "", err
	}
	if req.MountData, err = parseKV(mountData); err != nil {
		return req, "", err
	}
	req.RemoveEnv = splitComma(removeEnv)
	req.EnvGroups = splitComma(envGroups)
	req.AddEnvGroups = splitComma(addEnvGroups)
	req.RemoveEnvGroups = splitComma(removeEnvGroups)
	req.Command = splitComma(command)
	req.Args = splitComma(cmdArgs)

	if diskName != "" {
		req.Disk = &api.DeploymentDisk{
			Name:      diskName,
			MountPath: diskMountPath,
			SubPath:   diskSubPath,
		}
	}
	if cpuRequest != "" || memRequest != "" || cpuLimit != "" || memLimit != "" {
		req.Resources = &api.DeploymentResource{
			Requests: api.ResourceItem{CPU: cpuRequest, Memory: memRequest},
			Limits:   api.ResourceItem{CPU: cpuLimit, Memory: memLimit},
		}
	}
	if requireGoogleLogin || allowedEmails != "" || allowedDomains != "" {
		req.Access = &api.DeploymentAccessConfig{
			RequireGoogleLogin: requireGoogleLogin,
			AllowedEmails:      splitComma(allowedEmails),
			AllowedDomains:     splitComma(allowedDomains),
		}
	}
	if sidecarsFile != "" {
		b, err := os.ReadFile(sidecarsFile)
		if err != nil {
			return req, "", err
		}
		if err := yaml.Unmarshal(b, &req.Sidecars); err != nil {
			return req, "", fmt.Errorf("invalid sidecars file %q: %w", sidecarsFile, err)
		}
	}

	return req, outputMode, nil
}

func (rn Runner) deploymentSet(args ...string) error {
	// set currently has a single leaf, `image`; its flag set doubles as the
	// help banner for `set`, `set -h`, and `set image -h`.
	newImageFlags := func() (*flag.FlagSet, *api.DeploymentDeploy) {
		req := &api.DeploymentDeploy{}
		f := rn.subFlagSet("deployment", "set image")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Image, "image", "", "deployment image")
		return f, req
	}

	if len(args) == 0 || IsHelpArg(args[0]) {
		f, _ := newImageFlags()
		f.Usage()
		return nil
	}

	switch args[0] {
	default:
		return rn.unknownSub("deployment set", args[0])
	case "image":
		f, req := newImageFlags()
		if len(args) < 2 || IsHelpArg(args[1]) {
			f.Usage()
			return nil
		}
		req.Name = args[1]
		f.Parse(args[2:])
		resp, err := rn.API.Deployment().Deploy(context.Background(), req)
		if err != nil {
			return err
		}
		return rn.print(resp)
	}
}

func (rn Runner) disk(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("disk")
	}

	s := rn.API.Disk()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("disk", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("disk", args[0])
	case "create":
		var req api.DiskCreate
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Name, "name", "", "disk name")
		f.Int64Var(&req.Size, "size", 1, "disk size (Gi)")
		f.Parse(args[1:])
		resp, err = s.Create(context.Background(), &req)
	case "get":
		var req api.DiskGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Name, "name", "", "disk name")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "list":
		var req api.DiskList
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "update":
		var req api.DiskUpdate
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Name, "name", "", "disk name")
		f.Int64Var(&req.Size, "size", 0, "disk size (Gi)")
		f.Parse(args[1:])
		resp, err = s.Update(context.Background(), &req)
	case "delete":
		var req api.DiskDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Name, "name", "", "disk name")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	case "metrics":
		var (
			req       api.DiskMetrics
			timeRange string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Name, "name", "", "disk name")
		f.StringVar(&timeRange, "time-range", "1h", "time range (1h, 6h, 12h, 1d, 2d, 7d, 30d)")
		f.Parse(args[1:])
		req.TimeRange = api.DiskMetricsTimeRange(timeRange)
		resp, err = s.Metrics(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) pullSecret(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("pullsecret")
	}

	s := rn.API.PullSecret()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("pullsecret", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("pullsecret", args[0])
	case "create":
		var req api.PullSecretCreate
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "name")
		f.StringVar(&req.Spec.Server, "server", "", "server")
		f.StringVar(&req.Spec.Username, "username", "", "username")
		f.StringVar(&req.Spec.Password, "password", "", "password")
		f.Parse(args[1:])
		resp, err = s.Create(context.Background(), &req)
	case "list":
		var req api.PullSecretList
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "get":
		var req api.PullSecretGet
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "name")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "delete":
		var req api.PullSecretDelete
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "name")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) workloadIdentity(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("workloadidentity")
	}

	s := rn.API.WorkloadIdentity()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("workloadidentity", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("workloadidentity", args[0])
	case "create":
		var req api.WorkloadIdentityCreate
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Name, "name", "", "workload identity name")
		f.StringVar(&req.GSA, "gsa", "", "google service account")
		f.Parse(args[1:])
		resp, err = s.Create(context.Background(), &req)
	case "get":
		var req api.WorkloadIdentityGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Name, "name", "", "workload identity name")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "list":
		var req api.WorkloadIdentityList
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "delete":
		var req api.WorkloadIdentityDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Name, "name", "", "workload identity name")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) serviceAccount(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("serviceaccount")
	}

	s := rn.API.ServiceAccount()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("serviceaccount", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("serviceaccount", args[0])
	case "create":
		var req api.ServiceAccountCreate
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.SID, "id", "", "service account id")
		f.StringVar(&req.Name, "name", "", "name")
		f.StringVar(&req.Description, "description", "", "description")
		f.Parse(args[1:])
		resp, err = s.Create(context.Background(), &req)
	case "list":
		var req api.ServiceAccountList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "get":
		var req api.ServiceAccountGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.ID, "id", "", "service account id")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "update":
		var req api.ServiceAccountUpdate
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.SID, "id", "", "service account id")
		f.StringVar(&req.Name, "name", "", "name")
		f.StringVar(&req.Description, "description", "", "description")
		f.Parse(args[1:])
		resp, err = s.Update(context.Background(), &req)
	case "delete":
		var req api.ServiceAccountDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.ID, "id", "", "service account id")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	case "createkey":
		var req api.ServiceAccountCreateKey
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.ID, "id", "", "service account id")
		f.Parse(args[1:])
		resp, err = s.CreateKey(context.Background(), &req)
	case "deletekey":
		var req api.ServiceAccountDeleteKey
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.ID, "id", "", "service account id")
		f.StringVar(&req.Secret, "secret", "", "secret")
		f.Parse(args[1:])
		resp, err = s.DeleteKey(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) github(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("github")
	}

	s := rn.API.GitHub()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("github", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("github", args[0])
	case "link":
		var (
			req        api.GitHubLink
			repository string
			trigger    string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&repository, "repository", "", "repository (owner/name)")
		f.StringVar(&req.ServiceAccount, "service-account", "", "service account id")
		f.StringVar(&req.ProductionBranch, "production-branch", "", "production branch (empty = any branch; ignored for -trigger pr)")
		f.StringVar(&trigger, "trigger", "all", "deploy trigger: all | branch | pr")
		f.Parse(args[1:])
		req.Trigger = api.ParseGitHubTriggerString(trigger)
		// Resolve owner/name to the immutable repository id through the
		// github app — this also verifies the app is installed on the repo.
		lookup, lerr := s.LookupRepo(context.Background(), &api.GitHubLookupRepo{
			Project:    req.Project,
			Repository: repository,
		})
		if lerr != nil {
			return lerr
		}
		req.RepositoryID = lookup.RepositoryID
		req.Repository = lookup.Repository
		req.InstallationID = lookup.InstallationID
		resp, err = s.Link(context.Background(), &req)
	case "unlink":
		var (
			req        api.GitHubUnlink
			repository string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&repository, "repository", "", "repository (owner/name)")
		f.Int64Var(&req.RepositoryID, "repository-id", 0, "github repository id (alternative to -repository)")
		f.Parse(args[1:])
		if req.RepositoryID == 0 && repository != "" {
			lookup, lerr := s.LookupRepo(context.Background(), &api.GitHubLookupRepo{
				Project:    req.Project,
				Repository: repository,
			})
			if lerr != nil {
				return lerr
			}
			req.RepositoryID = lookup.RepositoryID
		}
		resp, err = s.Unlink(context.Background(), &req)
	case "update":
		var (
			req        api.GitHubUpdate
			repository string
			trigger    string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&repository, "repository", "", "repository (owner/name)")
		f.Int64Var(&req.RepositoryID, "repository-id", 0, "github repository id (alternative to -repository)")
		f.StringVar(&req.ServiceAccount, "service-account", "", "service account id")
		f.StringVar(&req.ProductionBranch, "production-branch", "", "production branch (empty = any branch; ignored for -trigger pr)")
		f.StringVar(&trigger, "trigger", "", "deploy trigger: all | branch | pr")
		f.Parse(args[1:])

		if req.RepositoryID == 0 && repository != "" {
			lookup, lerr := s.LookupRepo(context.Background(), &api.GitHubLookupRepo{
				Project:    req.Project,
				Repository: repository,
			})
			if lerr != nil {
				return lerr
			}
			req.RepositoryID = lookup.RepositoryID
		}
		if req.RepositoryID == 0 {
			return fmt.Errorf("-repository or -repository-id required")
		}

		// Update is a full replace, so seed every field from the existing link
		// and override only the flags the user actually passed — omitting a flag
		// preserves its current value instead of resetting it.
		list, lerr := s.List(context.Background(), &api.GitHubList{Project: req.Project})
		if lerr != nil {
			return lerr
		}
		var cur *api.GitHubLinkItem
		for _, it := range list.Items {
			if it.RepositoryID == req.RepositoryID {
				cur = it
				break
			}
		}
		if cur == nil {
			return fmt.Errorf("github: repository link not found for repository id %d", req.RepositoryID)
		}
		set := visitedFlags(f)
		if !set["service-account"] {
			req.ServiceAccount = cur.ServiceAccount
		}
		if !set["production-branch"] {
			req.ProductionBranch = cur.ProductionBranch
		}
		if set["trigger"] {
			req.Trigger = api.ParseGitHubTriggerString(trigger)
		} else {
			req.Trigger = cur.Trigger
		}
		resp, err = s.Update(context.Background(), &req)
	case "list":
		var req api.GitHubList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) collector(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	return nil
}
