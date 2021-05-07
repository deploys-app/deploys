package runner

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v2"

	"github.com/deploys-app/deploys/api"
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

func (rn Runner) print(v interface{}) error {
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
		return fmt.Errorf("invalid command")
	}

	rn.replaceShortFlag(args)

	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
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
		resp interface{}
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
		resp interface{}
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
		resp interface{}
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
		f.StringVar(&req.BillingAccount, "billingaccount", "", "billing account id")
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
			billingAccount string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&name, "name", "", "project name")
		f.StringVar(&billingAccount, "billingaccount", "", "billing account id")
		f.Parse(args[1:])

		if name != "" {
			req.Name = &name
		}
		if billingAccount != "" {
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
		resp interface{}
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
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Deployment()

	var (
		resp interface{}
		err  error
	)

	f := flag.NewFlagSet("", flag.ExitOnError)
	rn.registerFlags(f)
	switch args[0] {
	default:
		return fmt.Errorf("invalid command")
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
		var (
			req  api.DeploymentDeploy
			typ  string
			port int
		)
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "deployment name")
		f.StringVar(&req.Image, "image", "", "docker image")
		f.StringVar(&typ, "type", "", "deployment type")
		f.IntVar(&port, "port", 0, "port")
		f.Parse(args[1:])
		req.Type = api.ParseDeploymentTypeString(typ)
		if port > 0 {
			req.Port = &port
		}
		resp, err = s.Deploy(context.Background(), &req)
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
		resp interface{}
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
		var req api.RouteCreate
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Location, "location", "", "location")
		f.StringVar(&req.Domain, "domain", "", "domain")
		f.StringVar(&req.Path, "path", "", "path")
		f.StringVar(&req.Deployment, "deployment", "", "deployment name")
		f.Parse(args[1:])
		resp, err = s.Create(context.Background(), &req)
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

func (rn Runner) deploymentSet(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	s := rn.API.Deployment()

	var (
		resp interface{}
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
		resp interface{}
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
		resp interface{}
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
		f.StringVar(&req.Value, "value", "", "value")
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
		resp interface{}
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
		resp interface{}
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
