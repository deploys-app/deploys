package runner

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/deploys-app/api"
	"gopkg.in/yaml.v2"

	"github.com/robfig/cron/v3"
)

type tablePrinter interface {
	Table() [][]string
}

type Runner struct {
	API        api.Interface
	Output     *os.File
	OutputMode string
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
		rn.printTable(v.(tablePrinter).Table())
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
	if len(args) == 0 {
		return fmt.Errorf("invalid command: (empty args)")
	}

	rn.replaceShortFlag(args)

	switch args[0] {
	default:
		return fmt.Errorf("invalid command: '%s'", args[0])
	case "me":
		return rn.me(args[1:]...)
	case "location":
		return rn.location(args[1:]...)
	case "project":
		return rn.project(args[1:]...)
	case "role":
		return rn.role(args[1:]...)
	case "deployment", "deploy", "d":
		return rn.deployment(args[1:]...)
	case "route":
		return rn.route(args[1:]...)
	case "disk":
		return rn.disk(args[1:]...)
	case "pullsecret", "ps":
		return rn.pullSecret(args[1:]...)
	case "workloadidentity", "wi":
		return rn.workloadIdentity(args[1:]...)
	case "serviceaccount", "sa":
		return rn.serviceAccount(args[1:]...)
	case "collector":
		return rn.collector(args[1:]...)
	}
}

func (rn Runner) me(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Me()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
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
		req.Permissions = strings.Split(permissions, ",")
		resp, err = s.Authorized(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) location(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Location()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
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
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Project()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
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
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Role()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
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
		req.Permissions = strings.Split(permissions, ",")
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
		req.Roles = strings.Split(roles, ",")
		resp, err = s.Bind(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) deployment(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid deployment command: (empty args)")
	}

	s := rn.API.Deployment()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid deployment command: '%s'", args[0])
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
	case "deploy":
		return rn.deploymentDeploy(args[1:]...)
	case "set":
		return rn.deploymentSet(args[1:]...)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) route(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Route()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
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
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Deployment()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)

	var (
		req          api.DeploymentDeploy
		typ          string
		port         int
		minReplicas  int
		maxReplicas  int
		protoStr     string
		internal     bool
		envStr       string
		addEnvStr    string
		rmEnvStr     string
		cmdStr       string
		argsStr      string
		wi           string
		pull         string
		diskName     string
		diskMount    string
		diskSub      string
		schedule     string
		reqCPU       string
		reqMem       string
		limCPU       string
		limMem       string
		mountDataStr string
		sidecarsFile string
	)
	f.StringVar(&req.Location, "location", "", "location")
	f.StringVar(&req.Project, "project", "", "project id")
	f.StringVar(&req.Name, "name", "", "deployment name")
	f.StringVar(&req.Image, "image", "", "docker image")
	f.StringVar(&typ, "type", "", "deployment type (WebService,Worker,CronJob,TCPService,InternalTCPService)")
	f.IntVar(&port, "port", 0, "port")
	f.StringVar(&protoStr, "protocol", "", "protocol (http,https,h2c) [WebService]")
	f.BoolVar(&internal, "internal", false, "run as internal service [WebService]")
	f.IntVar(&minReplicas, "minReplicas", 0, "autoscale min replicas")
	f.IntVar(&maxReplicas, "maxReplicas", 0, "autoscale max replicas")
	f.StringVar(&envStr, "env", "", "env map: KEY=VAL,KEY2=VAL2 (overrides all env)")
	f.StringVar(&addEnvStr, "addEnv", "", "add env: KEY=VAL,KEY2=VAL2")
	f.StringVar(&rmEnvStr, "removeEnv", "", "remove env keys: KEY,KEY2")
	f.StringVar(&cmdStr, "command", "", "container command list: /bin/app,--flag")
	f.StringVar(&argsStr, "args", "", "container args list: --a,--b=1")
	f.StringVar(&wi, "workloadIdentity", "", "workload identity name")
	f.StringVar(&pull, "pullSecret", "", "pull secret name")
	f.StringVar(&diskName, "disk.name", "", "disk name")
	f.StringVar(&diskMount, "disk.mountPath", "", "disk mount path")
	f.StringVar(&diskSub, "disk.subPath", "", "disk sub path")
	f.StringVar(&schedule, "schedule", "", "cron schedule (CronJob)")
	f.StringVar(&reqCPU, "resources.requests.cpu", "", "CPU requests")
	f.StringVar(&reqMem, "resources.requests.memory", "", "Memory requests")
	f.StringVar(&limCPU, "resources.limits.cpu", "", "CPU limit")
	f.StringVar(&limMem, "resources.limits.memory", "", "Memory limit")
	f.StringVar(&mountDataStr, "mountData", "", "mount data: /path=VAL,/config=@file")
	f.StringVar(&sidecarsFile, "sidecarsFile", "", "path to YAML/JSON file for sidecars array")
	f.Parse(args[1:])

	// Validate deployment type
	if typ != "" {
		validTypes := map[string]struct{}{
			"WebService":         {},
			"Worker":             {},
			"CronJob":            {},
			"TCPService":         {},
			"InternalTCPService": {},
		}
		if _, ok := validTypes[typ]; !ok {
			return fmt.Errorf("invalid deployment type: %s", typ)
		}
	}
	req.Type = api.ParseDeploymentTypeString(typ)
	if port > 0 {
		req.Port = &port
	}

	// Validate protocol
	if protoStr != "" {
		validProtocols := map[string]struct{}{
			"http":  {},
			"https": {},
			"h2c":   {},
		}
		if _, ok := validProtocols[protoStr]; !ok {
			return fmt.Errorf("invalid protocol: %s", protoStr)
		}
	}
	if protoStr != "" {
		p := api.DeploymentProtocol(protoStr)
		req.Protocol = &p
	}
	if internal {
		req.Internal = &internal
	}
	if minReplicas > 0 {
		req.MinReplicas = &minReplicas
	}
	if maxReplicas > 0 {
		req.MaxReplicas = &maxReplicas
	}
	if envStr != "" {
		req.Env = parseKVList(envStr)
	}
	if addEnvStr != "" {
		req.AddEnv = parseKVList(addEnvStr)
	}
	if rmEnvStr != "" {
		req.RemoveEnv = splitCommaList(rmEnvStr)
	}
	if cmdStr != "" {
		req.Command = splitCommaList(cmdStr)
	}
	if argsStr != "" {
		req.Args = splitCommaList(argsStr)
	}
	if wi != "" {
		req.WorkloadIdentity = &wi
	}
	if pull != "" {
		req.PullSecret = &pull
	}
	if diskName != "" || diskMount != "" || diskSub != "" {
		req.Disk = &api.DeploymentDisk{
			Name:      diskName,
			MountPath: diskMount,
			SubPath:   diskSub,
		}
	}

	// Validate cron format if schedule is set
	if schedule != "" {
		_, err := cron.ParseStandard(schedule)
		if err != nil {
			return fmt.Errorf("invalid cron schedule format: %v", err)
		}
		req.Schedule = &schedule
	}

	if reqCPU != "" || reqMem != "" || limCPU != "" || limMem != "" {
		req.Resources = &api.DeploymentResource{
			Requests: api.ResourceItem{CPU: reqCPU, Memory: reqMem},
			Limits:   api.ResourceItem{CPU: limCPU, Memory: limMem},
		}
	}

	if mountDataStr != "" {
		md, err := parseMountData(mountDataStr)
		if err != nil {
			return err
		}
		req.MountData = md
	}

	if sidecarsFile != "" {
		b, err := os.ReadFile(sidecarsFile)
		if err != nil {
			return err
		}
		var sc []*api.Sidecar
		if strings.HasSuffix(sidecarsFile, ".yaml") || strings.HasSuffix(sidecarsFile, ".yml") {
			if err := yaml.Unmarshal(b, &sc); err != nil {
				return fmt.Errorf("parse sidecars yaml: %w", err)
			}
		} else {
			if err := json.Unmarshal(b, &sc); err != nil {
				return fmt.Errorf("parse sidecars json: %w", err)
			}
		}
		req.Sidecars = sc
	}
	resp, err = s.Deploy(context.Background(), &req)
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) deploymentSet(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Deployment()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
	case "image":
		if len(args) == 1 {
			return fmt.Errorf("deployment name requied")
		}

		var req api.DeploymentDeploy
		req.Name = args[1]
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Image, "image", "", "deployment image")
		f.Parse(args[2:])
		resp, err = s.Deploy(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) disk(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Disk()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
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
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}

func (rn Runner) pullSecret(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.PullSecret()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
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
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.WorkloadIdentity()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
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
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.ServiceAccount()

	var (
		resp any
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
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

func (rn Runner) collector(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	return nil
}
