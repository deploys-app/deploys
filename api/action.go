package api

type Action int

const (
	_ Action = iota
	Create
	Delete
)

func (a Action) String() string {
	switch a {
	case Create:
		return "create"
	case Delete:
		return "delete"
	default:
		return ""
	}
}

type DeploymentAction int

const (
	_ DeploymentAction = iota
	DeploymentActionDeploy
	DeploymentActionDelete
	DeploymentActionPause
)

func (a DeploymentAction) String() string {
	switch a {
	case DeploymentActionDeploy:
		return "deploy"
	case DeploymentActionDelete:
		return "delete"
	case DeploymentActionPause:
		return "pause"
	default:
		return ""
	}
}
