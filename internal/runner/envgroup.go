package runner

import (
	"context"

	"github.com/deploys-app/api"
)

func (rn Runner) envGroup(args ...string) error {
	if len(args) == 0 || IsHelpArg(args[0]) {
		return rn.groupUsage("envgroup")
	}

	s := rn.API.EnvGroup()

	var (
		resp any
		err  error
	)

	f := rn.subFlagSet("envgroup", args[0])
	switch args[0] {
	default:
		return rn.unknownSub("envgroup", args[0])
	case "create":
		var (
			req api.EnvGroupCreate
			env multiFlag
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "env group name")
		f.Var(&env, "env", "env KEY=VALUE (repeatable)")
		f.Parse(args[1:])
		req.Env, err = parseKV(env)
		if err != nil {
			return err
		}
		resp, err = s.Create(context.Background(), &req)
	case "get":
		var req api.EnvGroupGet
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "env group name")
		f.Parse(args[1:])
		resp, err = s.Get(context.Background(), &req)
	case "list":
		var req api.EnvGroupList
		f.StringVar(&req.Project, "project", "", "project id")
		f.Parse(args[1:])
		resp, err = s.List(context.Background(), &req)
	case "update":
		var (
			req       api.EnvGroupUpdate
			env       multiFlag
			addEnv    multiFlag
			removeEnv string
		)
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "env group name")
		f.Var(&env, "env", "env KEY=VALUE, replaces all existing env (repeatable)")
		f.Var(&addEnv, "add-env", "env KEY=VALUE to add to existing env (repeatable)")
		f.StringVar(&removeEnv, "remove-env", "", "env keys to remove (comma separated values)")
		f.Parse(args[1:])
		req.Env, err = parseKV(env)
		if err != nil {
			return err
		}
		req.AddEnv, err = parseKV(addEnv)
		if err != nil {
			return err
		}
		req.RemoveEnv = splitComma(removeEnv)
		resp, err = s.Update(context.Background(), &req)
	case "delete":
		var req api.EnvGroupDelete
		f.StringVar(&req.Project, "project", "", "project id")
		f.StringVar(&req.Name, "name", "", "env group name")
		f.Parse(args[1:])
		resp, err = s.Delete(context.Background(), &req)
	}
	if err != nil {
		return err
	}
	return rn.print(resp)
}
